package scanner

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/MWest2020/wanderer/internal/domainutil"
	"github.com/MWest2020/wanderer/internal/probe/whois"
	"github.com/MWest2020/wanderer/pkg/models"
)

// nsHolderFindings performs one RDAP lookup per unique registrable
// domain among the target's dns.ns hosts and returns the resulting
// whois.ns_holder / whois.ns_holder.unavailable Findings. Findings are
// keyed to the registrable domain, not the individual NS host, so N
// nameservers under one provider — which share one registrant — cost
// exactly one lookup. A lookup failure (including a TLD with no RDAP
// service) emits whois.ns_holder.unavailable for that domain; it never
// aborts the scan.
func (s *Scanner) nsHolderFindings(ctx context.Context, findings []models.Finding, logger *slog.Logger) []models.Finding {
	domains := uniqueRegistrableDomains(findings)
	if len(domains) == 0 {
		return nil
	}
	client := s.Config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	var out []models.Finding
	for _, domain := range domains {
		status, err := whois.LookupRegistrantStatus(ctx, domain, s.RDAPBaseURL, client, s.Config.UserAgent)
		if err != nil {
			logger.Warn("scan.ns_holder_lookup_failed", "domain", domain, "err", err)
			out = append(out, models.Finding{
				ProbeID:       "whois.ns_holder.unavailable",
				DimensionHint: models.DimensionAccountability,
				Subject:       domain,
				Severity:      models.SeverityInfo,
				Attributes: map[string]any{
					"reason": err.Error(),
				},
			})
			continue
		}
		out = append(out, models.Finding{
			ProbeID:       "whois.ns_holder",
			DimensionHint: models.DimensionAccountability,
			Subject:       domain,
			Severity:      models.SeverityObservation,
			Attributes: map[string]any{
				"registrant": status,
			},
		})
	}
	return out
}

// uniqueRegistrableDomains returns the sorted, deduplicated
// registrable domains of every dns.ns finding's host.
func uniqueRegistrableDomains(findings []models.Finding) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range findings {
		if f.ProbeID != "dns.ns" {
			continue
		}
		host, _ := f.Attributes["host"].(string)
		rd := domainutil.Registrable(host)
		if rd == "" || seen[rd] {
			continue
		}
		seen[rd] = true
		out = append(out, rd)
	}
	sort.Strings(out)
	return out
}
