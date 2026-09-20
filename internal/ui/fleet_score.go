package ui

import "github.com/MWest2020/wanderer/pkg/models"

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
