package ui

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestBuildFlowAnswers_MapsScoreToAnswerAndKeepsObservedFact(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Dimension: models.DimensionJuridisch,
		Rationale: []models.Rationale{
			{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex IPs in NL (EEA)", Score: models.ScoreSoeverein, Evidence: []string{"f1"}},
			{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "mx hosts in US (outside EEA)", Score: models.ScoreAfhankelijk, Evidence: []string{"f2"}},
			{CriteriumID: "wand.juridisch.registrar_jurisdiction", Verdict: "ignored — not a sovereignty flow", Score: models.ScoreSoeverein},
		},
	}}}
	findings := map[string]models.Finding{
		"f1": {ID: "f1", ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"country": "NL"}},
	}
	rows := BuildFlowAnswers([]models.Assessment{a}, findings)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (apex + mx; registrar_jurisdiction is not a flow rule)", len(rows))
	}

	hosting := rows[0]
	if hosting.RuleID != "wand.juridisch.apex_ip_eea" {
		t.Fatalf("rows[0] = %q, want the Hosting flow (fixed order)", hosting.RuleID)
	}
	if hosting.AnswerClass != "ja" || hosting.AnswerLabel != "Ja" {
		t.Errorf("hosting answer = %s/%s, want ja/Ja", hosting.AnswerClass, hosting.AnswerLabel)
	}
	if hosting.Question == "" {
		t.Error("expected a plain-language Dutch question")
	}
	if hosting.Verdict != "apex IPs in NL (EEA)" {
		t.Errorf("verdict = %q, want the rule's own observed-fact sentence unchanged", hosting.Verdict)
	}
	if len(hosting.Evidence) != 1 || hosting.Evidence[0].ProbeID != "ip.asn" {
		t.Errorf("expected the finding behind apex_ip_eea to be collapsed into Evidence, got %+v", hosting.Evidence)
	}

	mail := rows[1]
	if mail.AnswerClass != "nee" || mail.AnswerLabel != "Nee" {
		t.Errorf("mail answer = %s/%s, want nee/Nee", mail.AnswerClass, mail.AnswerLabel)
	}
}

func TestBuildFlowAnswers_OnbekendWhenScoreOnbekend(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.transit.eu_path", Verdict: "no geo-attributed transit hops", Score: models.ScoreOnbekend},
		},
	}}}
	rows := BuildFlowAnswers([]models.Assessment{a}, nil)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].AnswerClass != "onbekend" {
		t.Errorf("answer class = %s, want onbekend", rows[0].AnswerClass)
	}
	if rows[0].NoEvidenceNote == "" {
		t.Error("expected a NoEvidenceNote when there is no evidence to expand")
	}
}

func TestBuildFlowAnswers_OmitsFlowsThatNeverFired(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Rationale: []models.Rationale{
			{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
		},
	}}}
	rows := BuildFlowAnswers([]models.Assessment{a}, nil)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 (only Hosting fired)", len(rows))
	}
	for _, r := range rows {
		if strings.Contains(r.RuleID, "mx_vendor") {
			t.Errorf("Mail should not appear — its rule never fired")
		}
	}
}
