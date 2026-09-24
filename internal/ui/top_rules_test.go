package ui

import (
	"strings"
	"testing"
)

// TestBuildTopRuleViews_FlowAndHandelingForTheFleet pins design.md
// "The top 3 without the rationale": a rule that is one of the seven
// flow rules and has a handeling renders as flow + handeling, spoken
// about the fleet rather than one domain.
func TestBuildTopRuleViews_FlowAndHandelingForTheFleet(t *testing.T) {
	rows := []ConcernRow{{
		Framework:   "wand",
		CriteriumID: "wand.juridisch.apex_ip_eea",
		TargetCount: 7,
		Total:       11,
	}}
	views := BuildTopRuleViews(rows)
	if len(views) != 1 {
		t.Fatalf("views = %d, want 1", len(views))
	}
	v := views[0]
	if v.FlowLabel != "Hosting" {
		t.Errorf("FlowLabel = %q, want Hosting", v.FlowLabel)
	}
	if v.Fallback != "" {
		t.Errorf("Fallback = %q, want empty when flow+handeling both resolve", v.Fallback)
	}
	if v.Handeling == "" {
		t.Fatal("Handeling is empty, want a non-empty sentence")
	}
	// The whole sentence, not just "no placeholder left": run 01 dropped
	// {domein} and shipped "Verhuis de hosting van naar een provider",
	// which passed a placeholder-only check.
	if want := "Verhuis de hosting van elk getroffen domein naar een provider met een AS dat in de EER is geregistreerd."; v.Handeling != want {
		t.Errorf("Handeling = %q, want %q", v.Handeling, want)
	}
	if v.BarLabel != "7 van 11 domeinen" {
		t.Errorf("BarLabel = %q, want %q", v.BarLabel, "7 van 11 domeinen")
	}
	if v.WidthPercent == "" {
		t.Error("WidthPercent is empty")
	}
}

// TestBuildTopRuleViews_HandelingWithoutFlowSkipsTheFallback pins run
// 02's nakijken finding: on prod data, wand.operationeel.caa_restricts_
// issuance and wand.accountability.ns_holder_transparent — neither one
// of the seven sovereignty-flow rules — rendered as a bare rule ID even
// though both have a handeling. A handeling alone (no flow prefix) must
// win over the rule-ID fallback.
func TestBuildTopRuleViews_HandelingWithoutFlowSkipsTheFallback(t *testing.T) {
	rows := []ConcernRow{{
		Framework:   "wand",
		CriteriumID: "wand.operationeel.caa_restricts_issuance",
		TargetCount: 2,
		Total:       3,
	}}
	views := BuildTopRuleViews(rows)
	v := views[0]
	if v.Fallback != "" {
		t.Errorf("Fallback = %q, want empty when a handeling resolves", v.Fallback)
	}
	if v.FlowLabel != "" {
		t.Errorf("FlowLabel = %q, want empty: this rule is not one of the seven flows", v.FlowLabel)
	}
	want := "Voeg een CAA-record toe aan elk getroffen domein dat vastlegt welke certificaatautoriteiten mogen uitgeven."
	if v.Handeling != want {
		t.Errorf("Handeling = %q, want %q", v.Handeling, want)
	}
}

// TestBuildTopRuleViews_NoHandelingFallsBackToRuleID covers a rule with
// neither a flow nor a handeling (e.g. an eucsf-only concern): design.md
// "If a rule has no flow or no handeling, its ID is shown — never the
// rationale".
func TestBuildTopRuleViews_NoHandelingFallsBackToRuleID(t *testing.T) {
	rows := []ConcernRow{{
		Framework:   "eucsf",
		CriteriumID: "eucsf.sov2.cert_issuer_eu",
		TargetCount: 2,
		Total:       3,
	}}
	views := BuildTopRuleViews(rows)
	v := views[0]
	if v.Fallback != "eucsf.sov2.cert_issuer_eu" {
		t.Errorf("Fallback = %q, want the rule ID", v.Fallback)
	}
	if v.FlowLabel != "" || v.Handeling != "" {
		t.Errorf("FlowLabel/Handeling = %q/%q, want both empty when falling back", v.FlowLabel, v.Handeling)
	}
}

// TestHandelingForFleet_EverySentenceKeepsItsSubject walks every handeling
// that exists: each one must come out with its {domein} replaced by the
// fleet subject, not deleted, so no sentence is left with a gap where its
// object was ("van naar", "voor ,").
func TestHandelingForFleet_EverySentenceKeepsItsSubject(t *testing.T) {
	n := 0
	for _, fr := range flowRules {
		h, ok := handelingForFleet(fr.id)
		if !ok {
			continue
		}
		n++
		if strings.Contains(h, "{domein}") || !strings.Contains(h, fleetSubject) {
			t.Errorf("%s: %q, want {domein} replaced by %q", fr.id, h, fleetSubject)
		}
	}
	if n == 0 {
		t.Fatal("no flow rule has a handeling; the test checked nothing")
	}
}
