// Package transit is the transit-path probe. It traces the network
// path to a target and attributes each hop (IP, reverse DNS, ASN, org,
// country) so an operator can see where a target is actually hosted and
// which jurisdictions its traffic crosses — the concrete, observed
// answer to "where does my Nextcloud live, and what does the path cross".
//
// The trace runs from the scanner's vantage. The destination-side hops
// (the hosting provider) are robust regardless of vantage; the middle
// transit hops are vantage-flavoured (an agent-modus, on-host trace is
// the stronger follow-up). See propose-transit-path-probe.
//
// Tracing uses an unprivileged external tool (tracepath / traceroute)
// when present; absent that, the probe degrades to a single
// "transit.unavailable" finding rather than failing the scan.
package transit

import (
	"context"
	"net"
	"sort"
	"time"

	ipprobe "github.com/MWest2020/wanderer/internal/probe/ip"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Hop is one step on the path. NoReply marks a hop that did not answer
// (a firewalled router); such hops carry no IP.
type Hop struct {
	Num     int
	IP      string
	RTTms   float64
	NoReply bool
}

// Tracer runs a path trace to an IP. Implementations wrap an external
// tool; the default lives in tracepath.go.
type Tracer interface {
	// Available reports whether a tracing tool was found.
	Available() bool
	// Trace returns the ordered hops to ip, bounded by maxHops.
	Trace(ctx context.Context, ip string, maxHops int) ([]Hop, error)
}

// Probe traces the path to a target and enriches each hop.
type Probe struct {
	Tracer   Tracer
	Geo      *ipprobe.Probe // optional GeoLite2 enrichment; nil = degraded
	Resolver *net.Resolver  // optional; nil uses net.DefaultResolver
	MaxHops  int            // 0 → 30
}

// New returns a transit probe using the given tracer and (optional)
// GeoLite2 enrichment.
func New(tracer Tracer, geo *ipprobe.Probe) *Probe {
	// 20 hops reaches virtually all public hosting while keeping the
	// trace inside the scanner's per-probe budget on ICMP-filtered
	// paths (every silent hop costs wall-clock).
	return &Probe{Tracer: tracer, Geo: geo, MaxHops: 20}
}

// ID implements probe.Probe.
func (*Probe) ID() string { return "transit" }

// Run traces the path to the target's primary address and emits one
// Finding per hop plus an aggregate path Finding.
func (p *Probe) Run(ctx context.Context, target models.Target, _ probe.Config) ([]models.Finding, error) {
	if p.Tracer == nil || !p.Tracer.Available() {
		return []models.Finding{{
			ProbeID:    "transit.unavailable",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"reason": "no tracepath/traceroute tool available"},
		}}, nil
	}
	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addrs, err := resolver.LookupHost(ctx, target.Domain)
	targetIP := firstIP(addrs)
	if err != nil || targetIP == "" {
		reason := "could not resolve target to an IP"
		if err != nil {
			reason = err.Error()
		}
		return []models.Finding{{
			ProbeID:    "transit.resolve",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"error": reason},
		}}, nil
	}

	maxHops := p.MaxHops
	if maxHops <= 0 {
		maxHops = 30
	}
	hops, err := p.Tracer.Trace(ctx, targetIP, maxHops)
	if err != nil {
		return []models.Finding{{
			ProbeID:    "transit.error",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"target_ip": targetIP, "error": err.Error()},
		}}, nil
	}

	findings := make([]models.Finding, 0, len(hops)+1)
	countries := map[string]bool{}
	asns := map[uint]bool{}
	var dest hopEnrichment

	for _, h := range hops {
		attrs := map[string]any{"hop": h.Num}
		if h.NoReply {
			attrs["no_reply"] = true
			findings = append(findings, models.Finding{
				ProbeID:    "transit.hop",
				Subject:    target.Domain,
				Severity:   models.SeverityInfo,
				Attributes: attrs,
			})
			continue
		}
		// Hops carry numeric IPs (the tool runs with -n), so the
		// parser's "[LOCALHOST]" guard and our IP handling never see
		// hostnames. enrich adds our own reverse DNS for display.
		enr := p.enrich(ctx, h.IP)
		attrs["ip"] = h.IP
		if h.RTTms > 0 {
			attrs["rtt_ms"] = h.RTTms
		}
		if enr.rdns != "" {
			attrs["rdns"] = enr.rdns
		}
		if enr.hasGeo {
			attrs["asn"] = enr.asn
			attrs["organisation"] = enr.org
			attrs["country"] = enr.country
			countries[enr.country] = true
			if enr.asn != 0 {
				asns[enr.asn] = true
			}
		}
		enr.ip = h.IP
		dest = enr // last responded hop wins → destination
		findings = append(findings, models.Finding{
			ProbeID:       "transit.hop",
			DimensionHint: models.DimensionJuridisch,
			Subject:       target.Domain,
			Severity:      models.SeverityFinding,
			Attributes:    attrs,
		})
	}

	// The probe reports observed facts only; the wand.transit rule
	// applies the EEA / non-EU-carrier judgement.
	agg := map[string]any{
		"target_ip":      targetIP,
		"hops_total":     len(hops),
		"hops_responded": len(hops) - countNoReply(hops),
		"countries":      sortedStrings(countries),
		"asns":           sortedUints(asns),
	}
	if dest.ip != "" {
		agg["dest_ip"] = dest.ip
		if dest.hasGeo {
			agg["dest_country"] = dest.country
			agg["dest_organisation"] = dest.org
			agg["dest_asn"] = dest.asn
		}
	}
	findings = append(findings, models.Finding{
		ProbeID:       "transit.path",
		DimensionHint: models.DimensionJuridisch,
		Subject:       target.Domain,
		Severity:      models.SeverityFinding,
		Attributes:    agg,
	})
	return findings, nil
}

type hopEnrichment struct {
	ip      string
	rdns    string
	asn     uint
	org     string
	country string
	hasGeo  bool
}

// enrich adds reverse DNS and GeoLite2 ASN/country for a hop IP. Both
// are best-effort; failures leave the fields empty.
func (p *Probe) enrich(ctx context.Context, ipStr string) hopEnrichment {
	out := hopEnrichment{ip: ipStr}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return out
	}
	// Reverse DNS — bounded so a slow PTR lookup cannot stall the scan.
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if names, err := resolver.LookupAddr(rctx, ipStr); err == nil && len(names) > 0 {
		out.rdns = names[0]
	}
	if p.Geo != nil {
		if l, err := p.Geo.Lookup(ip); err == nil {
			out.asn, out.org, out.country, out.hasGeo = l.ASN, l.Organisation, l.Country, true
		}
	}
	return out
}

func firstIP(addrs []string) string {
	// Prefer the first IPv4 (traceroute tools default to v4); fall back
	// to the first address of any family.
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	for _, a := range addrs {
		if net.ParseIP(a) != nil {
			return a
		}
	}
	return ""
}

func countNoReply(hops []Hop) int {
	n := 0
	for _, h := range hops {
		if h.NoReply {
			n++
		}
	}
	return n
}

func sortedStrings(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func sortedUints(m map[uint]bool) []uint {
	out := make([]uint, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
