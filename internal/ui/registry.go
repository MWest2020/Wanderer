package ui

import (
	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/eucsf"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
)

// CataloguedRule is one row of the Reporting catalogue: enough
// metadata to describe what the rule measures without any
// scoring data. Wraps the underlying assessor.Rule with the
// framework key the UI needs for the per-rule detail link.
type CataloguedRule struct {
	Framework string
	Rule      assessor.Rule
}

// ListAllRules returns every rule from every registered pack,
// ordered by framework (wand > eucsf > alphabetical), then by
// dimension, then by rule ID. Used by the /ui/reporting
// catalogue page; cheap enough to call per-request because the
// rule packs are small in-memory slices.
func ListAllRules() []CataloguedRule {
	out := make([]CataloguedRule, 0, 32)
	for _, r := range wand.DefaultRules() {
		out = append(out, CataloguedRule{Framework: "wand", Rule: r})
	}
	for _, r := range eucsf.DefaultRules() {
		out = append(out, CataloguedRule{Framework: "eucsf", Rule: r})
	}
	// Stable ordering: framework first (wand before eucsf), then
	// dimension (juridisch / data_ai / technologie / operationeel /
	// mens — keep declaration order), then rule ID alphabetical.
	rank := func(fw string) int {
		switch fw {
		case "wand":
			return 0
		case "eucsf":
			return 1
		default:
			return 2
		}
	}
	// Sort using a stable algorithm so within-pack declaration
	// order survives.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if rank(out[j-1].Framework) > rank(out[j].Framework) ||
				(rank(out[j-1].Framework) == rank(out[j].Framework) && out[j-1].Rule.ID > out[j].Rule.ID) {
				out[j-1], out[j] = out[j], out[j-1]
			} else {
				break
			}
		}
	}
	return out
}

// lookupRule resolves a (framework, criterium-ID) pair to the live
// assessor.Rule from the matching rule pack. Returns ok=false when
// the framework is unknown or the rule has been retired since the
// Assessment was persisted; the caller is expected to render a
// "rule retired" placeholder in that case so the historical
// verdict + evidence stay visible.
//
// The legacy `dictu` framework key is accepted as a deprecated
// alias for `wand` for one release after ADR-0011 (the rename) so
// any DB row that bypassed the schema migration still renders.
func lookupRule(framework, criteriumID string) (assessor.Rule, bool) {
	var rules []assessor.Rule
	switch framework {
	case "wand", "dictu":
		rules = wand.DefaultRules()
	case "eucsf":
		rules = eucsf.DefaultRules()
	default:
		return assessor.Rule{}, false
	}
	for _, r := range rules {
		if r.ID == criteriumID {
			return r, true
		}
	}
	return assessor.Rule{}, false
}
