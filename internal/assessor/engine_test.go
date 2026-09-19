package assessor

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

// ruleAlways returns a Rule that always evaluates to the given
// RuleResult. Helper for table-driven tests.
func ruleAlways(id string, dim models.DimensionHint, res RuleResult) Rule {
	return Rule{
		ID:        id,
		Dimension: dim,
		Match:     func(_ []models.Finding) RuleResult { return res },
	}
}

func TestAssess_AllDimensionsEmitted(t *testing.T) {
	got := Assess(nil, nil)
	if len(got) != len(WandDimensions) {
		t.Fatalf("want %d dimensions, got %d", len(WandDimensions), len(got))
	}
	for i, want := range WandDimensions {
		if got[i].Dimension != want {
			t.Errorf("dim[%d] = %s, want %s", i, got[i].Dimension, want)
		}
		if got[i].Score != models.ScoreOnbekend {
			t.Errorf("dim[%d] score = %s, want onbekend", i, got[i].Score)
		}
		if got[i].Completeness != models.CompletenessIncomplete {
			t.Errorf("dim[%d] completeness = %s, want incomplete", i, got[i].Completeness)
		}
	}
}

func TestAssess_CompleteDimension(t *testing.T) {
	rules := []Rule{
		ruleAlways("j.1", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "v1", Evidence: []string{"f1"},
		}),
		ruleAlways("j.2", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreAfhankelijk, Verdict: "v2", Evidence: []string{"f2"},
		}),
	}
	got := Assess(nil, rules)
	jur := findDim(t, got, models.DimensionJuridisch)
	if jur.Completeness != models.CompletenessComplete {
		t.Errorf("want complete, got %s", jur.Completeness)
	}
	if jur.Score != models.ScoreAfhankelijk {
		t.Errorf("worst-wins: want afhankelijk, got %s", jur.Score)
	}
	if len(jur.Rationale) != 2 {
		t.Errorf("want 2 rationale, got %d", len(jur.Rationale))
	}
}

func TestAssess_PartialDimension(t *testing.T) {
	rules := []Rule{
		ruleAlways("t.1", models.DimensionTechnologie, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "hit", Evidence: []string{"f1"},
		}),
		ruleAlways("t.2", models.DimensionTechnologie, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "missed",
		}),
	}
	got := Assess(nil, rules)
	tec := findDim(t, got, models.DimensionTechnologie)
	if tec.Completeness != models.CompletenessPartial {
		t.Errorf("want partial, got %s", tec.Completeness)
	}
	if tec.Score != models.ScoreVoldoende {
		t.Errorf("want voldoende (only evidenced rule), got %s", tec.Score)
	}
	if len(tec.Rationale) != 2 {
		t.Errorf("want 2 rationale, got %d", len(tec.Rationale))
	}
}

func TestAssess_IncompleteDimension(t *testing.T) {
	rules := []Rule{
		ruleAlways("o.1", models.DimensionOperationeel, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "missed",
		}),
	}
	got := Assess(nil, rules)
	op := findDim(t, got, models.DimensionOperationeel)
	if op.Completeness != models.CompletenessIncomplete {
		t.Errorf("want incomplete, got %s", op.Completeness)
	}
	if op.Score != models.ScoreOnbekend {
		t.Errorf("want onbekend, got %s", op.Score)
	}
}

func TestScoreDimension_StructuralReasonExcludedFromWorstAndCompleteness(t *testing.T) {
	rules := []Rule{
		ruleAlways("a.1", models.DimensionAccountability, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "registry redacts .nl", Reason: ReasonRegistryRedacted,
		}),
		ruleAlways("a.2", models.DimensionAccountability, RuleResult{
			Score: models.ScoreSoeverein, Verdict: "ok", Evidence: []string{"f1"},
		}),
	}
	got := Assess(nil, rules)
	acc := findDim(t, got, models.DimensionAccountability)
	if acc.Completeness != models.CompletenessComplete {
		t.Errorf("want complete (structural rule excluded from denominator), got %s", acc.Completeness)
	}
	if acc.Score != models.ScoreSoeverein {
		t.Errorf("want soeverein (structural rule excluded from worst-score), got %s", acc.Score)
	}
	if len(acc.Rationale) != 2 {
		t.Fatalf("want both rationale entries retained, got %d", len(acc.Rationale))
	}
	if acc.Rationale[0].Reason != ReasonRegistryRedacted {
		t.Errorf("want reason preserved on rationale, got %q", acc.Rationale[0].Reason)
	}
}

func TestScoreDimension_AllStructuralIsNotApplicable(t *testing.T) {
	rules := []Rule{
		ruleAlways("a.1", models.DimensionAccountability, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "n.v.t. 1", Reason: ReasonRegistryRedacted,
		}),
		ruleAlways("a.2", models.DimensionAccountability, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "n.v.t. 2", Reason: ReasonNotPublishedByRegistry,
		}),
	}
	got := Assess(nil, rules)
	acc := findDim(t, got, models.DimensionAccountability)
	if acc.Score != models.ScoreOnbekend {
		t.Errorf("want onbekend, got %s", acc.Score)
	}
	if acc.Completeness != models.CompletenessIncomplete {
		t.Errorf("want incomplete/n.v.t. when every rule is structural, got %s", acc.Completeness)
	}
	if len(acc.Rationale) != 2 {
		t.Errorf("want structural rationale entries retained for display, got %d", len(acc.Rationale))
	}
}

func TestScoreDimension_GapReasonCountsAsHole(t *testing.T) {
	rules := []Rule{
		ruleAlways("a.1", models.DimensionAccountability, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "RDAP unavailable", Reason: ReasonProbeUnavailable,
		}),
		ruleAlways("a.2", models.DimensionAccountability, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "ok", Evidence: []string{"f1"},
		}),
	}
	got := Assess(nil, rules)
	acc := findDim(t, got, models.DimensionAccountability)
	if acc.Completeness != models.CompletenessPartial {
		t.Errorf("want partial (gap reason still counts as a hole), got %s", acc.Completeness)
	}
	if acc.Score != models.ScoreVoldoende {
		t.Errorf("want voldoende (only evidenced rule wins worst-score), got %s", acc.Score)
	}
}

func TestScoreDimension_UnregisteredReasonCodePanics(t *testing.T) {
	rules := []Rule{
		ruleAlways("a.1", models.DimensionAccountability, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "bogus", Reason: "totally_made_up",
		}),
	}
	defer func() {
		if recover() == nil {
			t.Fatal("want panic for an unregistered reason code")
		}
	}()
	Assess(nil, rules)
}

func TestAssess_RulePanicIsContained(t *testing.T) {
	rules := []Rule{
		{
			ID:        "j.panic",
			Dimension: models.DimensionJuridisch,
			Match:     func(_ []models.Finding) RuleResult { panic("boom") },
		},
		ruleAlways("j.ok", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "ok", Evidence: []string{"f1"},
		}),
	}
	got := Assess(nil, rules)
	jur := findDim(t, got, models.DimensionJuridisch)
	if jur.Score != models.ScoreVoldoende {
		t.Errorf("panic should not take down the dimension; got %s", jur.Score)
	}
	if jur.Completeness != models.CompletenessPartial {
		t.Errorf("panic counts as no-evidence; want partial, got %s", jur.Completeness)
	}
}

func TestAssess_Deterministic(t *testing.T) {
	// Rules intentionally added out of order — the engine must emit
	// them in stable ID order.
	rules := []Rule{
		ruleAlways("j.b", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "b", Evidence: []string{"f2"},
		}),
		ruleAlways("j.a", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "a", Evidence: []string{"f1"},
		}),
	}
	a := Assess(nil, rules)
	b := Assess(nil, rules)
	jurA := findDim(t, a, models.DimensionJuridisch)
	jurB := findDim(t, b, models.DimensionJuridisch)
	if len(jurA.Rationale) != len(jurB.Rationale) {
		t.Fatalf("rationale count drifted between runs")
	}
	for i := range jurA.Rationale {
		if jurA.Rationale[i].CriteriumID != jurB.Rationale[i].CriteriumID {
			t.Errorf("order drift at %d: %s vs %s", i, jurA.Rationale[i].CriteriumID, jurB.Rationale[i].CriteriumID)
		}
	}
	if jurA.Rationale[0].CriteriumID != "j.a" {
		t.Errorf("want ID-sorted, got %s first", jurA.Rationale[0].CriteriumID)
	}
}

func findDim(t *testing.T, out []models.DimensionScore, dim models.DimensionHint) models.DimensionScore {
	t.Helper()
	for _, d := range out {
		if d.Dimension == dim {
			return d
		}
	}
	t.Fatalf("dimension %s not emitted", dim)
	return models.DimensionScore{}
}
