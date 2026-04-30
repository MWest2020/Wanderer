package ui

import (
	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/internal/assessor/eucsf"
)

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
