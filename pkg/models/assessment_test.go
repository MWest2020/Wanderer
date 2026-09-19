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

// TestAssessmentJSON_PreReasonFieldStillLoads pins a snapshot of an
// Assessment as persisted before the accountability dimension: five
// DICTU dimensions, no `reason` field anywhere. It must load
// unchanged — reason is additive, WandDimensions extends the list but
// does not invalidate assessments that predate it.
func TestAssessmentJSON_PreReasonFieldStillLoads(t *testing.T) {
	raw := `{
		"id": "a_1",
		"scan_id": "s_1",
		"framework": "wand",
		"created_at": "2026-04-24T10:00:00Z",
		"dimensions": [
			{"dimension": "juridisch", "score": "afhankelijk", "completeness": "complete",
			 "rationale": [{"criterium_id": "wand.juridisch.cert_issuer_eea", "verdict": "cert issued in US (outside EEA)", "score": "afhankelijk", "evidence": ["f_1"]}]},
			{"dimension": "technologie", "score": "onbekend", "completeness": "partial",
			 "rationale": [{"criterium_id": "wand.technologie.third_parties_eea", "verdict": "no http.third_party finding", "score": "onbekend", "evidence": []}]},
			{"dimension": "data_ai", "score": "onbekend", "completeness": "incomplete", "rationale": null},
			{"dimension": "operationeel", "score": "soeverein", "completeness": "complete",
			 "rationale": [{"criterium_id": "wand.operationeel.cert_validity", "verdict": "certificate valid, 83 days remaining", "score": "soeverein", "evidence": ["f_2"]}]},
			{"dimension": "mens", "score": "onbekend", "completeness": "incomplete", "rationale": null}
		]
	}`

	var got Assessment
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal pre-reason assessment: %v", err)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("pre-reason assessment fails validation: %v", err)
	}
	if len(got.Dimensions) != 5 {
		t.Fatalf("want the original 5 dimensions preserved, got %d", len(got.Dimensions))
	}
	for _, d := range got.Dimensions {
		if d.Dimension == DimensionAccountability {
			t.Fatalf("pre-change assessment should not gain an accountability entry")
		}
		for _, r := range d.Rationale {
			if r.Reason != "" {
				t.Errorf("want empty Reason on pre-change rationale %s, got %q", r.CriteriumID, r.Reason)
			}
		}
	}
	jur := got.Dimensions[0]
	if jur.Dimension != DimensionJuridisch || jur.Score != ScoreAfhankelijk || jur.Completeness != CompletenessComplete {
		t.Errorf("juridisch dimension changed on load: %+v", jur)
	}
}
