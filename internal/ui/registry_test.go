package ui

import "testing"

func TestLookupRule_Wand(t *testing.T) {
	r, ok := lookupRule("wand", "wand.juridisch.cert_issuer_eea")
	if !ok {
		t.Fatal("expected wand rule to be found")
	}
	if r.Description == "" || r.Rationale == "" {
		t.Errorf("expected Description and Rationale populated, got %+v", r)
	}
}

// TestLookupRule_DictuAlias pins the deprecated `dictu` framework
// key arm: a DB row that bypassed the rename migration still
// renders by resolving against wand.DefaultRules. The alias goes
// away in the next release per ADR-0011.
func TestLookupRule_DictuAlias(t *testing.T) {
	r, ok := lookupRule("dictu", "wand.juridisch.cert_issuer_eea")
	if !ok {
		t.Fatal("dictu alias should resolve against wand.DefaultRules")
	}
	if r.Description == "" {
		t.Errorf("expected Description on aliased lookup, got %+v", r)
	}
}

func TestLookupRule_EUCSF(t *testing.T) {
	r, ok := lookupRule("eucsf", "eucsf.sov2.cert_issuer_eu")
	if !ok {
		t.Fatal("expected EUCSF rule to be found")
	}
	if r.Rationale == "" {
		t.Errorf("expected Rationale populated, got %+v", r)
	}
}

func TestLookupRule_RetiredCriteriumID(t *testing.T) {
	_, ok := lookupRule("wand", "wand.juridisch.no_such_rule_anymore")
	if ok {
		t.Fatal("expected lookup to fail on retired criterium ID")
	}
}

func TestLookupRule_UnknownFramework(t *testing.T) {
	_, ok := lookupRule("nonsense", "wand.juridisch.cert_issuer_eea")
	if ok {
		t.Fatal("expected lookup to fail on unknown framework")
	}
}
