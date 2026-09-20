// flow_answers.go renders the seven sovereignty flows (flowRules) as
// answer-sheet rows — the same AccountabilityAnswer shape and the same
// "answer-rows" template partial the accountability section already
// uses (design.md "UI direction"; run 04 task 4.1: "Hergebruik wat de
// accountability-sectie al doet; bouw geen tweede weergave"). Unlike
// accountability's per-outcome Dutch verdict copy, a flow's Verdict is
// the rule's own observed-fact sentence (Rationale.Verdict) — this run
// adds no new observation or scoring (proposal.md "Out: the assessor,
// the rules, the probes").
package ui

import "github.com/MWest2020/wanderer/pkg/models"

// BuildFlowAnswers renders the seven sovereignty flows into
// answer-sheet rows, in flowRules' fixed order. A flow whose rule
// never fired (no rationale for its ID across the given assessments)
// is omitted, mirroring SovereigntyFlows.
func BuildFlowAnswers(assessments []models.Assessment, findingsByID map[string]models.Finding) []AccountabilityAnswer {
	byRule := map[string]models.Rationale{}
	for _, a := range assessments {
		for _, d := range a.Dimensions {
			for _, r := range d.Rationale {
				byRule[r.CriteriumID] = r
			}
		}
	}
	var out []AccountabilityAnswer
	for _, fr := range flowRules {
		rat, ok := byRule[fr.id]
		if !ok {
			continue
		}
		_, label, class := accountabilityOutcome(rat)
		row := AccountabilityAnswer{
			RuleID:      fr.id,
			Question:    fr.question,
			AnswerLabel: label,
			AnswerClass: class,
			Verdict:     rat.Verdict,
		}
		if rule, ok := lookupRule("wand", fr.id); ok {
			row.RuleDescription = rule.Description
		}
		for _, id := range rat.Evidence {
			f, ok := findingsByID[id]
			if !ok {
				continue
			}
			row.Evidence = append(row.Evidence, accountabilityEvidenceFor(f))
		}
		if len(row.Evidence) == 0 {
			row.NoEvidenceNote = rat.Verdict
		}
		out = append(out, row)
	}
	return out
}
