// Package ip is the IP probe. For every A/AAAA found by the DNS probe
// (and anything else the scanner passes via Target.Related), it resolves
// the IP to an ASN and country code using a local MaxMind GeoLite2
// database. No network calls. If the database is missing or corrupt the
// probe fails fast at load time — never mid-scan.
package ip

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
	"github.com/oschwald/maxminddb-golang"
)

// Probe is the IP probe. It looks up each address in target and every
// address resolved from target.Related.
type Probe struct {
	asn     *maxminddb.Reader
	country *maxminddb.Reader

	// Resolver lets the probe resolve a hostname to its addresses when
	// only a domain is available (e.g. Target.Related). A nil value
	// uses net.DefaultResolver.
	Resolver *net.Resolver
}

// ID implements probe.Probe.
func (*Probe) ID() string { return "ip" }

// Open loads the GeoLite2 databases. asnPath is required; countryPath
// may be empty, in which case asnPath is used for both. Missing or
// corrupt DBs return an error so the caller can fail fast at startup.
func Open(asnPath, countryPath string) (*Probe, error) {
	if asnPath == "" {
		return nil, errors.New("ip: asn DB path is empty")
	}
	if _, err := os.Stat(asnPath); err != nil {
		return nil, fmt.Errorf("ip: asn DB: %w", err)
	}
	asn, err := maxminddb.Open(asnPath)
	if err != nil {
		return nil, fmt.Errorf("ip: open asn DB: %w", err)
	}
	country := asn
	if countryPath != "" && countryPath != asnPath {
		c, err := maxminddb.Open(countryPath)
		if err != nil {
			_ = asn.Close()
			return nil, fmt.Errorf("ip: open country DB: %w", err)
		}
		country = c
	}
	return &Probe{asn: asn, country: country}, nil
}

// Close releases both DB handles.
func (p *Probe) Close() error {
	var first error
	if p.asn != nil {
		if err := p.asn.Close(); err != nil {
			first = err
		}
	}
	if p.country != nil && p.country != p.asn {
		if err := p.country.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Lookup returns the ASN and country data for a single IP. Exposed for
// use by the http probe's third-party resolver.
type Lookup struct {
	ASN          uint   `json:"asn,omitempty"`
	Organisation string `json:"organisation,omitempty"`
	Country      string `json:"country,omitempty"`
}

func (p *Probe) Lookup(ip net.IP) (Lookup, error) {
	var out Lookup
	if p == nil || p.asn == nil {
		return out, errors.New("ip: probe not initialised")
	}
	var asnRec struct {
		AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
	}
	if err := p.asn.Lookup(ip, &asnRec); err != nil {
		return out, fmt.Errorf("ip: asn lookup: %w", err)
	}
	out.ASN = asnRec.AutonomousSystemNumber
	out.Organisation = asnRec.AutonomousSystemOrganization

	var cRec struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	if err := p.country.Lookup(ip, &cRec); err != nil {
		return out, fmt.Errorf("ip: country lookup: %w", err)
	}
	out.Country = cRec.Country.ISOCode
	return out, nil
}

// Run implements probe.Probe. It resolves the apex domain and any
// related domains, and records one Finding per (host, address) pair.
func (p *Probe) Run(ctx context.Context, target models.Target, _ probe.Config) ([]models.Finding, error) {
	if p == nil || p.asn == nil {
		return []models.Finding{{
			ProbeID:    "ip.unavailable",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"reason": "no geoip database configured"},
		}}, nil
	}
	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	hosts := append([]string{target.Domain}, target.Related...)
	var findings []models.Finding
	for _, h := range hosts {
		addrs, err := resolver.LookupHost(ctx, h)
		if err != nil {
			findings = append(findings, models.Finding{
				ProbeID:    "ip.resolve",
				Subject:    h,
				Severity:   models.SeverityInfo,
				Attributes: map[string]any{"error": err.Error()},
			})
			continue
		}
		for _, a := range addrs {
			ip := net.ParseIP(a)
			if ip == nil {
				continue
			}
			info, err := p.Lookup(ip)
			if err != nil {
				findings = append(findings, models.Finding{
					ProbeID:    "ip.lookup",
					Subject:    h,
					Severity:   models.SeverityInfo,
					Attributes: map[string]any{"address": a, "error": err.Error()},
				})
				continue
			}
			findings = append(findings, models.Finding{
				ProbeID:       "ip.asn",
				DimensionHint: models.DimensionJuridisch,
				Subject:       h,
				Severity:      models.SeverityFinding,
				Attributes: map[string]any{
					"address":      a,
					"asn":          info.ASN,
					"organisation": info.Organisation,
					"country":      info.Country,
				},
			})
		}
	}
	return findings, nil
}
