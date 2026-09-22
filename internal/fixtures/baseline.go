package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// BuildBaseline writes the minimal happy-path scenario: two
// organisations (voorbeeld + acme), voorbeeld carrying two
// perimeter domains and acme one, one scored scan each. Every
// existing perimeter rule that the demo Playwright suite exercises
// has at least one non-onbekend row after the seed runs.
//
// Score shape on purpose:
//
//   - voorbeeld.nl is fully soeverein (NL-issued TLS, NL-hosted
//     IP, NL-hosted mail) so the dashboard pill is green
//   - tweede.nl is voorbeeld's second fleet domain: afhankelijk on
//     the certificate dimension, scanned two days before
//     voorbeeld.nl, so the vloot-en-regels Playwright spec (run 06)
//     can prove "sort by score" and "sort by last scan" produce
//     different orders within one organisation's fleet
//   - acme.example.com is afhankelijk on the certificate dimension
//     (US issuer) so the reporting catalogue's Current-state
//     column shows mixed verdicts, and its domain registration
//     expires inside the 30-day urgent window so
//     wand.operationeel.domain_expiry has a target on the
//     "afhankelijk" side of its threshold for the regelpagina spec
func BuildBaseline(ctx context.Context, st *store.Store) error {
	voorbeeld, err := upsertOrg(ctx, st, "voorbeeld", "Voorbeeld B.V.")
	if err != nil {
		return err
	}
	acme, err := upsertOrg(ctx, st, "acme", "ACME B.V.")
	if err != nil {
		return err
	}

	voorbeeldTarget, err := upsertTarget(ctx, st, "voorbeeld.nl", models.TargetKindDomain, voorbeeld.ID)
	if err != nil {
		return err
	}
	voorbeeldSecondTarget, err := upsertTarget(ctx, st, "tweede.nl", models.TargetKindDomain, voorbeeld.ID)
	if err != nil {
		return err
	}
	acmeTarget, err := upsertTarget(ctx, st, "acme.example.com", models.TargetKindDomain, acme.ID)
	if err != nil {
		return err
	}

	if _, err := addCompletedScan(ctx, st, voorbeeldTarget, baseTime, baselineSovereignFindings("voorbeeld.nl")); err != nil {
		return fmt.Errorf("baseline: voorbeeld scan: %w", err)
	}
	if _, err := addCompletedScan(ctx, st, voorbeeldSecondTarget, baseTime.Add(-48*time.Hour), baselineDependentFindings("tweede.nl")); err != nil {
		return fmt.Errorf("baseline: voorbeeld second-domain scan: %w", err)
	}
	if _, err := addCompletedScan(ctx, st, acmeTarget, baseTime, baselineExpiringDependentFindings("acme.example.com")); err != nil {
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
			"organisation": "Voorbeeld B.V.",
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
		// Accountability-dimension findings (run 08b): shared by both
		// baseline scans so each scan's answer sheet demonstrates all
		// four answer states on its own. config.expected_registrant
		// resolves accountabilityDomain() so registrant_identifiable
		// can tell voorbeeld.nl's ".nl" TLD (registry-redacted, n.v.t.)
		// apart from acme.example.com's ".com" (no whois.registrant_identity
		// finding here, so it reads onbekend/probe_unavailable) — see
		// internal/assessor/wand/accountability_rules.go.
		mkFinding("config.expected_registrant", domain, models.DimensionAccountability, map[string]any{
			"expected_registrant": []string{},
		}),
		// no whois.registrant_identity / whois.reseller name lookup
		// succeeded — whois.reseller "present" below still scores
		// no_reseller independently of registrant_identifiable.
		mkFinding("whois.reseller", domain, models.DimensionAccountability, map[string]any{
			"present": true,
			"name":    "Reseller Hosting B.V.",
		}),
		mkFinding("http.securitytxt", domain, models.DimensionAccountability, map[string]any{
			"present":   true,
			"parseable": true,
			"contact":   []string{"mailto:security@" + domain},
			"expires":   baseTime.AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z07:00"),
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

// baselineExpiringDependentFindings adds a whois.expiry finding
// inside wand.operationeel.domain_expiry's 30-day urgent window on
// top of baselineDependentFindings, so the rule scores afhankelijk
// instead of its default onbekend (no registry publishes an expiry
// event for voorbeeld.nl's ".nl" TLD, so that domain stays onbekend
// on purpose — see accountability_rules.go's domainExpiry doc
// comment). The vloot-en-regels regelpagina spec (run 06) needs one
// target on the "afhankelijk" side of a rule that carries an
// explicit Threshold, and domain_expiry is the only such rule with a
// fixture-reachable failing row.
func baselineExpiringDependentFindings(domain string) []models.Finding {
	out := baselineDependentFindings(domain)
	out = append(out, mkFinding("whois.expiry", domain, models.DimensionAccountability, map[string]any{
		"present": true,
		"date":    baseTime.AddDate(0, 0, 20).Format(time.RFC3339),
	}))
	return out
}
