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
	flows := SovereigntyFlows([]models.Assessment{a}, nil)
	if len(flows) != 2 {
		t.Fatalf("flows = %d, want 2 (apex + mx; other ignored)", len(flows))
	}
	// Fixed order: Hosting (apex) before Mail (mx).
	if flows[0].Label != "Hosting" || flows[1].Label != "Mail" {
		t.Fatalf("order = %q,%q want Hosting,Mail", flows[0].Label, flows[1].Label)
	}
	// The verdict SHALL be the Dutch answer-sheet sentence, never the
	// rule's raw English Verdict (spec.md "Eén taal per laag").
	if flows[1].Verdict != "De mail wordt buiten de EER gerouteerd." || flows[1].Score != "afhankelijk" {
		t.Errorf("mail flow = %+v", flows[1])
	}
}

func TestSovereigntyFlows_EmptyWhenNoFlowRules(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Rationale: []models.Rationale{rationale("wand.operationeel.cert_validity", "x", models.ScoreSoeverein)},
	}}}
	if f := SovereigntyFlows([]models.Assessment{a}, nil); len(f) != 0 {
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
	for _, s := range BuildFlowStates([]models.Assessment{a}, false, "example.nl", nil) {
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
	for _, s := range BuildFlowStates([]models.Assessment{a}, true, "example.nl", nil) {
		byLabel[s.Label] = s
	}
	if got := byLabel["Mail"]; got.State != "niet_gemeten" {
		t.Errorf("Mail = %+v, want niet_gemeten — the scan is done and this flow never fired", got)
	}
	if got := byLabel["Hosting"]; got.State != "beantwoord" {
		t.Errorf("Hosting = %+v, want beantwoord regardless of done", got)
	}
}

// TestDutchFlowVerdict_UnknownRuleNeverFallsBackToRawEnglish is task
// 4.3's guard: a shown verdict SHALL come from accountability_nl.yaml's
// table (spec.md "Eén taal per laag"). If dutchFlowVerdict were ever
// changed to fall back to the rule's own (English) Rationale.Verdict
// when a ruleID or outcome is missing from the table — the exact shape
// of the run 04b regression — this test catches it.
func TestDutchFlowVerdict_UnknownRuleNeverFallsBackToRawEnglish(t *testing.T) {
	if got := dutchFlowVerdict("wand.juridisch.does_not_exist", models.ScoreAfhankelijk, ""); got != "" {
		t.Fatalf("dutchFlowVerdict for a rule absent from the table = %q, want empty — a shown verdict must come from the table, never a raw fallback", got)
	}
}

// TestDutchFlowVerdict_EveryFlowRuleHasAllFourOutcomes pins that
// accountability_nl.yaml actually carries a template for each of the
// seven flow rules and all four scores — the completeness the previous
// test's "no fallback" guarantee depends on in production.
func TestDutchFlowVerdict_EveryFlowRuleHasAllFourOutcomes(t *testing.T) {
	scores := []models.Score{models.ScoreSoeverein, models.ScoreVoldoende, models.ScoreAfhankelijk, models.ScoreOnbekend}
	for _, fr := range flowRules {
		for _, score := range scores {
			if got := dutchFlowVerdict(fr.id, score, ""); got == "" {
				t.Errorf("%s / %s has no Dutch verdict template in accountability_nl.yaml", fr.id, score)
			}
		}
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
