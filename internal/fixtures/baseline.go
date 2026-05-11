package fixtures

import (
	"context"
	"fmt"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// BuildBaseline writes the minimal happy-path scenario: two
// organisations (conduction + acme), one perimeter domain per
// org, one scored scan each. Every existing perimeter rule that
// the demo Playwright suite exercises has at least one
// non-onbekend row after the seed runs.
//
// Score shape on purpose:
//
//   - conduction.nl is fully soeverein (NL-issued TLS, NL-hosted
//     IP, NL-hosted mail) so the dashboard pill is green
//   - acme.example.com is afhankelijk on the certificate dimension
//     (US issuer) so the reporting catalogue's Current-state
//     column shows mixed verdicts
func BuildBaseline(ctx context.Context, st *store.Store) error {
	cond, err := upsertOrg(ctx, st, "conduction", "Conduction B.V.")
	if err != nil {
		return err
	}
	acme, err := upsertOrg(ctx, st, "acme", "ACME B.V.")
	if err != nil {
		return err
	}

	condTarget, err := upsertTarget(ctx, st, "conduction.nl", models.TargetKindDomain, cond.ID)
	if err != nil {
		return err
	}
	acmeTarget, err := upsertTarget(ctx, st, "acme.example.com", models.TargetKindDomain, acme.ID)
	if err != nil {
		return err
	}

	if _, err := addCompletedScan(ctx, st, condTarget, baseTime, baselineSovereignFindings("conduction.nl")); err != nil {
		return fmt.Errorf("baseline: conduction scan: %w", err)
	}
	if _, err := addCompletedScan(ctx, st, acmeTarget, baseTime, baselineDependentFindings("acme.example.com")); err != nil {
		return fmt.Errorf("baseline: acme scan: %w", err)
	}
	return nil
}

func baselineSovereignFindings(domain string) []models.Finding {
	return []models.Finding{
		mkFinding("tls.issuer", domain, models.DimensionJuridisch, map[string]any{
			"issuer_country": []string{"NL"},
			"issuer_org":     "GlobalSign nv-sa",
		}),
		mkFinding("tls.validity", domain, models.DimensionOperationeel, map[string]any{
			"days_left": 90,
			"not_after": baseTime.AddDate(0, 3, 0).Format("2006-01-02T15:04:05Z07:00"),
		}),
		mkFinding("dns.a", domain, models.DimensionJuridisch, map[string]any{
			"address": "5.39.18.20",
		}),
		mkFinding("dns.mx", domain, models.DimensionDataAI, map[string]any{
			"host": "mail." + domain,
		}),
		mkFinding("dns.ns", domain, models.DimensionOperationeel, map[string]any{
			"host": "ns1." + domain,
		}),
		mkFinding("dns.ns", domain, models.DimensionOperationeel, map[string]any{
			"host": "ns2." + domain,
		}),
		mkFinding("dns.caa", domain, models.DimensionOperationeel, map[string]any{
			"tag":   "issue",
			"value": "letsencrypt.org",
			"flag":  0,
		}),
		mkFinding("ip.asn", domain, models.DimensionTechnologie, map[string]any{
			"country":      "NL",
			"organisation": "TransIP B.V.",
			"asn":          20857,
		}),
		mkFinding("ip.asn", "mail."+domain, models.DimensionTechnologie, map[string]any{
			"country":      "NL",
			"organisation": "TransIP B.V.",
			"asn":          20857,
		}),
		mkFinding("whois.registrant", domain, models.DimensionJuridisch, map[string]any{
			"country":      "NL",
			"organisation": "Conduction B.V.",
		}),
		mkFinding("http.third_party", domain, models.DimensionTechnologie, map[string]any{
			"source_domain": domain,
			"third_party":   "cdn." + domain,
		}),
		mkFinding("ip.asn", "cdn."+domain, models.DimensionTechnologie, map[string]any{
			"country":      "NL",
			"organisation": "Leaseweb Netherlands B.V.",
			"asn":          60781,
		}),
	}
}

func baselineDependentFindings(domain string) []models.Finding {
	out := baselineSovereignFindings(domain)
	// Override the issuer so the cert rule scores afhankelijk —
	// every other rule stays soeverein, which is exactly the
	// mixed-verdict shape the Reporting status column wants to
	// surface.
	for i := range out {
		if out[i].ProbeID == "tls.issuer" {
			out[i].Attributes["issuer_country"] = []string{"US"}
			out[i].Attributes["issuer_org"] = "DigiCert Inc"
		}
	}
	return out
}
