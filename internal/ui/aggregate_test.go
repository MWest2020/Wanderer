package ui

import (
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

func TestPostureCounts_BucketsByFrameworkAndScore(t *testing.T) {
	snaps := []TargetSnapshot{
		{TargetID: "t1", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Score: models.ScoreSoeverein}}},
		}},
		{TargetID: "t2", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Score: models.ScoreAfhankelijk}}},
		}},
		{TargetID: "t3", Assessments: map[string]models.Assessment{
			"wand": {Dimensions: []models.DimensionScore{{Score: models.ScoreOnbekend}}},
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
	stub := func(fw, id string) (assessor.Rule, bool) {
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
