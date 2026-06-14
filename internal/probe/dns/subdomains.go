package dns

import (
	"context"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// commonPrefixes is the fixed list the prefix-probe sweeps. Adding a
// prefix is a one-line PR; the list stays small on purpose so the
// probe does not become a brute-force scanner. Use Amass for that.
var commonPrefixes = []string{
	"www", "mail", "m", "api", "auth", "sso",
	"mijn", "loket", "inloggen", "nextcloud", "webmail",
	"vpn", "portal", "intranet", "extranet",
	"wachtwoord", "wifi", "gast",
}

// prefixHit is the per-prefix resolution result inside subdomainSweep.
// Lifted to a named type so the wildcard helper can take a typed
// slice rather than an anonymous-struct alias.
type prefixHit struct {
	name string
	ips  []string
}

// subdomainSweep runs LookupHost for every common prefix under
// domain. Resolving names emit a `dns.subdomain` Finding; if every
// prefix resolves to the same IP set we emit a single
// `dns.subdomain.wildcard` instead so we do not flood the assessor
// with 18 spurious entries on a wildcard DNS configuration.
func (p *Probe) subdomainSweep(ctx context.Context, domain string) []models.Finding {
	var hits []prefixHit
	for _, prefix := range commonPrefixes {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		full := prefix + "." + domain
		addrs, err := p.Resolver.LookupHost(ctx, full)
		if err != nil || len(addrs) == 0 {
			continue
		}
		sort.Strings(addrs)
		hits = append(hits, prefixHit{name: full, ips: addrs})
	}
	if len(hits) == 0 {
		return nil
	}
	if isWildcard(hits) {
		return []models.Finding{{
			ProbeID:  "dns.subdomain.wildcard",
			Subject:  domain,
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"source":    "prefix_probe",
				"hit_count": len(hits),
				"ips":       hits[0].ips,
			},
		}}
	}
	out := make([]models.Finding, 0, len(hits))
	for _, h := range hits {
		out = append(out, models.Finding{
			ProbeID:  "dns.subdomain",
			Subject:  h.name,
			Severity: models.SeverityObservation,
			Attributes: map[string]any{
				"source":      "prefix_probe",
				"apex_domain": domain,
				"addresses":   h.ips,
			},
		})
	}
	return out
}

// isWildcard reports whether every hit resolves to the same IP set.
// The check is by normalised string comparison: same number of IPs,
// same sorted list.
func isWildcard(hits []prefixHit) bool {
	if len(hits) < 2 {
		return false
	}
	first := strings.Join(hits[0].ips, ",")
	for _, h := range hits[1:] {
		if strings.Join(h.ips, ",") != first {
			return false
		}
	}
	return true
}
