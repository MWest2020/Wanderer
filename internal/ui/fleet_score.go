package ui

import (
	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// FleetScore is one domain's row on the fleet screen (proposal.md
// "Niet ja/nee, maar x/n"): X sovereignty questions answered soeverein
// or voldoende out of N that could be answered at all, with the
// questions that scored onbekend kept apart — they never count toward
// N and never count as a pass. Worst names the heaviest open finding
// (proposal.md "toont het vlootscherm naast de score altijd de
// zwaarste openstaande bevinding") so a high score never hides an easy
// gap; it is empty when nothing scored afhankelijk.
type FleetScore struct {
	X          int
	N          int
	Unanswered int

	WorstFlow    string
	WorstVerdict string
}

// BuildFleetScore converts a scan's assessments into the fleet
// screen's x/n score (spec.md "Het vlootscherm scoort x van n, niet ja
// of nee"). It reuses classifyFlows — the same classification
// BuildAnswerVerdict builds its headline from — instead of re-deriving
// a second count of what's answered, unanswered, or heaviest, so the
// two views can't diverge on the same scan.
func BuildFleetScore(assessments []models.Assessment) FleetScore {
	flows := SovereigntyFlows(assessments)
	afhankelijk, unanswered, answered := classifyFlows(flows)

	score := FleetScore{
		X:          answered,
		N:          answered + len(afhankelijk),
		Unanswered: unanswered,
	}
	if len(afhankelijk) > 0 {
		score.WorstFlow = afhankelijk[0].Label
		score.WorstVerdict = afhankelijk[0].Verdict
	}
	return score
}

// FleetSummary is the organisation-wide vlootscore (proposal.md "Laag 1
// — de vloot", spec.md "De vloot is de eerste laag"): x/n summed over
// every domain's latest voltooide scan, that same scan's onbeantwoord
// count summed apart, how many domains have no voltooide scan at all,
// how many domains scored niet-soeverein, the three rules costing the
// fleet the most points, and the distribution per stroom.
type FleetSummary struct {
	X          int
	N          int
	Unanswered int

	DomainsWithoutScan int
	NotSovereign       int

	TopRules []ConcernRow
	Flows    []FlowRollup
}

// BuildFleetSummary aggregates an organisation's domains into the
// vloot-brede score. Per domain it runs the same BuildFleetScore /
// classifyFlows a single domain's answer page already uses — the
// proposal's valkuil ("een goede score mag een slecht domein niet
// verbergen") requires the fleet and the domain to count the same way,
// not a third scheme layered on top — and sums X, N and Unanswered
// across domains. TopRules reuses TopConcerns (aggregate.go): it
// already computes, per rule, the distinct domains that scored
// afhankelijk, sorted worst-first — exactly "de regels die de meeste
// punten kosten" — so it is called here capped at three rather than
// reimplemented. Flows reuses SovereigntyFlowRollup the same way for
// the per-stroom distribution ("Mail: 3 van 5 buiten de EER").
// RuleSummary/RuleTargetRows (aggregate.go) were not reused: they
// report every score bucket per rule and a per-target drill-down for
// /ui/reporting, a different granularity than "top three that cost the
// fleet points".
//
// Only a domain whose latest scan is "voltooid" — Complete or Partial,
// the same definition demo.go's latestCompletedScan documents — counts
// toward X, N, Unanswered, TopRules or Flows. A domain still scanning,
// failed outright, or never scanned contributes nothing to those and
// is counted once in DomainsWithoutScan instead, so an organisation
// mid-rollout can't drag its own score toward zero — nor can it quietly
// vanish from the picture.
func BuildFleetSummary(snaps []TargetSnapshot, ruleLookup func(framework, criteriumID string) (assessor.Rule, bool)) FleetSummary {
	var out FleetSummary
	scored := make([]TargetSnapshot, 0, len(snaps))
	for _, s := range snaps {
		switch models.ScanStatus(s.LastStatus) {
		case models.ScanStatusComplete, models.ScanStatusPartial:
		default:
			out.DomainsWithoutScan++
			continue
		}
		scored = append(scored, s)

		assessments := make([]models.Assessment, 0, len(s.Assessments))
		for _, a := range s.Assessments {
			assessments = append(assessments, a)
		}
		fs := BuildFleetScore(assessments)
		out.X += fs.X
		out.N += fs.N
		out.Unanswered += fs.Unanswered
		if fs.WorstFlow != "" {
			out.NotSovereign++
		}
	}
	out.TopRules = TopConcerns(scored, ruleLookup, 3)
	out.Flows = SovereigntyFlowRollup(scored)
	return out
}
