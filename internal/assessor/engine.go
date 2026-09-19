package assessor

import (
	"sort"

	"github.com/MWest2020/wanderer/pkg/models"
)

// WandDimensions is the canonical ordering of dimensions in every
// Assessment: the DICTU dimensions plus wand-native ones that extend
// beyond them (accountability has no DICTU counterpart — see
// ADR-0011). Stable order keeps reports diffable. Callers iterate
// this list; none may assume its length.
var WandDimensions = []models.DimensionHint{
	models.DimensionJuridisch,
	models.DimensionTechnologie,
	models.DimensionDataAI,
	models.DimensionOperationeel,
	models.DimensionMens,
	models.DimensionAccountability,
}

// Assess runs every rule against findings and aggregates the results
// into per-dimension scores. The returned slice always has one entry
// per WandDimensions entry, in that order, even for dimensions with
// no rules (those are emitted as incomplete with score onbekend).
func Assess(findings []models.Finding, rules []Rule) []models.DimensionScore {
	byDim := map[models.DimensionHint][]Rule{}
	for _, r := range rules {
		byDim[r.Dimension] = append(byDim[r.Dimension], r)
	}
	// Deterministic rule order within a dimension so two runs produce
	// the same Rationale ordering.
	for k := range byDim {
		sort.Slice(byDim[k], func(i, j int) bool {
			return byDim[k][i].ID < byDim[k][j].ID
		})
	}

	out := make([]models.DimensionScore, 0, len(WandDimensions))
	for _, dim := range WandDimensions {
		out = append(out, scoreDimension(dim, byDim[dim], findings))
	}
	return out
}

func scoreDimension(dim models.DimensionHint, rules []Rule, findings []models.Finding) models.DimensionScore {
	ds := models.DimensionScore{
		Dimension:    dim,
		Score:        models.ScoreOnbekend,
		Completeness: models.CompletenessIncomplete,
	}
	if len(rules) == 0 {
		return ds
	}

	evidenced := 0
	total := 0
	worst := models.Score("")
	for _, r := range rules {
		res := safeMatch(r, findings)
		evidence := append([]string(nil), res.Evidence...)
		rat := models.Rationale{
			CriteriumID: r.ID,
			Verdict:     res.Verdict,
			Score:       res.Score,
			Evidence:    evidence,
			Reason:      res.Reason,
		}
		if res.Reason != "" {
			class, _ := ReasonInfo(res.Reason)
			if class == ReasonStructural {
				// Not applicable to this target: excluded from both the
				// worst-score computation and the completeness
				// denominator, not merely from the evidence count. A
				// dimension whose every rule lands here is left with
				// total == 0 below and reports as not applicable.
				ds.Rationale = append(ds.Rationale, rat)
				continue
			}
			// ReasonGap falls through to the evidence check below: it
			// counts as a missing observation exactly like an
			// evidence-less rule does today.
		}
		total++
		if len(rat.Evidence) == 0 {
			// Rule had no evidence. Force Score to onbekend for the
			// rationale so downstream readers do not have to inspect
			// Evidence length themselves.
			rat.Score = models.ScoreOnbekend
			if rat.Verdict == "" {
				rat.Verdict = "no evidence — rule did not match"
			}
			ds.Rationale = append(ds.Rationale, rat)
			continue
		}
		evidenced++
		// Track the worst (lowest rank) evidence-backed score.
		if worst == "" || res.Score.Rank() < worst.Rank() {
			worst = res.Score
		}
		ds.Rationale = append(ds.Rationale, rat)
	}

	switch {
	case evidenced == 0:
		// Either nothing was evidenced, or every rule was structural
		// (total == 0) — both report as onbekend/incomplete, which
		// reads as "not applicable" when paired with an all-structural
		// Rationale, and is excluded from any overall score by callers
		// that already skip onbekend dimensions.
		ds.Completeness = models.CompletenessIncomplete
		ds.Score = models.ScoreOnbekend
	case evidenced == total:
		ds.Completeness = models.CompletenessComplete
		ds.Score = worst
	default:
		ds.Completeness = models.CompletenessPartial
		ds.Score = worst
	}
	return ds
}

// safeMatch invokes a Rule's Match function and normalises the result.
// Panics inside a rule are recovered and converted to a no-evidence
// result so a buggy rule cannot take down the whole assessment.
func safeMatch(r Rule, findings []models.Finding) (res RuleResult) {
	defer func() {
		if rec := recover(); rec != nil {
			res = RuleResult{
				Score:   models.ScoreOnbekend,
				Verdict: "rule panicked; skipped",
			}
		}
	}()
	res = r.Match(findings)
	// Normalise: a rule that returns empty-but-scored is treated as
	// no-evidence.
	if !res.Score.Valid() {
		res.Score = models.ScoreOnbekend
	}
	return res
}
