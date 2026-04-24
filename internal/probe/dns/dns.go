// Package dns is the DNS probe. It queries A/AAAA, MX, NS, CNAME, TXT,
// and CAA records for the target domain and records each result as a
// Finding.
//
// The probe holds a Resolver interface (not net.Resolver directly) so
// tests can inject a fake. The default resolver is a thin adapter around
// net.Resolver with context-scoped timeouts.
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Resolver is the minimal surface the probe needs. net.Resolver
// satisfies it via the adapter returned by NewNetResolver.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupCAA(ctx context.Context, name string) ([]CAA, error)
}

// CAA is the subset of a CAA record we record.
type CAA struct {
	Flag  uint8
	Tag   string
	Value string
}

// Probe is the DNS probe.
type Probe struct {
	Resolver Resolver
}

// New returns a DNS probe using the system resolver.
func New() *Probe {
	return &Probe{Resolver: NewNetResolver(net.DefaultResolver)}
}

// ID implements probe.Probe.
func (*Probe) ID() string { return "dns" }

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, _ probe.Config) ([]models.Finding, error) {
	if p.Resolver == nil {
		return nil, errors.New("dns: resolver not set")
	}
	domain := target.Domain
	var findings []models.Finding

	// A/AAAA via LookupHost — returns both families.
	addrs, err := p.Resolver.LookupHost(ctx, domain)
	switch {
	case err == nil:
		sort.Strings(addrs)
		for _, a := range addrs {
			kind := "a"
			if strings.Contains(a, ":") {
				kind = "aaaa"
			}
			findings = append(findings, models.Finding{
				ProbeID:       "dns." + kind,
				DimensionHint: models.DimensionNone,
				Subject:       domain,
				Severity:      models.SeverityInfo,
				Attributes:    map[string]any{"address": a},
			})
		}
		if len(addrs) == 0 {
			findings = append(findings, noAnswer(domain, "dns.a", "no A/AAAA answers"))
		}
	default:
		findings = append(findings, lookupError(domain, "dns.a", err))
	}

	// MX
	mx, err := p.Resolver.LookupMX(ctx, domain)
	if err != nil {
		findings = append(findings, lookupError(domain, "dns.mx", err))
	} else if len(mx) == 0 {
		findings = append(findings, noAnswer(domain, "dns.mx", "no MX records"))
	} else {
		sort.SliceStable(mx, func(i, j int) bool {
			if mx[i].Pref != mx[j].Pref {
				return mx[i].Pref < mx[j].Pref
			}
			return mx[i].Host < mx[j].Host
		})
		for _, m := range mx {
			findings = append(findings, models.Finding{
				ProbeID:       "dns.mx",
				DimensionHint: models.DimensionDataAI,
				Subject:       domain,
				Severity:      models.SeverityObservation,
				Attributes: map[string]any{
					"host":       m.Host,
					"preference": int(m.Pref),
				},
				Evidence: []byte(fmt.Sprintf("%d %s", m.Pref, m.Host)),
			})
		}
	}

	// NS
	ns, err := p.Resolver.LookupNS(ctx, domain)
	if err != nil {
		findings = append(findings, lookupError(domain, "dns.ns", err))
	} else {
		sort.Slice(ns, func(i, j int) bool { return ns[i].Host < ns[j].Host })
		for _, n := range ns {
			findings = append(findings, models.Finding{
				ProbeID:       "dns.ns",
				DimensionHint: models.DimensionOperationeel,
				Subject:       domain,
				Severity:      models.SeverityObservation,
				Attributes:    map[string]any{"host": n.Host},
			})
		}
	}

	// CNAME — only meaningful on non-apex lookups but we record what the
	// resolver returns for the apex so the assessor can see if anything
	// flattens.
	cname, err := p.Resolver.LookupCNAME(ctx, domain)
	if err == nil && cname != "" && strings.TrimSuffix(cname, ".") != domain {
		findings = append(findings, models.Finding{
			ProbeID:       "dns.cname",
			DimensionHint: models.DimensionDataAI,
			Subject:       domain,
			Severity:      models.SeverityObservation,
			Attributes:    map[string]any{"target": cname},
		})
	}

	// TXT — extract SPF/DKIM/DMARC hints by prefix. The raw record goes
	// into Evidence so nothing is lost.
	txt, err := p.Resolver.LookupTXT(ctx, domain)
	if err != nil {
		findings = append(findings, lookupError(domain, "dns.txt", err))
	} else {
		for _, r := range txt {
			kind := classifyTXT(r)
			findings = append(findings, models.Finding{
				ProbeID:       "dns.txt." + kind,
				DimensionHint: models.DimensionDataAI,
				Subject:       domain,
				Severity:      txtSeverity(kind),
				Attributes:    map[string]any{"record": r, "kind": kind},
				Evidence:      []byte(r),
			})
		}
	}
	// DMARC lives at _dmarc.<domain>.
	dmarc, err := p.Resolver.LookupTXT(ctx, "_dmarc."+domain)
	if err == nil {
		for _, r := range dmarc {
			findings = append(findings, models.Finding{
				ProbeID:       "dns.txt.dmarc",
				DimensionHint: models.DimensionDataAI,
				Subject:       "_dmarc." + domain,
				Severity:      models.SeverityObservation,
				Attributes:    map[string]any{"record": r, "kind": "dmarc"},
				Evidence:      []byte(r),
			})
		}
	}

	// CAA
	caa, err := p.Resolver.LookupCAA(ctx, domain)
	switch {
	case err != nil:
		findings = append(findings, lookupError(domain, "dns.caa", err))
	case len(caa) == 0:
		findings = append(findings, noAnswer(domain, "dns.caa", "no CAA records"))
	default:
		for _, c := range caa {
			findings = append(findings, models.Finding{
				ProbeID:       "dns.caa",
				DimensionHint: models.DimensionOperationeel,
				Subject:       domain,
				Severity:      models.SeverityObservation,
				Attributes: map[string]any{
					"flag":  int(c.Flag),
					"tag":   c.Tag,
					"value": c.Value,
				},
				Evidence: []byte(fmt.Sprintf("%d %s %q", c.Flag, c.Tag, c.Value)),
			})
		}
	}

	return findings, nil
}

func classifyTXT(r string) string {
	lr := strings.ToLower(r)
	switch {
	case strings.HasPrefix(lr, "v=spf1"):
		return "spf"
	case strings.HasPrefix(lr, "v=dkim1"):
		return "dkim"
	case strings.HasPrefix(lr, "v=dmarc1"):
		return "dmarc"
	default:
		return "other"
	}
}

func txtSeverity(kind string) models.Severity {
	if kind == "other" {
		return models.SeverityInfo
	}
	return models.SeverityObservation
}

func noAnswer(domain, probeID, reason string) models.Finding {
	return models.Finding{
		ProbeID:    probeID,
		Subject:    domain,
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{"no_answer": true, "reason": reason},
	}
}

func lookupError(domain, probeID string, err error) models.Finding {
	// Distinguish NXDOMAIN and timeout so the assessor can treat them
	// differently.
	kind := "error"
	if dnsErr := new(net.DNSError); errors.As(err, &dnsErr) {
		switch {
		case dnsErr.IsNotFound:
			kind = "nxdomain"
		case dnsErr.IsTimeout:
			kind = "timeout"
		case dnsErr.IsTemporary:
			kind = "temporary"
		}
	}
	return models.Finding{
		ProbeID:    probeID,
		Subject:    domain,
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{"error": err.Error(), "kind": kind},
	}
}
