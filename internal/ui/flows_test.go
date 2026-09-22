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
		Rationale: []models.Rationale{rationale("wand.operationeel.cert_validity", "x", models.ScoreSoeverein)},
	}}}
	if f := SovereigntyFlows([]models.Assessment{a}); len(f) != 0 {
		t.Fatalf("flows = %d, want 0", len(f))
	}
}

func snapWithFlows(id string, rs ...models.Rationale) TargetSnapshot {
	return TargetSnapshot{
		TargetID: id,
		Assessments: map[string]models.Assessment{
			"wand": {Framework: "wand", Dimensions: []models.DimensionScore{{Rationale: rs}}},
		},
	}
}

func TestBuildFlowStates_RunningScanShowsBezigForMissingFlows(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Dimension: models.DimensionJuridisch,
		Rationale: []models.Rationale{
			{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein, Evidence: []string{"f1"}},
			{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "no dns.mx finding — no mail routing to assess", Score: models.ScoreOnbekend},
		},
	}}}
	byLabel := map[string]FlowState{}
	for _, s := range BuildFlowStates([]models.Assessment{a}, false, "example.nl") {
		byLabel[s.Label] = s
	}
	if got := byLabel["Hosting"]; got.State != "beantwoord" || got.Score != "soeverein" {
		t.Errorf("Hosting = %+v, want beantwoord/soeverein", got)
	}
	if got := byLabel["Mail"]; got.State != "bezig" {
		t.Errorf("Mail = %+v, want bezig — its rule ran but found no evidence, and the scan can still produce it", got)
	}
	if got := byLabel["DNS"]; got.State != "bezig" {
		t.Errorf("DNS = %+v, want bezig — its rule has not even fired yet", got)
	}
}

func TestBuildFlowStates_DoneScanShowsNietGemetenForMissingFlows(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Dimension: models.DimensionJuridisch,
		Rationale: []models.Rationale{
			{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein, Evidence: []string{"f1"}},
		},
	}}}
	byLabel := map[string]FlowState{}
	for _, s := range BuildFlowStates([]models.Assessment{a}, true, "example.nl") {
		byLabel[s.Label] = s
	}
	if got := byLabel["Mail"]; got.State != "niet_gemeten" {
		t.Errorf("Mail = %+v, want niet_gemeten — the scan is done and this flow never fired", got)
	}
	if got := byLabel["Hosting"]; got.State != "beantwoord" {
		t.Errorf("Hosting = %+v, want beantwoord regardless of done", got)
	}
}

func TestSovereigntyFlowRollup_CountsAndWorst(t *testing.T) {
	snaps := []TargetSnapshot{
		snapWithFlows(
			"t1",
			rationale("wand.juridisch.mx_vendor_jurisdiction", "us", models.ScoreAfhankelijk),
			rationale("wand.juridisch.apex_ip_eea", "nl", models.ScoreSoeverein),
		),
		snapWithFlows(
			"t2",
			rationale("wand.juridisch.mx_vendor_jurisdiction", "nl", models.ScoreSoeverein),
			rationale("wand.juridisch.apex_ip_eea", "nl", models.ScoreSoeverein),
		),
	}
	roll := SovereigntyFlowRollup(snaps)
	byLabel := map[string]FlowRollup{}
	for _, r := range roll {
		byLabel[r.Label] = r
	}
	mail := byLabel["Mail"]
	if mail.Total != 2 || mail.Afhankelijk != 1 || mail.Worst != "afhankelijk" {
		t.Errorf("Mail rollup = %+v, want total2 afh1 worst=afhankelijk", mail)
	}
	host := byLabel["Hosting"]
	if host.Total != 2 || host.Afhankelijk != 0 || host.Worst != "soeverein" {
		t.Errorf("Hosting rollup = %+v, want total2 afh0 worst=soeverein", host)
	}
	// Fixed order: Hosting before Mail.
	if roll[0].Label != "Hosting" || roll[1].Label != "Mail" {
		t.Errorf("order = %q,%q", roll[0].Label, roll[1].Label)
	}
}
