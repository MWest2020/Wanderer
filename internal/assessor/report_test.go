package assessor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

func fixtureAssessment() *models.Assessment {
	return &models.Assessment{
		ID:        "a_1",
		ScanID:    "s_1",
		Framework: "dictu",
		CreatedAt: time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
		Dimensions: []models.DimensionScore{
			{
				Dimension:    models.DimensionJuridisch,
				Score:        models.ScoreAfhankelijk,
				Completeness: models.CompletenessComplete,
				Rationale: []models.Rationale{
					{CriteriumID: "dictu.juridisch.cert_issuer_eea", Verdict: "cert issued in US (outside EEA)", Score: models.ScoreAfhankelijk, Evidence: []string{"f_1"}},
				},
			},
			{
				Dimension:    models.DimensionTechnologie,
				Score:        models.ScoreOnbekend,
				Completeness: models.CompletenessPartial,
				Rationale: []models.Rationale{
					{CriteriumID: "dictu.technologie.third_parties_eea", Verdict: "no http.third_party finding", Score: models.ScoreOnbekend, Evidence: []string{}},
				},
			},
			{
				Dimension:    models.DimensionDataAI,
				Score:        models.ScoreOnbekend,
				Completeness: models.CompletenessIncomplete,
				Rationale:    nil,
			},
			{
				Dimension:    models.DimensionOperationeel,
				Score:        models.ScoreSoeverein,
				Completeness: models.CompletenessComplete,
				Rationale: []models.Rationale{
					{CriteriumID: "dictu.operationeel.cert_validity", Verdict: "certificate valid, 83 days remaining", Score: models.ScoreSoeverein, Evidence: []string{"f_2"}},
				},
			},
			{
				Dimension:    models.DimensionMens,
				Score:        models.ScoreOnbekend,
				Completeness: models.CompletenessIncomplete,
				Rationale:    nil,
			},
		},
	}
}

func fixtureRules() Rules {
	return Rules{
		{ID: "dictu.juridisch.cert_issuer_eea", Description: "TLS certificate issued by an authority in the EEA."},
		{ID: "dictu.technologie.third_parties_eea", Description: "HTTP third-party dependencies resolve to AS in the EEA."},
		{ID: "dictu.operationeel.cert_validity", Description: "TLS certificate is valid and not expiring within 30 days."},
	}
}

// TestRenderMarkdown_GoldenShape pins the structural shape of the
// markdown report. Any change to headings or summary rows must be
// deliberate; that is the point.
func TestRenderMarkdown_GoldenShape(t *testing.T) {
	a := fixtureAssessment()
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, a, fixtureRules(), "example.nl"); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()

	mustContain := []string{
		"# Wanderer Assessment — example.nl",
		"Scan: s_1",
		"Framework: dictu",
		"## Samenvatting",
		"| juridisch | afhankelijk | complete |",
		"| technologie | onbekend | partial |",
		"| data_ai | onbekend | n/a |",
		"| operationeel | soeverein | complete |",
		"| mens | onbekend | n/a |",
		"## juridisch — afhankelijk (complete)",
		"### dictu.juridisch.cert_issuer_eea — afhankelijk",
		"Verdict: cert issued in US (outside EEA)",
		"Evidence: f_1",
		"## data_ai — onbekend (n/a)",
		"_Geen regels beschikbaar voor deze dimensie._",
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("markdown missing: %q\n---\n%s", s, got)
		}
	}
}

func TestRenderJSON_HasFiveDimensions(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, fixtureAssessment()); err != nil {
		t.Fatalf("render: %v", err)
	}
	var parsed struct {
		Dimensions []any `json:"dimensions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Dimensions) != 5 {
		t.Errorf("want 5 dimensions, got %d", len(parsed.Dimensions))
	}
}

func TestRenderText_ContainsAllDimensions(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderText(&buf, fixtureAssessment(), fixtureRules(), "example.nl"); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, dim := range []string{"juridisch", "technologie", "data_ai", "operationeel", "mens"} {
		if !strings.Contains(got, dim) {
			t.Errorf("text report missing dimension %s", dim)
		}
	}
}
