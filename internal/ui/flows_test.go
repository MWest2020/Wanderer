package ui

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func rationale(id, verdict string, score models.Score) models.Rationale {
	return models.Rationale{CriteriumID: id, Verdict: verdict, Score: score}
}

func TestSovereigntyFlows_OrdersAndLabels(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Dimension: models.DimensionJuridisch,
		Rationale: []models.Rationale{
			rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
			rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
			rationale("wand.juridisch.some_other_rule", "ignored", models.ScoreSoeverein),
		},
	}}}
	flows := SovereigntyFlows([]models.Assessment{a})
	if len(flows) != 2 {
		t.Fatalf("flows = %d, want 2 (apex + mx; other ignored)", len(flows))
	}
	// Fixed order: Hosting (apex) before Mail (mx).
	if flows[0].Label != "Hosting" || flows[1].Label != "Mail" {
		t.Fatalf("order = %q,%q want Hosting,Mail", flows[0].Label, flows[1].Label)
	}
	if flows[1].Verdict != "mx hosts in US (outside EEA)" || flows[1].Score != "afhankelijk" {
		t.Errorf("mail flow = %+v", flows[1])
	}
}

func TestSovereigntyFlows_EmptyWhenNoFlowRules(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Rationale: []models.Rationale{rationale("wand.juridisch.cert_issuer_eea", "x", models.ScoreSoeverein)},
	}}}
	if f := SovereigntyFlows([]models.Assessment{a}); len(f) != 0 {
		t.Fatalf("flows = %d, want 0", len(f))
	}
}
