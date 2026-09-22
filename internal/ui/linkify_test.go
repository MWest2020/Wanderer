package ui

import (
	"strings"
	"testing"
)

// TestLinkifyVerdict_StandardsReportURL guards run 04 task 4.1: the
// Internet.nl report URL scoreStandardsCategory embeds in a rule's
// Verdict sentence (internal/assessor/wand/standards_rules.go
// "urlNote") must render as a real, clickable link — on a
// self-hosted netnl instance, not just internet.nl (design.md
// "Design gate outcome" #5).
func TestLinkifyVerdict_StandardsReportURL(t *testing.T) {
	in := "all 6 tested DNSSEC test(s) passed — see https://netnl.westerweel.work/site/westerweel.work/485/"
	got := string(linkifyVerdict(in))
	want := `<a href="https://netnl.westerweel.work/site/westerweel.work/485/" rel="noopener noreferrer" target="_blank">https://netnl.westerweel.work/site/westerweel.work/485/</a>`
	if !strings.Contains(got, want) {
		t.Fatalf("linkifyVerdict(%q) = %q, want it to contain %q", in, got, want)
	}
	if !strings.HasPrefix(got, "all 6 tested DNSSEC test(s) passed — see ") {
		t.Fatalf("linkifyVerdict(%q) = %q, lost the surrounding text", in, got)
	}
}

// TestLinkifyVerdict_MultipleURLsCommaJoined covers a rule whose
// evidence spans two imported reports (web + mail), joined with
// ", " (standardsTally.reportURLs / strings.Join) — the trailing
// comma must not be swallowed into either link's href.
func TestLinkifyVerdict_MultipleURLsCommaJoined(t *testing.T) {
	in := "all 5 tested SPF/DKIM/DMARC test(s) passed — see https://netnl.example.org/site/a/1/, https://netnl.example.org/site/b/2/"
	got := string(linkifyVerdict(in))
	for _, url := range []string{
		"https://netnl.example.org/site/a/1/",
		"https://netnl.example.org/site/b/2/",
	} {
		want := `href="` + url + `"`
		if !strings.Contains(got, want) {
			t.Errorf("linkifyVerdict(%q) = %q, want %q", in, got, want)
		}
		if strings.Contains(got, `href="`+url+`,"`) {
			t.Errorf("linkifyVerdict(%q) swallowed the comma separator into the href", in)
		}
	}
}

// TestLinkifyVerdict_NoURLIsUnchangedButEscaped covers the common
// case — most rule verdicts carry no URL at all (task 4.1's "not
// measured" case included) — and confirms plain text with HTML
// metacharacters is still escaped, not passed through raw.
func TestLinkifyVerdict_NoURLIsUnchangedButEscaped(t *testing.T) {
	in := "no internetnl.* findings imported for DNSSEC & IPv6 <fleet> — not measured"
	got := string(linkifyVerdict(in))
	if strings.Contains(got, "<fleet>") {
		t.Fatalf("linkifyVerdict(%q) = %q, expected HTML-escaped text with no raw markup", in, got)
	}
	if strings.Contains(got, "<a ") {
		t.Fatalf("linkifyVerdict(%q) = %q, expected no link when the text carries no URL", in, got)
	}
}

// TestLinkifyVerdict_NonHTTPSchemeNeverLinked defends the "opaque,
// never built, never trusted blindly" rule (design.md #5): report_url
// is carried verbatim from an imported file, so a non-http(s) scheme
// slipping past the pattern (e.g. a bare "https://" with no host)
// must not turn into a clickable link.
func TestLinkifyVerdict_NonHTTPSchemeNeverLinked(t *testing.T) {
	in := "measurement stale — see https:///no-host-here"
	got := string(linkifyVerdict(in))
	if strings.Contains(got, "<a ") {
		t.Fatalf("linkifyVerdict(%q) = %q, expected no link for a hostless URL", in, got)
	}
}
