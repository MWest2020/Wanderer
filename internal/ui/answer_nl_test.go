package ui

import "testing"

// TestAnswerNL_Completeness pins that answer_nl.yaml carries a
// non-empty template for every outcome BuildAnswerVerdict can render.
// A renamed or deleted key here would otherwise silently fall back to
// an empty headline (renderAnswerCopy's "loud in tests, quiet at
// runtime" trade-off).
func TestAnswerNL_Completeness(t *testing.T) {
	m, err := loadAnswerNL()
	if err != nil {
		t.Fatalf("load answer_nl.yaml: %v", err)
	}
	for _, key := range []string{"ja", "ja_unanswered", "nee", "onbekend"} {
		if m[key] == "" {
			t.Errorf("answer_nl.yaml: missing or empty outcome %q", key)
		}
	}
}
