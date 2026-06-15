package wand

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// transitEUPath scores the observed network path to a target. The
// destination hop (where the target is actually hosted) is the strong,
// vantage-robust signal and drives the score: an EEA destination is
// soeverein, a non-EEA destination afhankelijk. Non-EEA *transit* hops
// are surfaced in the verdict for awareness but do not by themselves
// downgrade an EEA destination — transit is vantage-flavoured (see
// propose-transit-path-probe Q2).
func transitEUPath() assessor.Rule {
	return assessor.Rule{
		ID:        "wand.transit.eu_path",
		Dimension: models.DimensionJuridisch,
		Description: "The network path to the target terminates in an EEA " +
			"jurisdiction (where the target is hosted).",
		Rationale: "A traceroute shows where a target actually lives and " +
			"which jurisdictions its traffic crosses. The hosting " +
			"jurisdiction (the destination hop) is the load-bearing " +
			"sovereignty fact; transit through a non-EU carrier is " +
			"noted so an operator sees the full path, not just the " +
			"endpoint.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			// Per-hop findings that responded and carry a country.
			var hops []models.Finding
			for _, f := range findings {
				if f.ProbeID != "transit.hop" || !assessor.IsEvidenceLike(f) {
					continue
				}
				if c, _ := f.Attributes["country"].(string); c != "" {
					hops = append(hops, f)
				}
			}
			if len(hops) == 0 {
				return assessor.RuleResult{
					Score: models.ScoreOnbekend,
					Verdict: "no geo-attributed transit hops — the transit probe " +
						"did not run, no tracing tool was available, or no " +
						"GeoLite2 database is configured",
				}
			}
			// Destination = the highest-numbered responding hop.
			sort.SliceStable(hops, func(i, j int) bool {
				return hopNum(hops[i]) < hopNum(hops[j])
			})
			dest := hops[len(hops)-1]
			destCountry, _ := dest.Attributes["country"].(string)

			// Non-EEA transit hops (for the informational mention).
			var transit []string
			for _, h := range hops {
				c, _ := h.Attributes["country"].(string)
				if !isEEACountry(c) {
					transit = append(transit, transitHopLabel(h))
				}
			}

			if !isEEACountry(destCountry) {
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("target is hosted at %s — a non-EEA jurisdiction", destLabel(dest)),
					Evidence: []string{dest.ID},
				}
			}
			verdict := fmt.Sprintf("target is hosted at %s — an EEA jurisdiction", destLabel(dest))
			if len(transit) > 0 {
				verdict += "; path transits non-EEA hops: " + strings.Join(transit, ", ")
			}
			return assessor.RuleResult{
				Score:    models.ScoreSoeverein,
				Verdict:  verdict,
				Evidence: []string{dest.ID},
			}
		},
	}
}

func hopNum(f models.Finding) int {
	switch v := f.Attributes["hop"].(type) {
	case int:
		return v
	case float64: // JSON round-trip
		return int(v)
	}
	return 0
}

// destLabel renders the destination hop as "<org> (<country>, <ip>)",
// degrading gracefully when org is unknown.
func destLabel(f models.Finding) string {
	ip, _ := f.Attributes["ip"].(string)
	country, _ := f.Attributes["country"].(string)
	org, _ := f.Attributes["organisation"].(string)
	if org != "" {
		return fmt.Sprintf("%s (%s, %s)", org, country, ip)
	}
	return fmt.Sprintf("%s (%s)", ip, country)
}

func transitHopLabel(f models.Finding) string {
	country, _ := f.Attributes["country"].(string)
	org, _ := f.Attributes["organisation"].(string)
	if org != "" {
		return fmt.Sprintf("%s (%s)", org, country)
	}
	ip, _ := f.Attributes["ip"].(string)
	return fmt.Sprintf("%s (%s)", ip, country)
}
