package ui

import "testing"

func TestLookupRule_DICTU(t *testing.T) {
	r, ok := lookupRule("dictu", "dictu.juridisch.cert_issuer_eea")
	if !ok {
		t.Fatal("expected DICTU rule to be found")
	}
	if r.Description == "" || r.Rationale == "" {
		t.Errorf("expected Description and Rationale populated, got %+v", r)
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
	_, ok := lookupRule("dictu", "dictu.juridisch.no_such_rule_anymore")
	if ok {
		t.Fatal("expected lookup to fail on retired criterium ID")
	}
}

func TestLookupRule_UnknownFramework(t *testing.T) {
	_, ok := lookupRule("nonsense", "dictu.juridisch.cert_issuer_eea")
	if ok {
		t.Fatal("expected lookup to fail on unknown framework")
	}
}
