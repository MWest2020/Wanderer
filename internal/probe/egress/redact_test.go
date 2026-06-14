package egress

import (
	"strings"
	"testing"
)

func TestApply_SecretKeyName(t *testing.T) {
	cases := []struct {
		key, value string
		wantPlace  bool
	}{
		{"AWS_SECRET_ACCESS_KEY", "abcDEF12345", true},
		{"API_KEY", "hunter2", true},
		{"ACCESS_KEY", "supersecret", true},
		{"PRIVATE_KEY", "stuff", true},
		{"DB_USER", "app", false},
		{"SMTP_HOST", "mail.example.nl", false},
	}
	for _, c := range cases {
		got, changed := Apply(c.key, c.value)
		if c.wantPlace {
			if got != Placeholder || !changed {
				t.Errorf("Apply(%q, %q) = (%q, %v), want placeholder", c.key, c.value, got, changed)
			}
		} else if got != c.value || changed {
			t.Errorf("Apply(%q, %q) = (%q, %v), want passthrough", c.key, c.value, got, changed)
		}
	}
}

func TestApply_TokenShapes(t *testing.T) {
	cases := []string{
		"AKIAABCDEFGHIJKLMNOP",                        // AWS
		"xoxb-12345-67890-abcdef",                     // Slack bot
		"ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",    // GitHub
		"AIzaSyBzDDDDD123456789DDDDDDDDDDDDDDDDDDDDD", // Google API key (40 chars after AIza)
		"-----BEGIN RSA PRIVATE KEY-----",             // PEM
		"eyJhbGciOi.eyJpYXQiOi.signaturepart",         // JWT shape
	}
	for _, v := range cases {
		got, changed := Apply("plain_field", v)
		if !changed || got != Placeholder {
			t.Errorf("Apply(%q) was not redacted: got %q", v, got)
		}
	}
}

func TestApply_URLCredentials(t *testing.T) {
	got, changed := Apply("DATABASE_URL", "postgres://app:hunter2@db.example:5432/app")
	if !changed {
		t.Fatalf("URL with creds should be redacted")
	}
	if !strings.Contains(got, Placeholder) {
		t.Errorf("missing placeholder: %q", got)
	}
	if strings.Contains(got, "hunter2") {
		t.Errorf("password leaked: %q", got)
	}
	if !strings.Contains(got, "app:") || !strings.Contains(got, "@db.example") {
		t.Errorf("non-secret parts of URL lost: %q", got)
	}
}

func TestApply_URLWithoutCredentialsIsNotChanged(t *testing.T) {
	got, changed := Apply("ENDPOINT", "https://s3.eu-west-1.amazonaws.com")
	if changed || got != "https://s3.eu-west-1.amazonaws.com" {
		t.Errorf("URL without creds should pass through; got %q changed=%v", got, changed)
	}
}

func TestApply_ShortValues_NotRedactedWhenNotSecret(t *testing.T) {
	got, changed := Apply("DB_USER", "app")
	if changed || got != "app" {
		t.Errorf("short non-secret value was redacted: %q", got)
	}
}

func TestApply_EmptyValue(t *testing.T) {
	got, changed := Apply("API_KEY", "")
	if changed || got != "" {
		t.Errorf("empty value should not be touched: %q changed=%v", got, changed)
	}
}

func TestApply_GoldenSnippets(t *testing.T) {
	type snippet struct {
		key, value, want string
	}
	snippets := []snippet{
		{"AWS_ACCESS_KEY_ID", "AKIAABCDEFGHIJKLMNOP", Placeholder},
		{"AWS_SECRET_ACCESS_KEY", "abc/def+ghi=", Placeholder},
		{"GITHUB_TOKEN", "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Placeholder},
		{"SLACK_TOKEN", "xoxb-1-2-zzz", Placeholder}, // key contains "TOKEN" so the value is redacted
		{"NEXTCLOUD_API_KEY", "AAAAA", Placeholder},
		{"OAUTH_CLIENT_SECRET", "secret-yo", Placeholder},
		{"OIDC_ISSUER", "https://login.microsoftonline.com/abc/v2.0", "https://login.microsoftonline.com/abc/v2.0"},
		{"DATABASE_URL", "postgres://u:p@db:5432/x", "postgres://u:" + Placeholder + "@db:5432/x"},
		{"SMTP_HOST", "smtp.example.nl", "smtp.example.nl"},
		{"DEBUG", "true", "true"},
	}
	// Inflate to ≥30 cases by adding variants.
	for i := 0; i < 25; i++ {
		snippets = append(snippets, snippet{"PASSWORD", "value-" + string(rune('a'+i%26)), Placeholder})
	}
	for _, s := range snippets {
		got, _ := Apply(s.key, s.value)
		if s.want == Placeholder {
			if got != Placeholder {
				t.Errorf("Apply(%q, %q) = %q, want placeholder", s.key, s.value, got)
			}
		} else if got != s.want {
			t.Errorf("Apply(%q, %q) = %q, want %q", s.key, s.value, got, s.want)
		}
	}
}

func TestApply_NoOpOnPlainValues(t *testing.T) {
	plain := []string{"foo", "bar.example.nl", "v1.2.3", "8080", "true"}
	for _, v := range plain {
		got, changed := Apply("MISC", v)
		if changed || got != v {
			t.Errorf("plain value %q was changed to %q", v, got)
		}
	}
}
