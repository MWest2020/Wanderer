package ui

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestBuildAccountabilityAnswers_SoevereinFillsRegistrant(t *testing.T) {
	findings := map[string]models.Finding{
		"f1": {ID: "f1", ProbeID: "whois.registrant_identity", Subject: "example.com", Attributes: map[string]any{"name": "Voorbeeld B.V."}},
	}
	dim := models.DimensionScore{
		Dimension: models.DimensionAccountability,
		Rationale: []models.Rationale{
			{CriteriumID: "wand.accountability.registrant_identifiable", Score: models.ScoreSoeverein, Evidence: []string{"f1"}},
		},
	}
	rows := BuildAccountabilityAnswers(dim, findings, "example.com")
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	row := rows[0]
	if row.AnswerClass != "ja" || row.AnswerLabel != "Ja" {
		t.Errorf("answer = %s/%s, want ja/Ja", row.AnswerClass, row.AnswerLabel)
	}
	if !strings.Contains(row.Verdict, "Voorbeeld B.V.") {
		t.Errorf("verdict %q does not name the registrant", row.Verdict)
	}
	if row.Remediation != "" {
		t.Errorf("soeverein must not carry a remediation line, got %q", row.Remediation)
	}
}

func TestBuildAccountabilityAnswers_AfhankelijkGetsRemediation(t *testing.T) {
	findings := map[string]models.Finding{
		"f1": {ID: "f1", ProbeID: "whois.reseller", Subject: "example.com", Attributes: map[string]any{"present": true, "name": "Resell Co"}},
	}
	dim := models.DimensionScore{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.accountability.no_reseller", Score: models.ScoreAfhankelijk, Evidence: []string{"f1"}},
		},
	}
	rows := BuildAccountabilityAnswers(dim, findings, "example.com")
	row := rows[0]
	if row.AnswerClass != "nee" {
		t.Fatalf("answer class = %s, want nee", row.AnswerClass)
	}
	if row.Remediation == "" || !strings.Contains(row.Remediation, "Resell Co") {
		t.Errorf("remediation %q does not name the reseller", row.Remediation)
	}
}

func TestBuildAccountabilityAnswers_StructuralIsNvtWithoutRemediation(t *testing.T) {
	dim := models.DimensionScore{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.accountability.registrant_identifiable", Score: models.ScoreOnbekend, Reason: assessor.ReasonRegistryRedacted, Verdict: "the SIDN registry does not publish registrant data for .nl domains"},
		},
	}
	rows := BuildAccountabilityAnswers(dim, nil, "example.nl")
	row := rows[0]
	if row.AnswerClass != "nvt" || row.AnswerLabel != "n.v.t." {
		t.Fatalf("answer = %s/%s, want nvt/n.v.t.", row.AnswerClass, row.AnswerLabel)
	}
	if row.Remediation != "" {
		t.Errorf("n.v.t. must not carry a remediation line, got %q", row.Remediation)
	}
	if !strings.Contains(row.Verdict, "nl") {
		t.Errorf("verdict %q does not resolve the .nl TLD", row.Verdict)
	}
	if row.NoEvidenceNote == "" {
		t.Error("expected a NoEvidenceNote when Evidence is empty")
	}
}

func TestBuildAccountabilityAnswers_ProbeUnavailableIsOnbekendNotNvt(t *testing.T) {
	dim := models.DimensionScore{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.accountability.soa_rname", Score: models.ScoreOnbekend, Reason: assessor.ReasonProbeUnavailable},
		},
	}
	rows := BuildAccountabilityAnswers(dim, nil, "example.com")
	row := rows[0]
	if row.AnswerClass != "onbekend" {
		t.Fatalf("answer class = %s, want onbekend (a gap reason is not n.v.t.)", row.AnswerClass)
	}
}
