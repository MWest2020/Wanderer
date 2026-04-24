package assessor

import (
	"sort"

	"github.com/MWest2020/wanderer/pkg/models"
)

// DICTUDimensions is the canonical ordering of DICTU dimensions in
// every Assessment. Stable order keeps reports diffable.
var DICTUDimensions = []models.DimensionHint{
	models.DimensionJuridisch,
	models.DimensionTechnologie,
	models.DimensionDataAI,
	models.DimensionOperationeel,
	models.DimensionMens,
}

// Assess runs every rule against findings and aggregates the results
// into per-dimension scores. The returned slice always has one entry
// per DICTUDimensions entry, in that order, even for dimensions with
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

	out := make([]models.DimensionScore, 0, len(DICTUDimensions))
	for _, dim := range DICTUDimensions {
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
	worst := models.Score("")
	for _, r := range rules {
		res := safeMatch(r, findings)
		evidence := append([]string(nil), res.Evidence...)
		rat := models.Rationale{
			CriteriumID: r.ID,
			Verdict:     res.Verdict,
			Score:       res.Score,
			Evidence:    evidence,
		}
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
		ds.Completeness = models.CompletenessIncomplete
		ds.Score = models.ScoreOnbekend
	case evidenced == len(rules):
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
