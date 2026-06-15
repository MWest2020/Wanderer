package ui

import "github.com/MWest2020/wanderer/pkg/models"

// Flow is one row of the Sovereignty overview — a single "what goes
// where" statement synthesised from an already-scored rule. Label is
// the flow category (Hosting, Mail, …); Verdict is the rule's observed
// statement (e.g. "mx hosts in US (outside EEA)"); Score drives the
// colour pill.
type Flow struct {
	Label   string
	Verdict string
	Score   string
}

// flowRule maps a wand rule ID to its flow label + display order. The
// overview is pure presentation: it re-groups signals the rule pack
// already produced into the org/host-as-the-spil-in-the-web picture
// ("your service lives here, its mail goes there, its DNS is run by …").
var flowRules = []struct {
	id, label string
}{
	{"wand.juridisch.apex_ip_eea", "Hosting"},
	{"wand.juridisch.mx_vendor_jurisdiction", "Mail"},
	{"wand.juridisch.ns_vendor_jurisdiction", "DNS"},
	{"wand.transit.eu_path", "Transit path"},
	{"wand.technologie.no_us_hyperscaler", "CDN / hyperscaler"},
	{"wand.technologie.third_parties_eea", "Third parties"},
}

// SovereigntyFlows synthesises the overview from a scan's assessments.
// It reads the per-rule rationales (which already carry the observed
// verdict + score) and emits one Flow per known flow rule, in a fixed
// order. Rules that did not fire (no rationale) are omitted. No EEA
// logic lives here — the flows mirror what the rule pack already scored.
func SovereigntyFlows(assessments []models.Assessment) []Flow {
	// Index the latest rationale per rule ID across all frameworks.
	type rv struct {
		verdict string
		score   models.Score
	}
	byRule := map[string]rv{}
	for _, a := range assessments {
		for _, d := range a.Dimensions {
			for _, r := range d.Rationale {
				byRule[r.CriteriumID] = rv{verdict: r.Verdict, score: r.Score}
			}
		}
	}
	var flows []Flow
	for _, fr := range flowRules {
		r, ok := byRule[fr.id]
		if !ok {
			continue
		}
		flows = append(flows, Flow{
			Label:   fr.label,
			Verdict: r.verdict,
			Score:   string(r.score),
		})
	}
	return flows
}
