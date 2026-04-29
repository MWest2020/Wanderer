package models

import (
	"strings"
	"testing"
)

func TestNormaliseDomain_AcceptsPublicForms(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"example.nl", "example.nl"},
		{"  EXAMPLE.NL.  ", "example.nl"},
		{"https://example.nl/path?x=1", "example.nl"},
		{"example.nl:443", "example.nl"},
	}
	for _, tc := range cases {
		got, err := NormaliseDomain(tc.in)
		if err != nil {
			t.Errorf("NormaliseDomain(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormaliseDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormaliseDomain_RejectsBareLabel(t *testing.T) {
	if _, err := NormaliseDomain("webapp-01"); err == nil {
		t.Errorf("expected TLD error for bare label, got nil")
	}
}

func TestNormaliseHost_AcceptsBareLabel(t *testing.T) {
	got, err := NormaliseHost("WEBAPP-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "webapp-01" {
		t.Errorf("got %q, want webapp-01", got)
	}
}

func TestNormaliseHost_RejectsURLSyntax(t *testing.T) {
	for _, bad := range []string{"http://h", "h/path", "h?x=1", ""} {
		if _, err := NormaliseHost(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestTargetValidate_DomainKindDefault(t *testing.T) {
	tgt := &Target{Domain: "Example.NL."}
	if err := tgt.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if tgt.Kind != TargetKindDomain {
		t.Errorf("default kind = %q, want %q", tgt.Kind, TargetKindDomain)
	}
	if tgt.Domain != "example.nl" {
		t.Errorf("normalised domain = %q", tgt.Domain)
	}
}

func TestTargetValidate_HostKindAcceptsBareHostname(t *testing.T) {
	tgt := &Target{Domain: "wanderer-test-host", Kind: TargetKindHost}
	if err := tgt.Validate(); err != nil {
		t.Fatalf("host kind should accept bare label: %v", err)
	}
	if tgt.Domain != "wanderer-test-host" {
		t.Errorf("normalised host = %q", tgt.Domain)
	}
}

func TestTargetValidate_DomainKindStillRejectsBareHostname(t *testing.T) {
	tgt := &Target{Domain: "no-tld-here", Kind: TargetKindDomain}
	err := tgt.Validate()
	if err == nil {
		t.Fatalf("domain kind should reject bare label, got nil")
	}
	if !strings.Contains(err.Error(), "no TLD") {
		t.Errorf("expected TLD error, got %v", err)
	}
}

func TestTargetValidate_UnknownKindRejected(t *testing.T) {
	tgt := &Target{Domain: "example.nl", Kind: TargetKind("rocket")}
	if err := tgt.Validate(); err == nil {
		t.Fatalf("unknown kind should error")
	}
}
