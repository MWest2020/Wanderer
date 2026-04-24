package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScoreRank(t *testing.T) {
	cases := []struct {
		s    Score
		want int
	}{
		{ScoreAfhankelijk, 1},
		{ScoreVoldoende, 2},
		{ScoreSoeverein, 3},
		{ScoreOnbekend, 0},
		{Score("garbage"), 0},
	}
	for _, c := range cases {
		if got := c.s.Rank(); got != c.want {
			t.Errorf("%s.Rank() = %d, want %d", c.s, got, c.want)
		}
	}
}

func TestAssessmentValidate(t *testing.T) {
	good := Assessment{
		ScanID:    "s_1",
		Framework: "dictu",
		CreatedAt: time.Now().UTC(),
		Dimensions: []DimensionScore{
			{
				Dimension:    DimensionJuridisch,
				Score:        ScoreAfhankelijk,
				Completeness: CompletenessComplete,
				Rationale: []Rationale{
					{CriteriumID: "dictu.1.1", Verdict: "cert issuer in US", Score: ScoreAfhankelijk, Evidence: []string{"f_1"}},
				},
			},
		},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("good assessment rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(a *Assessment)
	}{
		{"empty scan id", func(a *Assessment) { a.ScanID = "" }},
		{"empty framework", func(a *Assessment) { a.Framework = "" }},
		{"no dimensions", func(a *Assessment) { a.Dimensions = nil }},
		{"bad dimension", func(a *Assessment) { a.Dimensions[0].Dimension = "bogus" }},
		{"bad score", func(a *Assessment) { a.Dimensions[0].Score = "bogus" }},
		{"bad completeness", func(a *Assessment) { a.Dimensions[0].Completeness = "bogus" }},
		{"missing criterium id", func(a *Assessment) { a.Dimensions[0].Rationale[0].CriteriumID = "" }},
		{"bad rationale score", func(a *Assessment) { a.Dimensions[0].Rationale[0].Score = "bogus" }},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			a := good
			// Deep-copy the slice we are about to mutate so cases are
			// independent.
			dims := make([]DimensionScore, len(good.Dimensions))
			copy(dims, good.Dimensions)
			if len(good.Dimensions[0].Rationale) > 0 {
				rats := make([]Rationale, len(good.Dimensions[0].Rationale))
				copy(rats, good.Dimensions[0].Rationale)
				dims[0].Rationale = rats
			}
			a.Dimensions = dims
			c.mut(&a)
			if err := a.Validate(); err == nil {
				t.Errorf("expected validation error")
			}
		})
	}
}

func TestAssessmentJSONRoundTrip(t *testing.T) {
	a := Assessment{
		ID:        "a_1",
		ScanID:    "s_1",
		Framework: "dictu",
		CreatedAt: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
		Dimensions: []DimensionScore{
			{
				Dimension:    DimensionOperationeel,
				Score:        ScoreVoldoende,
				Completeness: CompletenessPartial,
				Rationale: []Rationale{
					{CriteriumID: "dictu.op.1", Verdict: "cert valid", Score: ScoreVoldoende, Evidence: []string{"f_1", "f_2"}},
					{CriteriumID: "dictu.op.2", Verdict: "no evidence", Score: ScoreOnbekend, Evidence: []string{}},
				},
			},
		},
		Report: "# Wanderer Assessment",
	}
	buf, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Assessment
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != a.ID || got.ScanID != a.ScanID || got.Framework != a.Framework {
		t.Errorf("top-level fields diverged: %+v", got)
	}
	if len(got.Dimensions) != 1 || got.Dimensions[0].Dimension != DimensionOperationeel {
		t.Errorf("dimensions diverged: %+v", got.Dimensions)
	}
	if len(got.Dimensions[0].Rationale) != 2 {
		t.Errorf("rationale count diverged: %d", len(got.Dimensions[0].Rationale))
	}
	if !got.CreatedAt.Equal(a.CreatedAt) {
		t.Errorf("created_at diverged: %v vs %v", got.CreatedAt, a.CreatedAt)
	}
}
