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
	if !acc.NotApplicable {
		t.Errorf("want NotApplicable explicitly set true when every rule is structural")
	}
}

// TestScoreDimension_NotApplicableExplicitFlag pins task 7.4: a
// dimension whose every rationale is structural is marked not
// applicable via an explicit field, not only derivable by a caller
// re-scanning Rationale for reason class. A dimension with no rules at
// all (the pre-existing "no rule pack for this dimension" case) is a
// different situation and must NOT be flagged NotApplicable.
func TestScoreDimension_NotApplicableExplicitFlag(t *testing.T) {
	got := Assess(nil, []Rule{
		ruleAlways("m.1", models.DimensionMens, RuleResult{
			Score: models.ScoreOnbekend, Verdict: "hit", Evidence: []string{"f1"},
		}),
	})
	mens := findDim(t, got, models.DimensionMens)
	if mens.NotApplicable {
		t.Errorf("evidenced dimension must not be NotApplicable")
	}
	juridisch := findDim(t, got, models.DimensionJuridisch)
	if juridisch.NotApplicable {
		t.Errorf("dimension with zero registered rules must not be NotApplicable (it is simply unaddressed, not structurally inapplicable)")
	}
}

// TestScoreDimension_ReasonForcesOnbekend pins task 7.4's engine fix:
// a rationale with a reason always scores onbekend, whatever the rule
// itself returned — run 01 passed res.Score through unchanged, which
// let a buggy rule leak a non-onbekend score alongside a reason.
func TestScoreDimension_ReasonForcesOnbekend(t *testing.T) {
	rules := []Rule{
		ruleAlways("a.1", models.DimensionAccountability, RuleResult{
			// A rule bug: scores soeverein while also carrying a
			// reason. The engine must not trust this — it forces
			// onbekend regardless of the class the reason belongs to.
			Score: models.ScoreSoeverein, Verdict: "buggy", Evidence: []string{"f1"}, Reason: ReasonProbeUnavailable,
		}),
	}
	got := Assess(nil, rules)
	acc := findDim(t, got, models.DimensionAccountability)
	if len(acc.Rationale) != 1 {
		t.Fatalf("want 1 rationale, got %d", len(acc.Rationale))
	}
	if acc.Rationale[0].Score != models.ScoreOnbekend {
		t.Errorf("rationale.Score = %s, want onbekend forced by the reason", acc.Rationale[0].Score)
	}
	if acc.Rationale[0].Reason != ReasonProbeUnavailable {
		t.Errorf("reason not preserved: got %q", acc.Rationale[0].Reason)
	}
}

// TestScoreDimension_OldAssessmentReasonlessUnaffected pins the
// "keep old assessments loading unchanged" requirement: a rationale
// with no reason is passed through exactly as returned, untouched by
// the new forcing behaviour.
func TestScoreDimension_OldAssessmentReasonlessUnaffected(t *testing.T) {
	rules := []Rule{
		ruleAlways("j.1", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "v1", Evidence: []string{"f1"},
		}),
	}
	got := Assess(nil, rules)
	jur := findDim(t, got, models.DimensionJuridisch)
	if jur.Rationale[0].Score != models.ScoreVoldoende {
		t.Errorf("reasonless rationale.Score = %s, want voldoende unchanged", jur.Rationale[0].Score)
	}
	if jur.Rationale[0].Reason != "" {
		t.Errorf("reasonless rationale.Reason = %q, want empty", jur.Rationale[0].Reason)
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

// TestScoreDimension_HandelingExposedForNonSoevereinOnly pins task 3.1:
// the engine copies RuleResult.Handeling onto the Rationale whenever
// the final score is not soeverein, and drops it (leaves it empty)
// when the rule scored soeverein — even if the rule itself set
// Handeling on the result, e.g. because a rule pack sets it
// unconditionally (see wand.withHandeling) and leaves the decision to
// the engine.
func TestScoreDimension_HandelingExposedForNonSoevereinOnly(t *testing.T) {
	rules := []Rule{
		ruleAlways("j.fails", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreAfhankelijk, Verdict: "v1", Evidence: []string{"f1"},
			Handeling: "doe iets aan {domein}",
		}),
		ruleAlways("j.ok", models.DimensionJuridisch, RuleResult{
			Score: models.ScoreSoeverein, Verdict: "v2", Evidence: []string{"f2"},
			Handeling: "doe iets aan {domein}",
		}),
	}
	got := Assess(nil, rules)
	jur := findDim(t, got, models.DimensionJuridisch)
	if len(jur.Rationale) != 2 {
		t.Fatalf("want 2 rationale, got %d", len(jur.Rationale))
	}
	byID := map[string]models.Rationale{}
	for _, r := range jur.Rationale {
		byID[r.CriteriumID] = r
	}
	if byID["j.fails"].Handeling == "" {
		t.Errorf("j.fails: want Handeling carried over for a non-soeverein score, got empty")
	}
	if byID["j.ok"].Handeling != "" {
		t.Errorf("j.ok: want Handeling empty for a soeverein score, got %q", byID["j.ok"].Handeling)
	}
}

// TestScoreDimension_HandelingOmittedForNoEvidence pins that a rule
// with no evidence (forced to onbekend) still surfaces its Handeling —
// "niet soeverein" covers onbekend too, per task 3.1.
func TestScoreDimension_HandelingOmittedForNoEvidence(t *testing.T) {
	rules := []Rule{
		ruleAlways("o.gap", models.DimensionOperationeel, RuleResult{
			Score: models.ScoreVoldoende, Verdict: "no evidence though",
			Handeling: "doe iets aan {domein}",
		}),
	}
	got := Assess(nil, rules)
	op := findDim(t, got, models.DimensionOperationeel)
	if op.Rationale[0].Score != models.ScoreOnbekend {
		t.Fatalf("want score forced to onbekend, got %s", op.Rationale[0].Score)
	}
	if op.Rationale[0].Handeling == "" {
		t.Errorf("want Handeling carried over for the forced-onbekend no-evidence case")
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
