package packages

import (
	"context"
	"strings"
	"testing"
)

func TestParseDpkg(t *testing.T) {
	raw := `bash 5.2.21-1 amd64 install ok installed
systemd 252.22-1ubuntu3 amd64 install ok installed
half-installed-pkg 1.0 amd64 deinstall ok config-files
`
	got := parseDpkg(raw)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d (third should be skipped)", len(got))
	}
	for _, f := range got {
		if f.ProbeID != "inventory.packages.dpkg" {
			t.Errorf("probe = %s", f.ProbeID)
		}
	}
}

func TestParseDpkg_EOLFlagged(t *testing.T) {
	raw := "php 7.4.30 amd64 install ok installed\n"
	got := parseDpkg(raw)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if string(got[0].Severity) != "observation" {
		t.Errorf("EOL php should be observation, got %s", got[0].Severity)
	}
}

func TestParseRpm(t *testing.T) {
	raw := "bash 5.1.16-12.el9 x86_64\nphp 7.4.30-1.el9 x86_64\n"
	got := parseRpm(raw)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[1].Subject != "php" || string(got[1].Severity) != "observation" {
		t.Errorf("expected EOL php from rpm, got %+v", got[1])
	}
}

func TestDpkgInspector_AvailableViaInjectedFunc(t *testing.T) {
	d := Dpkg{
		QueryFunc: func(_ context.Context) (string, error) {
			return "curl 7.88.1-10 amd64 install ok installed\n", nil
		},
	}
	ok, _ := d.Available()
	if !ok {
		t.Fatalf("want available with injected func")
	}
	got, err := d.Inspect(context.Background())
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(got) != 1 || got[0].Subject != "curl" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"7.4", "8.1", true},
		{"8.1", "8.1", false},
		{"8.2", "8.1", false},
		{"3.0.1", "3.0.2", true},
		{"7.4.30", "8.1", true},
	}
	for _, c := range cases {
		if got := versionLess(c.a, c.b); got != c.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestParseDpkg_TabSeparatedEmitsMaintainer(t *testing.T) {
	raw := "bash\t5.2.21-1\tamd64\tBash Maintainers <bash@packages.debian.org>\tinstall ok installed\n" +
		"systemd\t252.22-1ubuntu3\tamd64\tUbuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>\tinstall ok installed\n" +
		"half-installed\t1.0\tamd64\tSomeone <x@y.example>\tdeinstall ok config-files\n"
	got := parseDpkg(raw)
	if len(got) != 2 {
		t.Fatalf("want 2 (third half-installed skipped), got %d", len(got))
	}
	if got[0].Subject != "bash" {
		t.Fatalf("subject = %s, want bash", got[0].Subject)
	}
	m, _ := got[0].Attributes["maintainer"].(string)
	if !strings.Contains(m, "bash@packages.debian.org") {
		t.Errorf("maintainer = %q, want substring of email", m)
	}
	// Status field still captured.
	if s, _ := got[0].Attributes["status"].(string); !strings.Contains(s, "installed") {
		t.Errorf("status = %q, want substring 'installed'", s)
	}
}

func TestParseRpm_TabSeparatedEmitsVendor(t *testing.T) {
	raw := "bash\t5.2.32-1.fc42\tx86_64\tFedora Project\n" +
		"vim\t9.1-3.el9\tx86_64\tRed Hat, Inc.\n" +
		"local-pkg\t1.0-1\tx86_64\t(none)\n"
	got := parseRpm(raw)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if v, _ := got[0].Attributes["vendor"].(string); v != "Fedora Project" {
		t.Errorf("bash vendor = %q, want Fedora Project", v)
	}
	if v, _ := got[1].Attributes["vendor"].(string); v != "Red Hat, Inc." {
		t.Errorf("vim vendor = %q, want Red Hat, Inc.", v)
	}
	// (none) is the placeholder for locally-built packages.
	if _, hasVendor := got[2].Attributes["vendor"]; hasVendor {
		t.Errorf("local-pkg should have no vendor attribute when rpm reports (none)")
	}
}

func TestParseDpkg_HandlesShortLines(t *testing.T) {
	raw := "incomplete line\n"
	got := parseDpkg(raw)
	if len(got) != 0 {
		t.Errorf("want 0 from short line, got %d", len(got))
	}
	// Ensure not panic on empty/whitespace.
	got = parseDpkg("\n\n   \n")
	if len(got) != 0 {
		t.Errorf("want 0 from whitespace, got %d", len(got))
	}
	// Exercise the trim path.
	got = parseDpkg(strings.Repeat("\n", 5))
	if len(got) != 0 {
		t.Errorf("want 0, got %d", len(got))
	}
}
