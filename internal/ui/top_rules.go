package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor/wand"
)

// TopRuleView is one row of the door's "kost de vloot de meeste punten"
// top 3 (design.md "The top 3 without the rationale"): the flow the rule
// belongs to and its handeling, in the vlootlaag's own language — never
// the rule's English Description/Rationale, which stays one click away
// on the rule's reporting page. FlowLabel and Handeling are both "" when
// the underlying rule has no flow or no handeling entry; Fallback then
// carries the rule ID instead, per design.md ("its ID is shown — never
// the rationale").
type TopRuleView struct {
	Link string

	FlowLabel string
	Handeling string
	Fallback  string

	TargetCount  int
	Total        int
	WidthPercent string
	BarLabel     string
}

// domeinParamRe matches the {domein} placeholder in a handeling
// sentence — the same placeholder fillParams substitutes, but here it
// is left out rather than filled, since the top 3 speaks about the
// fleet, not one domain (design.md "the rule's handeling with {domein}
// left out").
var domeinParamRe = regexp.MustCompile(`\{domein\}`)

// handelingWithoutDomain returns ruleID's Dutch remediation sentence
// with its {domein} placeholder removed and the resulting double space
// collapsed, so the sentence still reads as one sentence rather than
// leaving a visible gap. ok is false when ruleID has no handeling entry.
func handelingWithoutDomain(ruleID string) (string, bool) {
	h, ok := wand.HandelingFor(ruleID)
	if !ok {
		return "", false
	}
	blanked := domeinParamRe.ReplaceAllString(h, "")
	return strings.Join(strings.Fields(blanked), " "), true
}

// flowLabelForRule returns the Dutch stroom label for ruleID — the same
// seven flowRules the Sovereignty overview groups by — or "" when ruleID
// is not one of them.
func flowLabelForRule(ruleID string) string {
	for _, fr := range flowRules {
		if fr.id == ruleID {
			return fr.label
		}
	}
	return ""
}

// BuildTopRuleViews turns the fleet's top ConcernRows (aggregate.go's
// TopConcerns, the same rows the door's TopRules already carried) into
// the vlootlaag's top-3 shape: stroom + handeling + a bar of how many of
// how many domains it hits, with the rule ID as the only fallback when
// either is missing — spec.md "de top-3 ... SHALL in de taal van deze
// laag staan ... de onderbouwing van een regel hoort op de regelpagina,
// niet hier".
func BuildTopRuleViews(rows []ConcernRow) []TopRuleView {
	out := make([]TopRuleView, 0, len(rows))
	for _, r := range rows {
		v := TopRuleView{
			Link:        "/ui/reporting/" + r.Framework + "/" + r.CriteriumID,
			TargetCount: r.TargetCount,
			Total:       r.Total,
			BarLabel:    fmt.Sprintf("%d van %d domeinen", r.TargetCount, r.Total),
		}
		if r.Total > 0 {
			v.WidthPercent = fmt.Sprintf("%.2f%%", float64(r.TargetCount)/float64(r.Total)*100)
		}
		flowLabel := flowLabelForRule(r.CriteriumID)
		handeling, hasHandeling := handelingWithoutDomain(r.CriteriumID)
		if flowLabel == "" || !hasHandeling {
			v.Fallback = r.CriteriumID
		} else {
			v.FlowLabel = flowLabel
			v.Handeling = handeling
		}
		out = append(out, v)
	}
	return out
}
