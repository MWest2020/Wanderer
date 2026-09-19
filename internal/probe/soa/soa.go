// Package soa is the SOA probe. It queries the zone's SOA record and
// reports the RNAME (the zone's designated mailbox for contact) as a
// contactability signal: does the mailbox domain resolve, and does it
// advertise an MX. No SMTP connections are made — existence plus MX
// presence is the honest passive ceiling; delivery is never confirmed.
package soa

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Resolver is the minimal surface the probe needs. NewNetResolver
// wraps *net.Resolver plus a hand-rolled SOA query (net.Resolver
// exposes no SOA lookup, the same gap the DNS probe works around for
// CAA).
type Resolver interface {
	LookupSOA(ctx context.Context, domain string) (mname, rname string, err error)
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// Probe is the SOA probe.
type Probe struct {
	Resolver Resolver
}

// New returns a SOA probe using the system resolver.
func New() *Probe {
	return &Probe{Resolver: NewNetResolver(net.DefaultResolver)}
}

// ID implements probe.Probe.
func (*Probe) ID() string { return "soa" }

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, _ probe.Config) ([]models.Finding, error) {
	if p.Resolver == nil {
		return nil, errors.New("soa: resolver not set")
	}
	domain := target.Domain

	mname, rname, err := p.Resolver.LookupSOA(ctx, domain)
	if err != nil {
		return []models.Finding{unavailable(domain, err.Error())}, nil
	}

	attrs := map[string]any{
		"mname": mname,
		"rname": rname,
	}

	local, mailboxDomain, perr := splitRNAME(rname)
	if perr != nil {
		attrs["rname_parse_error"] = perr.Error()
		return []models.Finding{{
			ProbeID:       "dns.soa",
			DimensionHint: models.DimensionAccountability,
			Subject:       domain,
			Severity:      models.SeverityObservation,
			Attributes:    attrs,
		}}, nil
	}
	attrs["mailbox"] = local + "@" + mailboxDomain
	attrs["mailbox_domain"] = mailboxDomain

	resolves := false
	if addrs, err := p.Resolver.LookupHost(ctx, mailboxDomain); err == nil && len(addrs) > 0 {
		resolves = true
	}
	attrs["mailbox_domain_resolves"] = resolves

	hasMX := false
	if mx, err := p.Resolver.LookupMX(ctx, mailboxDomain); err == nil && len(mx) > 0 {
		hasMX = true
	}
	attrs["mailbox_domain_has_mx"] = hasMX

	return []models.Finding{{
		ProbeID:       "dns.soa",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
	}}, nil
}

// splitRNAME parses an SOA RNAME in RFC 1035 form into an email local
// part and domain. The first unescaped dot separates the two; an
// escaped dot (`\.`) belongs to the local part and is unescaped in the
// returned local part (e.g. `john\.doe.example.nl` ->
// ("john.doe", "example.nl")).
func splitRNAME(rname string) (local, domain string, err error) {
	name := strings.TrimSuffix(rname, ".")
	var b strings.Builder
	escaped := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case escaped:
			b.WriteByte(c)
			escaped = false
		case c == '\\':
			escaped = true
		case c == '.':
			domain = name[i+1:]
			if domain == "" {
				return "", "", fmt.Errorf("soa: rname %q has no domain part", rname)
			}
			return b.String(), domain, nil
		default:
			b.WriteByte(c)
		}
	}
	return "", "", fmt.Errorf("soa: rname %q has no unescaped dot", rname)
}

func unavailable(domain, reason string) models.Finding {
	return models.Finding{
		ProbeID:  "dns.soa.unavailable",
		Subject:  domain,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"reason": reason,
		},
	}
}
