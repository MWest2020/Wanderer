package ui

import (
	"strings"
	"testing"
)

// TestBuildTopRuleViews_FlowAndHandelingWithoutDomain pins design.md
// "The top 3 without the rationale": a rule that is one of the seven
// flow rules and has a handeling renders as flow + handeling, with
// {domein} left out (not filled with a placeholder or a real domain).
func TestBuildTopRuleViews_FlowAndHandelingWithoutDomain(t *testing.T) {
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
	if want := "{domein}"; strings.Contains(v.Handeling, want) {
		t.Errorf("Handeling = %q, must not carry the raw {domein} placeholder", v.Handeling)
	}
	if v.BarLabel != "7 van 11 domeinen" {
		t.Errorf("BarLabel = %q, want %q", v.BarLabel, "7 van 11 domeinen")
	}
	if v.WidthPercent == "" {
		t.Error("WidthPercent is empty")
	}
}

// TestBuildTopRuleViews_NoFlowFallsBackToRuleID covers a rule outside
// the seven sovereignty-flow rules (e.g. an eucsf-only concern): design.
// md "If a rule has no flow or no handeling, its ID is shown — never
// the rationale".
func TestBuildTopRuleViews_NoFlowFallsBackToRuleID(t *testing.T) {
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

// TestBuildTopRuleViews_FlowWithoutHandelingFallsBackToRuleID covers a
// rule that is a sovereignty-flow rule but (hypothetically) has no
// handeling entry — the fallback rule applies the same way, since a
// half-built line (flow with no action) would read as a dead end.
func TestBuildTopRuleViews_FlowWithoutHandelingFallsBackToRuleID(t *testing.T) {
	rows := []ConcernRow{{
		Framework:   "wand",
		CriteriumID: "wand.juridisch.does_not_exist",
		TargetCount: 1,
		Total:       1,
	}}
	views := BuildTopRuleViews(rows)
	v := views[0]
	if v.Fallback != "wand.juridisch.does_not_exist" {
		t.Errorf("Fallback = %q, want the rule ID", v.Fallback)
	}
}
