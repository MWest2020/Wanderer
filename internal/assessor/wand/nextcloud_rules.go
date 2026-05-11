package wand

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// nextcloudObjectstoreEU scores the Nextcloud objectstore
// backend(s) the inspector observed. Reads
// `inventory.nextcloud.objectstore` Findings; each carries a
// `country` attribute (populated by the inspector via geoip).
// Soeverein when every backend resolves to an EEA jurisdiction;
// afhankelijk on any non-EEA hit; onbekend without findings.
func nextcloudObjectstoreEU() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.nextcloud.objectstore_eu",
		Dimension:   models.DimensionTechnologie,
		Description: "Nextcloud objectstore backend resolves to an EEA jurisdiction.",
		Rationale: "Where Nextcloud stores its file data is the single " +
			"most material sovereignty signal a Nextcloud install " +
			"produces. An S3 backend resolving to a US-headquartered " +
			"hyperscaler means every file uploaded through the EU-facing " +
			"frontend traverses a CLOUD-Act-bound storage layer. The rule " +
			"flags this without forbidding it — operators choose backends, " +
			"not the assessor.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var inspected []models.Finding
			var nonEEA []models.Finding
			for _, f := range findings {
				if f.ProbeID != "inventory.nextcloud.objectstore" {
					continue
				}
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				inspected = append(inspected, f)
				if country, _ := f.Attributes["country"].(string); !isEEACountry(country) {
					nonEEA = append(nonEEA, f)
				}
			}
			if len(inspected) == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no inventory.nextcloud.objectstore finding — Nextcloud inspector did not run, or no objectstore is configured",
				}
			}
			if len(nonEEA) == 0 {
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d objectstore backend(s) — every backend resolves to an EEA jurisdiction", len(inspected)),
					Evidence: sampleNextcloudEvidence(inspected),
				}
			}
			sort.Slice(nonEEA, func(i, j int) bool {
				return nonEEA[i].Subject < nonEEA[j].Subject
			})
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  nextcloudObjectstoreVerdict(nonEEA),
				Evidence: nextcloudFindingIDs(nonEEA),
			}
		},
	}
}

// nextcloudOIDCProviderEU scores the OIDC IdP(s) wired into the
// Nextcloud instance. Same shape as objectstore. The country
// attribute comes from a geoip lookup the inspector performs on
// the issuer URL's hostname (when a resolver is configured).
func nextcloudOIDCProviderEU() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.nextcloud.oidc_provider_eu",
		Dimension:   models.DimensionTechnologie,
		Description: "Nextcloud OIDC identity provider resolves to an EEA jurisdiction.",
		Rationale: "An OIDC IdP is the gatekeeper for every Nextcloud login. " +
			"When the IdP runs in a non-EEA jurisdiction, account " +
			"creation, group membership, and login traces sit in a " +
			"foreign control plane. The rule names the jurisdiction so " +
			"operators can act on it.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			// inventory.nextcloud.oidc.unavailable means the
			// inspector saw no user_oidc app (or an incompatible
			// alternative). That is not "no findings" — it is a
			// known gap, so the verdict text names it.
			var unavailable models.Finding
			var inspected []models.Finding
			var nonEEA []models.Finding
			for _, f := range findings {
				switch f.ProbeID {
				case "inventory.nextcloud.oidc.unavailable":
					if assessor.IsEvidenceLike(f) {
						// Inspector found user_oidc but produced a
						// malformed parse, treat as data.
						continue
					}
					unavailable = f
				case "inventory.nextcloud.oidc_provider":
					if !assessor.IsEvidenceLike(f) {
						continue
					}
					inspected = append(inspected, f)
					if country, _ := f.Attributes["country"].(string); !isEEACountry(country) {
						nonEEA = append(nonEEA, f)
					}
				}
			}
			if len(inspected) == 0 {
				v := "no inventory.nextcloud.oidc_provider finding — Nextcloud inspector did not run, or no IdP is configured"
				if unavailable.ProbeID != "" {
					alt, _ := unavailable.Attributes["alternative_app"].(string)
					if alt != "" {
						v = "user_oidc app not installed (alternative seen: " + alt + ") — install user_oidc for IdP-level scoring"
					} else {
						v = "user_oidc app not installed — install it for IdP-level scoring"
					}
				}
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: v}
			}
			if len(nonEEA) == 0 {
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d OIDC provider(s) — every IdP resolves to an EEA jurisdiction", len(inspected)),
					Evidence: sampleNextcloudEvidence(inspected),
				}
			}
			sort.Slice(nonEEA, func(i, j int) bool {
				return nonEEA[i].Subject < nonEEA[j].Subject
			})
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  nextcloudOIDCVerdict(nonEEA),
				Evidence: nextcloudFindingIDs(nonEEA),
			}
		},
	}
}

func nextcloudObjectstoreVerdict(hits []models.Finding) string {
	if len(hits) == 1 {
		country, _ := hits[0].Attributes["country"].(string)
		if country == "" {
			country = "unknown"
		}
		return fmt.Sprintf("objectstore %s resolves to %s — non-EEA jurisdiction", hits[0].Subject, country)
	}
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		country, _ := h.Attributes["country"].(string)
		names = append(names, fmt.Sprintf("%s (%s)", h.Subject, country))
	}
	return fmt.Sprintf("%d non-EEA objectstore backends: %s", len(hits), strings.Join(names, ", "))
}

func nextcloudOIDCVerdict(hits []models.Finding) string {
	if len(hits) == 1 {
		country, _ := hits[0].Attributes["country"].(string)
		host, _ := hits[0].Attributes["issuer_host"].(string)
		if country == "" {
			country = "unknown"
		}
		return fmt.Sprintf("OIDC provider %s (%s) resolves to %s — non-EEA jurisdiction", hits[0].Subject, host, country)
	}
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		country, _ := h.Attributes["country"].(string)
		names = append(names, fmt.Sprintf("%s (%s)", h.Subject, country))
	}
	return fmt.Sprintf("%d non-EEA OIDC providers: %s", len(hits), strings.Join(names, ", "))
}

// isEEACountry returns true for two-letter ISO codes in the
// EU-27 + Iceland / Liechtenstein / Norway set.
func isEEACountry(c string) bool {
	return eeaCountries[strings.ToUpper(c)]
}

// nextcloudFindingIDs returns the IDs of the given Findings,
// skipping empties so unit-tested rules (which set ID="" before
// persistence) do not panic.
func nextcloudFindingIDs(in []models.Finding) []string {
	out := make([]string, 0, len(in))
	for _, f := range in {
		if f.ID != "" {
			out = append(out, f.ID)
		}
	}
	return out
}

// sampleNextcloudEvidence caps soeverein-branch evidence at 10
// IDs — the negative-evidence pattern from add-host-side-scoring.
const nextcloudSampleEvidenceCap = 10

func sampleNextcloudEvidence(inspected []models.Finding) []string {
	limit := len(inspected)
	if limit > nextcloudSampleEvidenceCap {
		limit = nextcloudSampleEvidenceCap
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if inspected[i].ID != "" {
			out = append(out, inspected[i].ID)
		}
	}
	return out
}
