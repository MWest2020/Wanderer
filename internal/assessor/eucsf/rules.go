// Package eucsf ships the EU Cloud Sovereignty Framework (SEAL)
// rule pack as a sibling to the DICTU pack. Each rule consumes the
// same models.Finding shape; the verdict is expressed on the SEAL
// 0–4 scale (`models.SealLevel`) and translated back to the
// engine-wide `models.Score` so per-dimension aggregation works
// unchanged. SEAL reference:
// https://commission.europa.eu/document/download/09579818-64a6-4dd5-9577-446ab6219113_en
package eucsf

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// euCountries is the EU-27 (without EEA-only Iceland / Liechtenstein
// / Norway). SEAL is an EU framework; a separate carve-out for EEA
// would change the substantive verdict.
var euCountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
	"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
	"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
	"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
	"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
	"ES": true, "SE": true,
}

// knownUSHyperscalerOrgs mirrors the DICTU set; SEAL flags the same
// US hyperscalers under a different criterium ID.
var knownUSHyperscalerOrgs = []string{
	"amazon", "google", "microsoft",
	"cloudflare", "akamai", "fastly",
}

// DefaultRules returns the MVP SEAL rule pack — five rules covering
// the criteria SOV-2, SOV-3, SOV-4, and SOV-6 of the framework, to
// the depth current Wanderer findings can address.
func DefaultRules() []assessor.Rule {
	return []assessor.Rule{
		certIssuerEU(),
		apexJurisdiction(),
		mxJurisdiction(),
		operationalEU(),
		noUSHyperscaler(),
		hostNoUSTelemetry(),
		nextcloudSupplyChain(),
		containerSupplyChain(),
	}
}

// ----- helpers shared with the rule bodies below -----

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

// emit wraps a SEAL level into a RuleResult, attaching the SEAL level
// to Attributes is left to the engine — but the rule's verdict text
// surfaces it in the natural-language explanation.
func emit(level models.SealLevel, verdict string, evidence []string) assessor.RuleResult {
	return assessor.RuleResult{
		Score:    level.ToScore(),
		Verdict:  verdict + " [SEAL " + strings.TrimPrefix(string(level), "seal_") + "]",
		Evidence: evidence,
	}
}

// noEvidence is the canonical "rule could not score" return.
func noEvidence(verdict string) assessor.RuleResult {
	return assessor.RuleResult{
		Score:   models.ScoreOnbekend,
		Verdict: verdict,
	}
}

// ----- SOV-2 — TLS issuer jurisdiction -----

func certIssuerEU() assessor.Rule {
	return assessor.Rule{
		ID:          "eucsf.sov2.cert_issuer_eu",
		Dimension:   models.DimensionJuridisch,
		Description: "TLS certificate issued by a CA registered in the EU.",
		Rationale: "Under SOV-2 (EU CSF), the cryptographic identity of an EU-sovereign " +
			"service is anchored by an EU-registered Certificate Authority. " +
			"A non-EU CA can be compelled by its home jurisdiction to revoke " +
			"or refuse renewal — a vector that bypasses any contractual " +
			"protection in the service's own SLA.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "tls.issuer" {
					continue
				}
				countries := stringsFromAttr(f.Attributes, "issuer_country")
				if len(countries) == 0 {
					continue
				}
				inEU := 0
				for _, c := range countries {
					if euCountries[strings.ToUpper(c)] {
						inEU++
					}
				}
				switch {
				case inEU == len(countries):
					return emit(models.SEAL4,
						fmt.Sprintf("cert issued in %s (EU)", strings.Join(countries, ",")),
						[]string{f.ID})
				case inEU > 0:
					return emit(models.SEAL3,
						fmt.Sprintf("cert issuer split across %s", strings.Join(countries, ",")),
						[]string{f.ID})
				default:
					return emit(models.SEAL1,
						fmt.Sprintf("cert issued in %s (outside EU)", strings.Join(countries, ",")),
						[]string{f.ID})
				}
			}
			return noEvidence("no tls.issuer finding — TLS probe did not run")
		},
	}
}

// ----- SOV-2 — Apex IP jurisdiction -----

func apexJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID:          "eucsf.sov2.apex_jurisdiction",
		Dimension:   models.DimensionJuridisch,
		Description: "Apex IP addresses register to ASNs in the EU.",
		Rationale: "SOV-2 expects the apex of the service to be hosted in an EU " +
			"member state's jurisdiction. The apex IP is the front door of the " +
			"service; SEAL levels reflect the share of apex traffic landing on " +
			"EU-registered ASs. Mixed jurisdictions earn a partial level.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var apex string
			for _, f := range findings {
				if f.ProbeID == "dns.a" || f.ProbeID == "dns.aaaa" {
					apex = f.Subject
					break
				}
			}
			if apex == "" {
				return noEvidence("no apex DNS finding — cannot identify apex host")
			}
			seen, inEU := 0, 0
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
				if euCountries[country] {
					inEU++
				}
			}
			if seen == 0 {
				return noEvidence("no ip.asn finding for apex — IP probe did not run")
			}
			switch {
			case inEU == seen:
				return emit(models.SEAL4,
					fmt.Sprintf("apex IPs in %s (EU)", strings.Join(countries, ",")),
					evidence)
			case inEU > 0:
				return emit(models.SEAL3,
					fmt.Sprintf("apex IPs split across %s", strings.Join(countries, ",")),
					evidence)
			default:
				return emit(models.SEAL1,
					fmt.Sprintf("apex IPs in %s (outside EU)", strings.Join(countries, ",")),
					evidence)
			}
		},
	}
}

// ----- SOV-3 — MX host jurisdiction -----

func mxJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID: "eucsf.sov3.mx_jurisdiction",
		Rationale: "SOV-3 covers data flows out of the service. Mail destined for the " +
			"organisation flows through its `MX` host; an MX in a non-EU AS is a " +
			"continuous outbound data flow into a foreign jurisdiction. SEAL " +
			"awards higher levels when every MX target lands on an EU-registered " +
			"AS, partial credit for mixed setups, and SEAL 0 for fully non-EU mail.",
		Dimension:   models.DimensionDataAI,
		Description: "MX hosts register to ASNs in the EU.",
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
				return noEvidence("no dns.mx finding — no mail routing to assess")
			}
			seen, inEU := 0, 0
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
				if euCountries[country] {
					inEU++
				}
			}
			if seen == 0 {
				return noEvidence("mx hosts found but no ip.asn lookup — IP probe did not run")
			}
			for _, id := range mxHosts {
				evidence = append(evidence, id)
			}
			switch {
			case inEU == seen:
				return emit(models.SEAL4,
					fmt.Sprintf("mx hosts in %s (EU)", strings.Join(countries, ",")),
					evidence)
			case inEU > 0:
				return emit(models.SEAL3,
					fmt.Sprintf("mx hosts split across %s", strings.Join(countries, ",")),
					evidence)
			default:
				return emit(models.SEAL1,
					fmt.Sprintf("mx hosts in %s (outside EU)", strings.Join(countries, ",")),
					evidence)
			}
		},
	}
}

// ----- SOV-4 — Operational dependencies in the EU -----

func operationalEU() assessor.Rule {
	return assessor.Rule{
		ID: "eucsf.sov4.operational_eu",
		Rationale: "SOV-4 is operational sovereignty: the people and processes that " +
			"keep the service running. The current evidence approximates this by " +
			"checking that the operational artefacts the perimeter probes can " +
			"see (nameservers, certificate issuance controls) live within EU " +
			"jurisdiction. A future revision will add operator-of-record " +
			"signals from the agent.",
		Dimension:   models.DimensionTechnologie,
		Description: "Third-party HTTP dependencies resolve to ASNs in the EU.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			thirdParties := map[string]string{}
			for _, f := range findings {
				if f.ProbeID != "http.third_party" {
					continue
				}
				thirdParties[strings.ToLower(f.Subject)] = f.ID
			}
			if len(thirdParties) == 0 {
				return noEvidence("no http.third_party finding — HTTP probe found no third parties or did not run")
			}
			seen, inEU := 0, 0
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
				if euCountries[country] {
					inEU++
				}
			}
			if seen == 0 {
				return noEvidence("third parties found but no ip.asn lookup — IP probe did not run")
			}
			for _, id := range thirdParties {
				evidence = append(evidence, id)
			}
			switch {
			case inEU == seen:
				return emit(models.SEAL4,
					fmt.Sprintf("all %d third-party hosts resolve in the EU", seen),
					evidence)
			case inEU > 0:
				return emit(models.SEAL3,
					fmt.Sprintf("%d of %d third-party hosts resolve in the EU", inEU, seen),
					evidence)
			default:
				return emit(models.SEAL1,
					fmt.Sprintf("all %d third-party hosts resolve outside the EU", seen),
					evidence)
			}
		},
	}
}

// ----- SOV-6 — No US hyperscaler dependence -----

func noUSHyperscaler() assessor.Rule {
	return assessor.Rule{
		ID: "eucsf.sov6.no_us_hyperscaler",
		Rationale: "SOV-6 mirrors DICTU's hyperscaler rule under the SEAL framing: a " +
			"service whose apex or third-party traffic flows through a US-" +
			"headquartered hyperscaler operates within the CLOUD Act's " +
			"jurisdictional reach, regardless of the data centre's physical " +
			"location. SEAL 0 when any hyperscaler is in the path; SEAL 4 when " +
			"none are.",
		Dimension:   models.DimensionTechnologie,
		Description: "Apex and third-party hosts do not depend on known US hyperscalers.",
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
				for _, f := range findings {
					if f.ProbeID == "ip.asn" {
						return emit(models.SEAL4,
							"no known US hyperscaler in the apex or third-party path",
							[]string{f.ID})
					}
				}
				return noEvidence("no ip.asn finding — IP probe did not run")
			}
			return emit(models.SEAL1,
				fmt.Sprintf("US hyperscaler in path: %s", strings.Join(dedupe(matched), ", ")),
				evidence)
		},
	}
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
