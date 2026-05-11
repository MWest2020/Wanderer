package eucsf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// nextcloudSupplyChain is the SEAL analogue of the two wand
// Nextcloud rules. SEAL rolls supply-chain / vendor exposure
// into one observation, so this single rule walks both
// `inventory.nextcloud.objectstore` and
// `inventory.nextcloud.oidc_provider` Findings and emits one
// combined verdict.
func nextcloudSupplyChain() assessor.Rule {
	return assessor.Rule{
		ID:          "eucsf.sov6.nextcloud_supply_chain",
		Dimension:   models.DimensionTechnologie,
		Description: "Nextcloud supply chain (objectstore + IdP) resolves to EEA jurisdictions.",
		Rationale: "EUCSF sov-6 covers vendor dependency. For a Nextcloud " +
			"deployment the two materially-loaded surfaces are where " +
			"file data is stored (objectstore) and which identity " +
			"provider gates access (OIDC). The rule rolls them up " +
			"into one SEAL observation so the dashboard shows a single " +
			"supply-chain verdict alongside the wand-side split.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var inspectedStore, inspectedOIDC []models.Finding
			var nonEEAStore, nonEEAOIDC []models.Finding
			for _, f := range findings {
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				switch f.ProbeID {
				case "inventory.nextcloud.objectstore":
					inspectedStore = append(inspectedStore, f)
					country, _ := f.Attributes["country"].(string)
					if !eucsfIsEEA(country) {
						nonEEAStore = append(nonEEAStore, f)
					}
				case "inventory.nextcloud.oidc_provider":
					inspectedOIDC = append(inspectedOIDC, f)
					country, _ := f.Attributes["country"].(string)
					if !eucsfIsEEA(country) {
						nonEEAOIDC = append(nonEEAOIDC, f)
					}
				}
			}
			if len(inspectedStore)+len(inspectedOIDC) == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no inventory.nextcloud.objectstore / .oidc_provider findings — Nextcloud inspector did not run or no relevant configuration",
				}
			}
			hits := append([]models.Finding{}, nonEEAStore...)
			hits = append(hits, nonEEAOIDC...)
			if len(hits) == 0 {
				return assessor.RuleResult{
					Score: models.ScoreSoeverein,
					Verdict: fmt.Sprintf(
						"inspected %d objectstore + %d OIDC provider(s) — every Nextcloud supply-chain dependency resolves to an EEA jurisdiction [SEAL 4]",
						len(inspectedStore), len(inspectedOIDC),
					),
					Evidence: sampleNextcloudCombinedEvidence(inspectedStore, inspectedOIDC),
				}
			}
			sort.Slice(hits, func(i, j int) bool {
				if hits[i].ProbeID != hits[j].ProbeID {
					return hits[i].ProbeID < hits[j].ProbeID
				}
				return hits[i].Subject < hits[j].Subject
			})
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  combinedNextcloudVerdict(nonEEAStore, nonEEAOIDC),
				Evidence: collectNextcloudIDs(hits),
			}
		},
	}
}

func combinedNextcloudVerdict(stores, idps []models.Finding) string {
	parts := make([]string, 0, len(stores)+len(idps))
	for _, h := range stores {
		country, _ := h.Attributes["country"].(string)
		parts = append(parts, fmt.Sprintf("objectstore %s (%s)", h.Subject, country))
	}
	for _, h := range idps {
		country, _ := h.Attributes["country"].(string)
		parts = append(parts, fmt.Sprintf("OIDC %s (%s)", h.Subject, country))
	}
	return fmt.Sprintf("%d non-EEA Nextcloud supply-chain hits: %s [SEAL 1]", len(parts), strings.Join(parts, ", "))
}

func collectNextcloudIDs(in []models.Finding) []string {
	out := make([]string, 0, len(in))
	for _, f := range in {
		if f.ID != "" {
			out = append(out, f.ID)
		}
	}
	return out
}

const nextcloudCombinedSampleCap = 10

func sampleNextcloudCombinedEvidence(a, b []models.Finding) []string {
	merged := append([]models.Finding{}, a...)
	merged = append(merged, b...)
	limit := len(merged)
	if limit > nextcloudCombinedSampleCap {
		limit = nextcloudCombinedSampleCap
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if merged[i].ID != "" {
			out = append(out, merged[i].ID)
		}
	}
	return out
}

// eucsfEEACountries mirrors the wand list. Duplicated rather
// than cross-package import so reviewers can eyeball the SEAL
// pack's policy in one place.
var eucsfEEACountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
	"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
	"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
	"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
	"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
	"ES": true, "SE": true,
	"IS": true, "LI": true, "NO": true,
}

func eucsfIsEEA(c string) bool {
	return eucsfEEACountries[strings.ToUpper(c)]
}
