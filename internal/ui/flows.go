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

// FlowRollup is one flow category aggregated across an organisation's
// targets: how many were assessed for it and how many landed
// afhankelijk (the actionable count), with the worst score reached for
// the pill colour. It turns the per-scan overview into the org-as-the-
// spider-in-the-web posture ("across your services, mail is the weak
// spot").
type FlowRollup struct {
	Label       string
	Total       int
	Afhankelijk int
	Worst       string
}

// SovereigntyFlowRollup aggregates the per-target flows across a set of
// snapshots (an organisation's latest scans) into one row per flow
// category. Categories no target was assessed for are omitted.
func SovereigntyFlowRollup(snaps []TargetSnapshot) []FlowRollup {
	type acc struct {
		total, afh int
		worst      models.Score
	}
	byLabel := map[string]*acc{}
	for _, s := range snaps {
		assessments := make([]models.Assessment, 0, len(s.Assessments))
		for _, a := range s.Assessments {
			assessments = append(assessments, a)
		}
		for _, f := range SovereigntyFlows(assessments) {
			a := byLabel[f.Label]
			if a == nil {
				a = &acc{}
				byLabel[f.Label] = a
			}
			a.total++
			score := models.Score(f.Score)
			if score == models.ScoreAfhankelijk {
				a.afh++
			}
			if worseScore(score, a.worst) {
				a.worst = score
			}
		}
	}
	var out []FlowRollup
	for _, fr := range flowRules { // fixed order
		a := byLabel[fr.label]
		if a == nil {
			continue
		}
		out = append(out, FlowRollup{
			Label:       fr.label,
			Total:       a.total,
			Afhankelijk: a.afh,
			Worst:       string(a.worst),
		})
	}
	return out
}

// worseScore reports whether candidate is a worse (less sovereign)
// score than current, treating onbekend as the mildest so a genuine
// afhankelijk/voldoende always wins the pill. Empty current loses.
func worseScore(candidate, current models.Score) bool {
	return scoreSeverity(candidate) > scoreSeverity(current)
}

func scoreSeverity(s models.Score) int {
	switch s {
	case models.ScoreAfhankelijk:
		return 3
	case models.ScoreVoldoende:
		return 2
	case models.ScoreSoeverein:
		return 1
	default: // onbekend / empty
		return 0
	}
}
