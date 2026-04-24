// Package dictu ships the MVP rule set for the DICTU Toetsingsinstrument
// Soevereiniteit Clouddiensten. Each rule is a pure Go function of the
// finding set; adding a rule means adding a function here and wiring it
// into DefaultRules. Rules that need data a current probe does not
// produce return RuleResult{} (no evidence) so the assessor can report
// them as Incomplete rather than absent.
package dictu

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// eeaCountries is the EU-27 plus Iceland, Liechtenstein, Norway.
// Kept as an explicit set rather than a lookup function so two
// reviewers can eyeball what counts as "in the EEA".
var eeaCountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
	"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
	"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
	"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
	"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
	"ES": true, "SE": true,
	"IS": true, "LI": true, "NO": true,
}

// knownUSHyperscalerOrgs are substrings (case-insensitive) that
// identify a handful of large US-headquartered hyperscalers in
// MaxMind's AS organisation string. The list is intentionally short:
// the rule's job is to flag obvious concentration, not to be a
// comprehensive vendor catalogue.
var knownUSHyperscalerOrgs = []string{
	"amazon", "google", "microsoft",
	"cloudflare", "akamai", "fastly",
}

// DefaultRules returns the MVP DICTU rule set. Rules are independently
// testable — this function exists only as the default wiring.
func DefaultRules() []assessor.Rule {
	return []assessor.Rule{
		certIssuerEEA(),
		apexIPInEEA(),
		mxVendorJurisdiction(),
		certValidity(),
		dnsRedundancy(),
		caaRestricts(),
		thirdPartiesEEA(),
		noUSHyperscaler(),
		mxPresent(),
		oidcFederation(),
	}
}

// ---------- Juridisch ----------

func certIssuerEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.juridisch.cert_issuer_eea",
		Dimension:   models.DimensionJuridisch,
		Description: "TLS certificate issued by an authority in the EEA.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "tls.issuer" {
					continue
				}
				countries := stringsFromAttr(f.Attributes, "issuer_country")
				if len(countries) == 0 {
					continue
				}
				inEEA := 0
				for _, c := range countries {
					if eeaCountries[strings.ToUpper(c)] {
						inEEA++
					}
				}
				switch {
				case inEEA == len(countries):
					return assessor.RuleResult{
						Score:    models.ScoreSoeverein,
						Verdict:  fmt.Sprintf("cert issued in %s (EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				case inEEA > 0:
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("cert issuer jurisdictions %s (mixed EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				default:
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  fmt.Sprintf("cert issued in %s (outside EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				}
			}
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no tls.issuer finding — TLS probe did not run or failed"}
		},
	}
}

func apexIPInEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.juridisch.apex_ip_eea",
		Dimension:   models.DimensionJuridisch,
		Description: "Apex IP addresses resolve to AS registered in the EEA.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var apex string
			for _, f := range findings {
				if f.ProbeID == "dns.A" || f.ProbeID == "dns.AAAA" {
					apex = f.Subject
					break
				}
			}
			if apex == "" {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no apex DNS finding — cannot identify apex host"}
			}
			var seen, inEEA int
			var evidence []string
			var countries []string
			for _, f := range findings {
				if f.ProbeID != "ip.asn" || f.Subject != apex {
					continue
				}
				country := strings.ToUpper(stringFromAttr(f.Attributes, "country"))
				if country == "" {
					continue
				}
				seen++
				countries = append(countries, country)
				evidence = append(evidence, f.ID)
				if eeaCountries[country] {
					inEEA++
				}
			}
			if seen == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no ip.asn finding for apex — IP probe did not run"}
			}
			switch {
			case inEEA == seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("apex IPs in %s (EEA)", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			case inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  fmt.Sprintf("apex IPs split across %s", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("apex IPs in %s (outside EEA)", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			}
		},
	}
}

func mxVendorJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.juridisch.mx_vendor_jurisdiction",
		Dimension:   models.DimensionJuridisch,
		Description: "MX hosts resolve to AS registered in the EEA.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			mxHosts := map[string]string{}
			for _, f := range findings {
				if f.ProbeID != "dns.mx" {
					continue
				}
				h := strings.TrimSuffix(stringFromAttr(f.Attributes, "host"), ".")
				if h == "" {
					continue
				}
				mxHosts[strings.ToLower(h)] = f.ID
			}
			if len(mxHosts) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.mx finding — no mail routing to assess"}
			}
			var seen, inEEA int
			var evidence []string
			var countries []string
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				host := strings.TrimSuffix(strings.ToLower(f.Subject), ".")
				if _, ok := mxHosts[host]; !ok {
					continue
				}
				country := strings.ToUpper(stringFromAttr(f.Attributes, "country"))
				if country == "" {
					continue
				}
				seen++
				countries = append(countries, country)
				evidence = append(evidence, f.ID)
				if eeaCountries[country] {
					inEEA++
				}
			}
			if seen == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "mx hosts found but no ip.asn lookup — IP probe did not run"}
			}
			// Merge in the MX finding IDs so the rationale cites both
			// sides of the correlation.
			for _, id := range mxHosts {
				evidence = append(evidence, id)
			}
			switch {
			case inEEA == seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("mx hosts in %s (EEA)", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			case inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  fmt.Sprintf("mx hosts split across %s", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("mx hosts in %s (outside EEA)", strings.Join(countries, ",")),
					Evidence: evidence,
				}
			}
		},
	}
}

// ---------- Operationeel ----------

func certValidity() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.operationeel.cert_validity",
		Dimension:   models.DimensionOperationeel,
		Description: "TLS certificate is valid and not expiring within 30 days.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "tls.validity" {
					continue
				}
				expired, _ := f.Attributes["expired"].(bool)
				expiringSoon, _ := f.Attributes["expiring_soon"].(bool)
				daysLeft, _ := f.Attributes["days_left"].(int)
				if v, ok := f.Attributes["days_left"].(float64); ok {
					// JSON unmarshal surfaces numbers as float64 if a
					// caller round-trips through JSON before assessing.
					daysLeft = int(v)
				}
				switch {
				case expired:
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  "certificate expired",
						Evidence: []string{f.ID},
					}
				case expiringSoon:
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("certificate expires in %d days", daysLeft),
						Evidence: []string{f.ID},
					}
				default:
					return assessor.RuleResult{
						Score:    models.ScoreSoeverein,
						Verdict:  fmt.Sprintf("certificate valid, %d days remaining", daysLeft),
						Evidence: []string{f.ID},
					}
				}
			}
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no tls.validity finding — TLS probe did not run"}
		},
	}
}

func dnsRedundancy() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.operationeel.dns_redundancy",
		Dimension:   models.DimensionOperationeel,
		Description: "At least two authoritative nameservers are delegated.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			hosts := map[string]bool{}
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "dns.ns" {
					continue
				}
				h := strings.TrimSuffix(strings.ToLower(stringFromAttr(f.Attributes, "host")), ".")
				if h == "" {
					continue
				}
				hosts[h] = true
				evidence = append(evidence, f.ID)
			}
			if len(hosts) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.ns finding — DNS probe did not return NS records"}
			}
			if len(hosts) < 2 {
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  "only one nameserver delegated; no redundancy",
					Evidence: evidence,
				}
			}
			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("%d nameservers delegated", len(hosts)),
				Evidence: evidence,
			}
		},
	}
}

func caaRestricts() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.operationeel.caa_restricts_issuance",
		Dimension:   models.DimensionOperationeel,
		Description: "CAA records restrict which CAs may issue certificates.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var evidence []string
			var hasCAA bool
			var noAnswer bool
			for _, f := range findings {
				if f.ProbeID != "dns.caa" {
					continue
				}
				if _, isNoAnswer := f.Attributes["no_answer"]; isNoAnswer {
					noAnswer = true
					evidence = append(evidence, f.ID)
					continue
				}
				tag := strings.ToLower(stringFromAttr(f.Attributes, "tag"))
				value := stringFromAttr(f.Attributes, "value")
				if (tag == "issue" || tag == "issuewild") && value != "" {
					hasCAA = true
					evidence = append(evidence, f.ID)
				}
			}
			switch {
			case hasCAA:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  "CAA records restrict certificate issuance",
					Evidence: evidence,
				}
			case noAnswer:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  "no CAA records — any public CA may issue",
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.caa finding — DNS probe did not query CAA"}
			}
		},
	}
}

// ---------- Technologie ----------

func thirdPartiesEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.technologie.third_parties_eea",
		Dimension:   models.DimensionTechnologie,
		Description: "HTTP third-party dependencies resolve to AS registered in the EEA.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			thirdParties := map[string]string{} // host -> finding id
			for _, f := range findings {
				if f.ProbeID != "http.third_party" {
					continue
				}
				thirdParties[strings.ToLower(f.Subject)] = f.ID
			}
			if len(thirdParties) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no http.third_party finding — HTTP probe found no third parties or did not run"}
			}
			seen, inEEA := 0, 0
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				host := strings.ToLower(f.Subject)
				if _, ok := thirdParties[host]; !ok {
					continue
				}
				country := strings.ToUpper(stringFromAttr(f.Attributes, "country"))
				if country == "" {
					continue
				}
				seen++
				evidence = append(evidence, f.ID)
				if eeaCountries[country] {
					inEEA++
				}
			}
			if seen == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "third parties found but no ip.asn lookup — IP probe did not run"}
			}
			for _, id := range thirdParties {
				evidence = append(evidence, id)
			}
			switch {
			case inEEA == seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("all %d third-party hosts resolve in the EEA", seen),
					Evidence: evidence,
				}
			case inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  fmt.Sprintf("%d of %d third-party hosts resolve in the EEA", inEEA, seen),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("all %d third-party hosts resolve outside the EEA", seen),
					Evidence: evidence,
				}
			}
		},
	}
}

func noUSHyperscaler() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.technologie.no_us_hyperscaler",
		Dimension:   models.DimensionTechnologie,
		Description: "Apex and third-party hosts are not routed via known US hyperscalers.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var matched []string
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				org := strings.ToLower(stringFromAttr(f.Attributes, "organisation"))
				if org == "" {
					continue
				}
				for _, needle := range knownUSHyperscalerOrgs {
					if strings.Contains(org, needle) {
						matched = append(matched, stringFromAttr(f.Attributes, "organisation"))
						evidence = append(evidence, f.ID)
						break
					}
				}
			}
			if len(evidence) == 0 {
				// No ip.asn at all? Return no-evidence. Any ip.asn
				// seen without a hyperscaler match means we have
				// evidence of absence.
				for _, f := range findings {
					if f.ProbeID == "ip.asn" {
						return assessor.RuleResult{
							Score:    models.ScoreVoldoende,
							Verdict:  "no known US hyperscaler in the apex or third-party path",
							Evidence: []string{f.ID},
						}
					}
				}
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no ip.asn finding — IP probe did not run"}
			}
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  fmt.Sprintf("US hyperscaler in path: %s", strings.Join(dedupe(matched), ", ")),
				Evidence: evidence,
			}
		},
	}
}

// ---------- Data & AI ----------

func mxPresent() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.data_ai.mx_present",
		Dimension:   models.DimensionDataAI,
		Description: "Domain has configured mail exchangers (routing is knowable).",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var evidence []string
			for _, f := range findings {
				if f.ProbeID == "dns.mx" {
					evidence = append(evidence, f.ID)
				}
			}
			if len(evidence) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.mx finding — domain may not accept mail"}
			}
			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("%d MX records published", len(evidence)),
				Evidence: evidence,
			}
		},
	}
}

func oidcFederation() assessor.Rule {
	return assessor.Rule{
		ID:          "dictu.data_ai.oidc_federation",
		Dimension:   models.DimensionDataAI,
		Description: "Identity federation endpoints are sovereign. Requires the egress probe (not yet landed).",
		Match: func(_ []models.Finding) assessor.RuleResult {
			// Intentional no-evidence until add-egress-probe ships.
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no evidence — requires add-egress-probe"}
		},
	}
}

// ---------- helpers ----------

func stringFromAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if s, ok := attrs[key].(string); ok {
		return s
	}
	return ""
}

func stringsFromAttr(attrs map[string]any, key string) []string {
	if attrs == nil {
		return nil
	}
	switch v := attrs[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	}
	return nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
