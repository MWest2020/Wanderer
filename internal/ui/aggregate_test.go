package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestWorstScore_MixedDimensions(t *testing.T) {
	dims := []models.DimensionScore{
		{Dimension: models.DimensionJuridisch, Score: models.ScoreAfhankelijk},
		{Dimension: models.DimensionOperationeel, Score: models.ScoreSoeverein},
		{Dimension: models.DimensionDataAI, Score: models.ScoreOnbekend},
	}
	if got := WorstScore(dims); got != models.ScoreAfhankelijk {
		t.Errorf("WorstScore = %s, want afhankelijk", got)
	}
}

func TestWorstScore_AllOnbekend(t *testing.T) {
	dims := []models.DimensionScore{
		{Score: models.ScoreOnbekend},
		{Score: models.ScoreOnbekend},
	}
	if got := WorstScore(dims); got != models.ScoreOnbekend {
		t.Errorf("WorstScore = %s, want onbekend", got)
	}
}

func TestWorstScore_EmptySliceIsOnbekend(t *testing.T) {
	if got := WorstScore(nil); got != models.ScoreOnbekend {
		t.Errorf("WorstScore(nil) = %s, want onbekend", got)
	}
}

func TestWorstScore_OnbekendDoesNotDragDown(t *testing.T) {
	dims := []models.DimensionScore{
		{Score: models.ScoreVoldoende},
		{Score: models.ScoreOnbekend},
	}
	if got := WorstScore(dims); got != models.ScoreVoldoende {
		t.Errorf("WorstScore = %s, want voldoende (onbekend ignored)", got)
	}
}

func TestWorstScoreCovering_ListsOnlyRatedDimensions(t *testing.T) {
	dims := []models.DimensionScore{
		{Dimension: models.DimensionJuridisch, Score: models.ScoreAfhankelijk},
		{Dimension: models.DimensionOperationeel, Score: models.ScoreSoeverein},
		{Dimension: models.DimensionDataAI, Score: models.ScoreOnbekend},
		{Dimension: models.DimensionAccountability, Score: models.ScoreOnbekend, NotApplicable: true},
	}
	score, covered := WorstScoreCovering(dims)
	if score != models.ScoreAfhankelijk {
		t.Errorf("score = %s, want afhankelijk", score)
	}
	want := []string{"juridisch", "operationeel"}
	if len(covered) != len(want) || covered[0] != want[0] || covered[1] != want[1] {
		t.Errorf("covered = %v, want %v", covered, want)
	}
}

func TestWorstScoreCovering_AbsentDimensionIsNotCovered(t *testing.T) {
	// A scan predating the accountability dimension simply has no entry
	// for it — WorstScoreCovering must not invent one.
	dims := []models.DimensionScore{
		{Dimension: models.DimensionJuridisch, Score: models.ScoreSoeverein},
	}
	_, covered := WorstScoreCovering(dims)
	for _, c := range covered {
		if c == string(models.DimensionAccountability) {
			t.Fatalf("covered = %v, should not list an absent dimension", covered)
		}
	}
}

func TestAccountabilityPill_NotAssessedWhenDimensionAbsent(t *testing.T) {
	pill := AccountabilityPill([]models.DimensionScore{
		{Dimension: models.DimensionJuridisch, Score: models.ScoreSoeverein},
	})
	if pill.Label != "niet beoordeeld" || pill.Class != "unassessed" {
		t.Errorf("pill = %+v, want niet beoordeeld/unassessed", pill)
	}
}

func TestAccountabilityPill_NotApplicable(t *testing.T) {
	pill := AccountabilityPill([]models.DimensionScore{
		{Dimension: models.DimensionAccountability, Score: models.ScoreOnbekend, NotApplicable: true},
	})
	if pill.Label != "n.v.t." || pill.Class != "nvt" {
		t.Errorf("pill = %+v, want n.v.t./nvt", pill)
	}
}

func TestAccountabilityPill_ScoredDimension(t *testing.T) {
	pill := AccountabilityPill([]models.DimensionScore{
		{Dimension: models.DimensionAccountability, Score: models.ScoreAfhankelijk},
	})
	if pill.Label != "afhankelijk" || pill.Class != "afhankelijk" {
		t.Errorf("pill = %+v, want afhankelijk/afhankelijk", pill)
	}
}

func TestDimensionScannerWarnings_ScannerSubjectOnly(t *testing.T) {
	dim := models.DimensionScore{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.operationeel.variant_convergence", Score: models.ScoreOnbekend, Reason: assessor.ReasonScannerNoIPv6},
			{CriteriumID: "wand.accountability.registrant_identifiable", Score: models.ScoreOnbekend, Reason: assessor.ReasonRegistryRedacted},
		},
	}
	warnings := DimensionScannerWarnings(dim)
	if len(warnings) != 1 || warnings[0] == "" {
		t.Fatalf("warnings = %v, want exactly one scanner warning", warnings)
	}
	if !strings.Contains(warnings[0], "IPv6") {
		t.Errorf("warning %q does not mention IPv6", warnings[0])
	}
}

func TestPostureCounts_BucketsByFrameworkAndScore(t *testing.T) {
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Score: models.ScoreSoeverein}}},
		}},
		{TargetID: "t2", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Score: models.ScoreAfhankelijk}}},
		}},
		{TargetID: "t3", Assessments: map[string]models.Assessment{
			"wand":  {Dimensions: []models.DimensionScore{{Score: models.ScoreOnbekend}}},
			"eucsf": {Dimensions: []models.DimensionScore{{Score: models.ScoreSoeverein}}},
		}},
	}
	got := PostureCounts(snaps)
	if got["wand"][models.ScoreSoeverein] != 1 {
		t.Errorf("wand soeverein = %d, want 1", got["wand"][models.ScoreSoeverein])
	}
	if got["wand"][models.ScoreAfhankelijk] != 1 {
		t.Errorf("wand afhankelijk = %d, want 1", got["wand"][models.ScoreAfhankelijk])
	}
	if got["wand"][models.ScoreOnbekend] != 1 {
		t.Errorf("wand onbekend = %d, want 1", got["wand"][models.ScoreOnbekend])
	}
	if got["eucsf"][models.ScoreSoeverein] != 1 {
		t.Errorf("eucsf soeverein = %d, want 1", got["eucsf"][models.ScoreSoeverein])
	}
}

func TestTopConcerns_OneRuleManyFindingsCountsTargetOnce(t *testing.T) {
	// One target whose Assessment has the same rule firing
	// afhankelijk on two rationales. TargetCount must be 1.
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{
				{Rationale: []models.Rationale{
					{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
					{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
				}},
			}},
		}},
	}
	got := TopConcerns(snaps, nil, 5)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TargetCount != 1 {
		t.Errorf("TargetCount = %d, want 1", got[0].TargetCount)
	}
}

func TestTopConcerns_SameRuleManyTargetsCounts(t *testing.T) {
	// Same rule firing afhankelijk on five distinct targets.
	mkSnap := func(id string) TargetSnapshot {
		return TargetSnapshot{
			TargetID: id,
			Assessments: map[string]models.Assessment{
				"wand": {Dimensions: []models.DimensionScore{
					{Rationale: []models.Rationale{
						{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
					}},
				}},
			},
		}
	}
	snaps := []TargetSnapshot{mkSnap("t1"), mkSnap("t2"), mkSnap("t3"), mkSnap("t4"), mkSnap("t5")}
	got := TopConcerns(snaps, nil, 10)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TargetCount != 5 {
		t.Errorf("TargetCount = %d, want 5", got[0].TargetCount)
	}
}

func TestTopConcerns_LookupAttachesDescription(t *testing.T) {
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{
				{Rationale: []models.Rationale{
					{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
				}},
			}},
		}},
	}
	stub := func(_, id string) (assessor.Rule, bool) {
		return assessor.Rule{ID: id, Description: "desc", Rationale: "why"}, true
	}
	got := TopConcerns(snaps, stub, 5)
	if got[0].Description != "desc" {
		t.Errorf("Description not attached: %+v", got[0])
	}
	if got[0].Rationale != "why" {
		t.Errorf("Rationale not attached: %+v", got[0])
	}
}

func TestTopConcerns_NonAfhankelijkIgnored(t *testing.T) {
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{
				{Rationale: []models.Rationale{
					{CriteriumID: "x", Score: models.ScoreSoeverein},
					{CriteriumID: "y", Score: models.ScoreOnbekend},
				}},
			}},
		}},
	}
	if got := TopConcerns(snaps, nil, 5); len(got) != 0 {
		t.Errorf("expected zero concerns, got %d (%+v)", len(got), got)
	}
}

func TestTopConcerns_TotalIsDenominatorForFailRate(t *testing.T) {
	// run 03 task 2.4: "faalt op X van Y domeinen" needs Y — how many
	// domains this rule was evaluated against, not just how many
	// failed. Three targets: two afhankelijk, one soeverein — Total
	// must be 3, TargetCount must stay 2 (the count itself is
	// untouched, still afhankelijk-only).
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{
				{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
			}}}},
		}},
		{TargetID: "t2", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{
				{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk},
			}}}},
		}},
		{TargetID: "t3", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{
				{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreSoeverein},
			}}}},
		}},
	}
	got := TopConcerns(snaps, nil, 5)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TargetCount != 2 {
		t.Errorf("TargetCount = %d, want 2 (afhankelijk only)", got[0].TargetCount)
	}
	if got[0].Total != 3 {
		t.Errorf("Total = %d, want 3 (every target the rule fired against)", got[0].Total)
	}
}

func TestTopConcerns_MergesSameConcernAcrossFrameworks(t *testing.T) {
	// run 03 task 2.5: wand's cert_issuer_eea and eucsf's
	// cert_issuer_eu flag the same real-world gap. A target failing
	// under both frameworks must count once, and the merged row must
	// name both frameworks rather than appear twice.
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand":  {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreAfhankelijk}}}}},
			"eucsf": {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{{CriteriumID: "eucsf.sov2.cert_issuer_eu", Score: models.ScoreAfhankelijk}}}}},
		}},
		{TargetID: "t2", Assessments: map[string]models.Assessment{
			"eucsf": {Dimensions: []models.DimensionScore{{Rationale: []models.Rationale{{CriteriumID: "eucsf.sov2.cert_issuer_eu", Score: models.ScoreAfhankelijk}}}}},
		}},
	}
	got := TopConcerns(snaps, nil, 5)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 merged row, got %+v", len(got), got)
	}
	if got[0].TargetCount != 2 {
		t.Errorf("TargetCount = %d, want 2 — t1 counts once despite failing both frameworks", got[0].TargetCount)
	}
	if got[0].Framework != "wand" {
		t.Errorf("Framework = %q, want wand (rank-first, so the reporting link resolves)", got[0].Framework)
	}
	wantFrameworks := []string{"wand", "eucsf"}
	if len(got[0].Frameworks) != len(wantFrameworks) {
		t.Fatalf("Frameworks = %v, want %v", got[0].Frameworks, wantFrameworks)
	}
	for i, fw := range wantFrameworks {
		if got[0].Frameworks[i] != fw {
			t.Errorf("Frameworks[%d] = %q, want %q", i, got[0].Frameworks[i], fw)
		}
	}
}

func TestDashboardTemplate_TopRulesShowsProblemNotGoal(t *testing.T) {
	// run 03 task 2.4/2.5: the rendered top-3 list must read "faalt op
	// X van Y domeinen" (the defect, with a denominator) rather than
	// the rule's goal-state Description with a bare count, and a
	// concern merged across two frameworks must name both once.
	tmpl, err := Templates()
	if err != nil {
		t.Fatalf("Templates: %v", err)
	}
	view := doorView{
		HasFleet: true,
		TopRules: []ConcernRow{{
			Framework:   "wand",
			CriteriumID: "wand.juridisch.cert_issuer_eea",
			Description: "TLS certificate issued by an authority in the EEA.",
			Rationale:   "Certificate Authorities can revoke or refuse to renew certificates.",
			TargetCount: 2,
			Total:       3,
			Frameworks:  []string{"wand", "eucsf"},
		}},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "dashboard.tmpl", view); err != nil {
		t.Fatalf("execute: %v", err)
	}
	body := buf.String()
	if strings.Contains(body, "TLS certificate issued by an authority in the EEA.") {
		t.Errorf("goal-state Description must not render; body:\n%s", body)
	}
	if !strings.Contains(body, "Certificate Authorities can revoke or refuse to renew certificates.") {
		t.Errorf("problem-framed Rationale must render; body:\n%s", body)
	}
	if !strings.Contains(body, "faalt op 2 van 3 domeinen") {
		t.Errorf("expected 'faalt op 2 van 3 domeinen'; body:\n%s", body)
	}
	if !strings.Contains(body, "(wand, eucsf)") {
		t.Errorf("expected both frameworks named once; body:\n%s", body)
	}
}

func TestRecentActivity_OrderedAndCapped(t *testing.T) {
	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	scans := []store.ScanRow{
		{ID: "s1", Domain: "a.example", StartedAt: t0.Add(1 * time.Hour), Status: "complete"},
		{ID: "s2", Domain: "b.example", StartedAt: t0.Add(2 * time.Hour), Status: "complete"},
		{ID: "s3", Domain: "c.example", StartedAt: t0.Add(3 * time.Hour), Status: "complete"},
		{ID: "s4", Domain: "d.example", StartedAt: t0.Add(4 * time.Hour), Status: "complete"},
		{ID: "s5", Domain: "e.example", StartedAt: t0.Add(5 * time.Hour), Status: "complete"},
		{ID: "s6", Domain: "f.example", StartedAt: t0.Add(6 * time.Hour), Status: "complete"},
		{ID: "s7", Domain: "g.example", StartedAt: t0.Add(7 * time.Hour), Status: "complete"},
	}
	got := RecentActivity(scans, nil, 5)
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	if got[0].ScanID != "s7" {
		t.Errorf("expected newest first; got[0] = %s", got[0].ScanID)
	}
	if got[4].ScanID != "s3" {
		t.Errorf("expected oldest of the top 5 last; got[4] = %s", got[4].ScanID)
	}
}

func TestRecentActivity_HasAssessmentFlag(t *testing.T) {
	t0 := time.Now()
	scans := []store.ScanRow{
		{ID: "s1", Domain: "a", StartedAt: t0, Status: "complete"},
		{ID: "s2", Domain: "b", StartedAt: t0.Add(time.Hour), Status: "complete"},
	}
	stub := func(id string) bool { return id == "s2" }
	got := RecentActivity(scans, stub, 10)
	if !got[0].HasAssessment {
		t.Errorf("s2 should have assessment flag set")
	}
	if got[1].HasAssessment {
		t.Errorf("s1 should not have assessment flag set")
	}
}

func TestBuildHeadline_MixedCoverage(t *testing.T) {
	mkAssess := func(fw string) models.Assessment {
		return models.Assessment{Framework: fw, Dimensions: []models.DimensionScore{{
			Dimension: models.DimensionJuridisch, Score: models.ScoreSoeverein,
		}}}
	}
	t0 := time.Now()
	snaps := []TargetSnapshot{
		{
			TargetID: "t1", Domain: "a.example", Kind: models.TargetKindDomain,
			Assessments: map[string]models.Assessment{"wand": mkAssess("wand"), "eucsf": mkAssess("eucsf")},
		},
		{
			TargetID: "t2", Domain: "b.example", Kind: models.TargetKindDomain,
			Assessments: map[string]models.Assessment{"wand": mkAssess("wand")},
		},
		{
			TargetID: "t3", Domain: "host-foo", Kind: models.TargetKindHost,
			Assessments: map[string]models.Assessment{"eucsf": mkAssess("eucsf")},
		},
	}
	scans := []store.ScanRow{
		{ID: "s1", Domain: "a.example", StartedAt: t0},
		{ID: "s2", Domain: "a.example", StartedAt: t0.Add(time.Hour)},
		{ID: "s3", Domain: "b.example", StartedAt: t0.Add(2 * time.Hour)},
		{ID: "s4", Domain: "host-foo", StartedAt: t0.Add(3 * time.Hour)},
		{ID: "s5", Domain: "host-foo", StartedAt: t0.Add(4 * time.Hour)},
	}
	got := BuildHeadline(snaps, scans)
	if got.TotalScans != 5 {
		t.Errorf("TotalScans = %d, want 5", got.TotalScans)
	}
	if got.PerimeterTargets != 2 {
		t.Errorf("PerimeterTargets = %d, want 2", got.PerimeterTargets)
	}
	if got.AgentHostTargets != 1 {
		t.Errorf("AgentHostTargets = %d, want 1", got.AgentHostTargets)
	}
	if !got.LastScanAt.Equal(scans[4].StartedAt) {
		t.Errorf("LastScanAt = %s, want %s", got.LastScanAt, scans[4].StartedAt)
	}
	if len(got.Frameworks) != 2 || got.Frameworks[0] != "eucsf" || got.Frameworks[1] != "wand" {
		t.Errorf("Frameworks = %v, want sorted [eucsf wand]", got.Frameworks)
	}
}

func TestBuildHeadline_EmptyStore(t *testing.T) {
	got := BuildHeadline(nil, nil)
	if got.TotalScans != 0 || got.PerimeterTargets != 0 || got.AgentHostTargets != 0 {
		t.Errorf("counts not zero on empty store: %+v", got)
	}
	if !got.LastScanAt.IsZero() {
		t.Errorf("LastScanAt should be zero for empty store, got %s", got.LastScanAt)
	}
	if len(got.Frameworks) != 0 {
		t.Errorf("Frameworks should be empty, got %v", got.Frameworks)
	}
}

func TestBuildHeadline_EmptyKindCountedAsDomain(t *testing.T) {
	// A snapshot with empty Kind field defaults to perimeter —
	// matches the pkg/models.Target validation behaviour.
	snaps := []TargetSnapshot{{TargetID: "t1", Domain: "a.example" /* Kind: "" */}}
	got := BuildHeadline(snaps, nil)
	if got.PerimeterTargets != 1 {
		t.Errorf("empty Kind should count as perimeter, got %d", got.PerimeterTargets)
	}
}

func TestPostureCountsByKind_FiltersCorrectly(t *testing.T) {
	mkAssess := func(score models.Score) models.Assessment {
		return models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
			Dimension: models.DimensionJuridisch, Score: score,
		}}}
	}
	snaps := []TargetSnapshot{
		{
			TargetID: "t1", Domain: "a", Kind: models.TargetKindDomain,
			Assessments: map[string]models.Assessment{"wand": mkAssess(models.ScoreSoeverein)},
		},
		{
			TargetID: "t2", Domain: "b", Kind: models.TargetKindDomain,
			Assessments: map[string]models.Assessment{"wand": mkAssess(models.ScoreAfhankelijk)},
		},
		{
			TargetID: "t3", Domain: "host-1", Kind: models.TargetKindHost,
			Assessments: map[string]models.Assessment{"wand": mkAssess(models.ScoreVoldoende)},
		},
	}
	external := PostureCountsByKind(snaps, models.TargetKindDomain)
	if external["wand"][models.ScoreSoeverein] != 1 || external["wand"][models.ScoreAfhankelijk] != 1 {
		t.Errorf("external counts wrong: %+v", external["wand"])
	}
	if external["wand"][models.ScoreVoldoende] != 0 {
		t.Errorf("external should not include host-only target: %+v", external["wand"])
	}
	internal := PostureCountsByKind(snaps, models.TargetKindHost)
	if internal["wand"][models.ScoreVoldoende] != 1 {
		t.Errorf("internal should have 1 voldoende, got %+v", internal["wand"])
	}
	if internal["wand"][models.ScoreSoeverein] != 0 {
		t.Errorf("internal should not include domain targets: %+v", internal["wand"])
	}
}

// snap is a tiny helper for the reporting tests below.
func snap(targetID, domain string, fw string, dimRationales ...models.Rationale) TargetSnapshot {
	return TargetSnapshot{
		TargetID:   targetID,
		Domain:     domain,
		Kind:       models.TargetKindDomain,
		LastScanID: "scan-" + targetID,
		LastScanAt: time.Now(),
		Assessments: map[string]models.Assessment{
			fw: {
				Framework: fw,
				Dimensions: []models.DimensionScore{{
					Dimension: models.DimensionJuridisch,
					Rationale: dimRationales,
				}},
			},
		},
	}
}

func TestRuleSummary_DistinctTargetCountsPerScore(t *testing.T) {
	rule := "wand.juridisch.cert_issuer_eea"
	snaps := []TargetSnapshot{
		snap("t1", "a.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreSoeverein, Verdict: "EU"}),
		snap("t2", "b.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreAfhankelijk, Verdict: "US"}),
	}
	got := RuleSummary(snaps, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if got[0].Counts[models.ScoreSoeverein] != 1 || got[0].Counts[models.ScoreAfhankelijk] != 1 {
		t.Errorf("counts wrong: %+v", got[0].Counts)
	}
	if got[0].Counts[models.ScoreVoldoende] != 0 || got[0].Counts[models.ScoreOnbekend] != 0 {
		t.Errorf("unset scores should be zero: %+v", got[0].Counts)
	}
}

func TestRuleSummary_TwiceOnSameTargetCountsOnce(t *testing.T) {
	rule := "wand.data_ai.mx_jurisdiction"
	// One target's Assessment fires the same rule twice (multi-host
	// dimension). Distinct-target convention means count = 1.
	snaps := []TargetSnapshot{
		snap(
			"t1", "a.example", "wand",
			models.Rationale{CriteriumID: rule, Score: models.ScoreAfhankelijk, Verdict: "MX1"},
			models.Rationale{CriteriumID: rule, Score: models.ScoreAfhankelijk, Verdict: "MX2"},
		),
	}
	got := RuleSummary(snaps, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 rule")
	}
	if got[0].Counts[models.ScoreAfhankelijk] != 1 {
		t.Errorf("count = %d, want 1 (distinct targets)", got[0].Counts[models.ScoreAfhankelijk])
	}
}

func TestRuleSummary_StableOrder_WandBeforeEUCSF(t *testing.T) {
	snaps := []TargetSnapshot{
		snap("t1", "a", "eucsf", models.Rationale{CriteriumID: "eucsf.sov2.apex_jurisdiction", Score: models.ScoreSoeverein}),
		snap("t2", "b", "wand", models.Rationale{CriteriumID: "wand.juridisch.cert_issuer_eea", Score: models.ScoreSoeverein}),
	}
	got := RuleSummary(snaps, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0].Framework != "wand" {
		t.Errorf("first row should be wand, got %s", got[0].Framework)
	}
	if got[1].Framework != "eucsf" {
		t.Errorf("second row should be eucsf, got %s", got[1].Framework)
	}
}

func TestRuleTargetRows_FiltersByRule(t *testing.T) {
	rule := "wand.juridisch.cert_issuer_eea"
	other := "wand.juridisch.something_else"
	snaps := []TargetSnapshot{
		snap("t1", "a.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreSoeverein}),
		snap("t2", "b.example", "wand", models.Rationale{CriteriumID: other, Score: models.ScoreAfhankelijk}),
		snap("t3", "c.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreAfhankelijk}),
	}
	got := RuleTargetRows(snaps, "wand", rule)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows for the rule, got %d", len(got))
	}
	for _, r := range got {
		if r.Domain == "b.example" {
			t.Errorf("b.example should be absent — different rule")
		}
	}
}

func TestRuleTargetRows_AfhankelijkBeforeSoeverein(t *testing.T) {
	rule := "wand.juridisch.cert_issuer_eea"
	snaps := []TargetSnapshot{
		snap("t1", "a.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreSoeverein}),
		snap("t2", "b.example", "wand", models.Rationale{CriteriumID: rule, Score: models.ScoreAfhankelijk}),
	}
	got := RuleTargetRows(snaps, "wand", rule)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0].Score != models.ScoreAfhankelijk {
		t.Errorf("afhankelijk should sort first, got %s first", got[0].Score)
	}
}
