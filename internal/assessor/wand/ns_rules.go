package wand

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// observedDNSOperators returns the recognisable managed-DNS operator
// names the scanner's dns.ns_hosting synthesis observed (de-duplicated),
// so the rule can lead its verdict with the "who" — "DNS run by
// Cloudflare …" rather than a bare country. Tolerant of the routes
// attribute's in-memory ([]map[string]any) and JSON-reloaded ([]any)
// shapes. Empty when the synthesis named no operator, in which case the
// rule keeps its country-only verdict.
func observedDNSOperators(findings []models.Finding) []string {
	var ops []string
	seen := map[string]bool{}
	add := func(m map[string]any) {
		op, _ := m["operator"].(string)
		if op == "" || seen[op] {
			return
		}
		seen[op] = true
		ops = append(ops, op)
	}
	for _, f := range findings {
		if f.ProbeID != "dns.ns_hosting" {
			continue
		}
		switch routes := f.Attributes["routes"].(type) {
		case []map[string]any:
			for _, m := range routes {
				add(m)
			}
		case []any:
			for _, r := range routes {
				if m, ok := r.(map[string]any); ok {
					add(m)
				}
			}
		}
		break
	}
	return ops
}

// nsVendorJurisdiction scores who runs the organisation's authoritative
// DNS and where. It correlates the dns.ns hosts with the ip.asn lookups
// the IP probe performed on them (the scanner adds NS hosts to
// Target.Related), the same correlation pattern as the MX rule. DNS is
// the control plane for every name the organisation publishes; a
// non-EEA DNS operator (e.g. a US-headquartered managed-DNS provider)
// resolves and can withhold those names under foreign jurisdiction.
func nsVendorJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.juridisch.ns_vendor_jurisdiction",
		Dimension:   models.DimensionJuridisch,
		Description: "Authoritative nameserver hosts resolve to AS registered in the EEA.",
		Rationale: "Authoritative DNS is the control plane for every hostname the " +
			"organisation publishes. A non-EEA DNS vendor resolves (and can " +
			"withhold or redirect) those names under a foreign jurisdiction. " +
			"The rule names the operator and country so an operator sees who " +
			"actually runs their DNS.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			nsHosts := map[string]string{} // host -> dns.ns finding ID
			for _, f := range findings {
				if f.ProbeID != "dns.ns" {
					continue
				}
				h := strings.TrimSuffix(strings.ToLower(stringFromAttr(f.Attributes, "host")), ".")
				if h == "" {
					continue
				}
				nsHosts[h] = f.ID
			}
			if len(nsHosts) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.ns finding — no authoritative DNS to assess"}
			}

			var jt jurisdictionTally
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				host := strings.TrimSuffix(strings.ToLower(f.Subject), ".")
				if _, ok := nsHosts[host]; !ok {
					continue
				}
				jt.add(f)
			}
			if jt.seen == 0 {
				return jt.noCountryResult(
					"authoritative DNS run by",
					"authoritative DNS on an AS operator with no country (anycast?) — jurisdiction undetermined",
					"ns hosts found but no ip.asn lookup — IP probe did not run or no --geoip database",
				)
			}
			seen, inEEA := jt.seen, jt.inEEA
			countries := jt.countries
			evidence := jt.evidence
			// Cite both sides of the correlation.
			for _, id := range nsHosts {
				evidence = append(evidence, id)
			}
			sort.Strings(countries)
			// Lead with the observed operator ("DNS run by Cloudflare —
			// …") when the scanner named one; otherwise keep the
			// country-only verdict.
			lead := func(detail string) string {
				if who := joinAnd(observedDNSOperators(findings)); who != "" {
					return fmt.Sprintf("DNS run by %s — %s", who, detail)
				}
				return detail
			}
			switch {
			case inEEA == seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  lead(fmt.Sprintf("authoritative DNS in %s (EEA)", strings.Join(dedupe(countries), ","))),
					Evidence: evidence,
				}
			case inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  lead(fmt.Sprintf("authoritative DNS split across %s", strings.Join(dedupe(countries), ","))),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  lead(fmt.Sprintf("authoritative DNS in %s (outside EEA)", strings.Join(dedupe(countries), ","))),
					Evidence: evidence,
				}
			}
		},
	}
}
