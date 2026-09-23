package api

import "testing"

// TestValidImportToken_EmptyConfiguredNeverMatches unit-tests
// validImportToken directly (bypassing HTTP header whitespace
// trimming, which would otherwise swallow a trailing-space edge case
// before it reaches this function): an unset WANDERER_IMPORT_TOKEN
// must refuse every header, including a bearer value that happens to
// be empty too — the route fails closed, it never treats "nothing
// configured" as "nothing required".
func TestValidImportToken_EmptyConfiguredNeverMatches(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"empty bearer value", "Bearer "},
		{"no header", ""},
		{"arbitrary value", "Bearer anything"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if validImportToken("", tc.header) {
				t.Errorf("validImportToken(%q, %q) = true, want false (no token configured)", "", tc.header)
			}
		})
	}
}

func TestValidImportToken_MatchingBearerToken(t *testing.T) {
	if !validImportToken("s3cr3t", "Bearer s3cr3t") {
		t.Error("want matching bearer token to validate")
	}
}

func TestValidImportToken_WrongOrMissingBearer(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"wrong token", "Bearer nope"},
		{"missing header", ""},
		{"missing Bearer prefix", "s3cr3t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if validImportToken("s3cr3t", tc.header) {
				t.Errorf("validImportToken(%q, %q) = true, want false", "s3cr3t", tc.header)
			}
		})
	}
}
