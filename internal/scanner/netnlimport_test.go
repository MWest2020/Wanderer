package scanner

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

// TestParseNetnlFindings_WebFixture pins the importer against the
// golden web fixture (design.md "Design gate outcome"): 38 results
// across five categories with these exact counts. A drift here means
// the import is wrong, not the fixture.
func TestParseNetnlFindings_WebFixture(t *testing.T) {
	got, err := LoadNetnlFindings(filepath.Join("testdata", "findings-v1-web-20260922.json"), discardLogger())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Schema != NetnlSchemaV1 {
		t.Fatalf("schema = %q, want %q", got.Schema, NetnlSchemaV1)
	}
	if len(got.Domains) != 1 {
		t.Fatalf("domains = %d, want 1", len(got.Domains))
	}
	d := got.Domains[0]
	if d.Domain != "westerweel.work" || d.Type != "web" {
		t.Fatalf("domain/type = %q/%q, want westerweel.work/web", d.Domain, d.Type)
	}
	if len(d.Results) != 38 {
		t.Fatalf("results = %d, want 38", len(d.Results))
	}
	wantByCategory := map[string]int{
		"web_https":      22,
		"web_ipv6":       5,
		"web_appsecpriv": 5,
		"web_rpki":       4,
		"web_dnssec":     2,
	}
	assertCategoryCounts(t, d.Results, wantByCategory)
}

// TestParseNetnlFindings_MailFixture is the mail-side counterpart,
// including the one "error" status result (mail_starttls_tls_available)
// — design.md finding 2 confirms `error` is a real, non-theoretical
// verdict value.
func TestParseNetnlFindings_MailFixture(t *testing.T) {
	got, err := LoadNetnlFindings(filepath.Join("testdata", "findings-v1-mail-20260922.json"), discardLogger())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Domains) != 1 {
		t.Fatalf("domains = %d, want 1", len(got.Domains))
	}
	d := got.Domains[0]
	if d.Type != "mail" {
		t.Fatalf("type = %q, want mail", d.Type)
	}
	if len(d.Results) != 38 {
		t.Fatalf("results = %d, want 38", len(d.Results))
	}
	wantByCategory := map[string]int{
		"mail_starttls": 19,
		"mail_rpki":     6,
		"mail_auth":     5,
		"mail_dnssec":   4,
		"mail_ipv6":     4,
	}
	assertCategoryCounts(t, d.Results, wantByCategory)

	var sawError bool
	for _, r := range d.Results {
		if r.Test == "mail_starttls_tls_available" {
			sawError = true
			if r.Status != "error" || r.Verdict != "other" {
				t.Errorf("mail_starttls_tls_available = status=%q verdict=%q, want error/other", r.Status, r.Verdict)
			}
		}
	}
	if !sawError {
		t.Fatal("mail_starttls_tls_available not found")
	}
}

func TestParseNetnlFindings_SchemaMismatchAborts(t *testing.T) {
	in := strings.NewReader(`{"schema":"netnl-findings/v2","domains":[]}`)
	_, err := ParseNetnlFindings(in, discardLogger())
	if err == nil {
		t.Fatal("want error for unsupported schema version")
	}
	if !strings.Contains(err.Error(), NetnlSchemaV1) {
		t.Errorf("error %q does not name the expected version %q", err.Error(), NetnlSchemaV1)
	}
}

// TestParseNetnlFindings_TruncatedFinalEntry covers spec.md "Truncated
// file": the final domain entry is cut off mid-object. The two valid
// entries before it must survive; the broken one is warned about, not
// fatal.
func TestParseNetnlFindings_TruncatedFinalEntry(t *testing.T) {
	in := strings.NewReader(`{"schema":"netnl-findings/v1","domains":[` +
		`{"domain":"a.example.nl","type":"web","status":"ok","measured_at":"2026-09-22T10:00:00Z","results":[]},` +
		`{"domain":"b.example.nl","type":"web","status":"ok","measured_at":"2026-09-22T10:00:00Z","results":[]},` +
		`{"domain":"c.example.nl","type":"web`)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	got, err := ParseNetnlFindings(in, logger)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got.Domains) != 2 {
		t.Fatalf("domains = %d, want 2 (the valid entries before the truncation)", len(got.Domains))
	}
	if !strings.Contains(buf.String(), "scanner.netnl.malformed_entry") {
		t.Errorf("truncated entry should produce a warn log; got %s", buf.String())
	}
}

// TestParseNetnlFindings_MalformedEntrySkippedNeighboursKept exercises
// a structurally-valid-JSON entry that is missing a required field
// (no "domain"): it is skipped, but entries around it are kept —
// distinct from the truncated-file case where the stream itself
// breaks.
func TestParseNetnlFindings_MalformedEntrySkippedNeighboursKept(t *testing.T) {
	in := strings.NewReader(`{"schema":"netnl-findings/v1","domains":[` +
		`{"domain":"a.example.nl","type":"web","status":"ok","measured_at":"2026-09-22T10:00:00Z","results":[]},` +
		`{"type":"web","status":"ok","measured_at":"2026-09-22T10:00:00Z","results":[]},` +
		`{"domain":"c.example.nl","type":"web","status":"ok","measured_at":"2026-09-22T10:00:00Z","results":[]}` +
		`]}`)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	got, err := ParseNetnlFindings(in, logger)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got.Domains) != 2 {
		t.Fatalf("domains = %d, want 2 (a and c, b skipped)", len(got.Domains))
	}
	if got.Domains[0].Domain != "a.example.nl" || got.Domains[1].Domain != "c.example.nl" {
		t.Errorf("unexpected surviving domains: %+v", got.Domains)
	}
	if !strings.Contains(buf.String(), "missing domain") {
		t.Errorf("missing-domain entry should log the reason; got %s", buf.String())
	}
}

// TestParseNetnlFindings_ErrorStatusWithEmptyResultsIsNotMalformed
// covers the "valkuil" from design.md: a domain measured with
// status=error and empty results means "measured, it broke" — it is
// not a malformed entry and must not be skipped.
func TestParseNetnlFindings_ErrorStatusWithEmptyResultsIsNotMalformed(t *testing.T) {
	in := strings.NewReader(`{"schema":"netnl-findings/v1","domains":[` +
		`{"domain":"broken.example.nl","type":"mail","status":"error","measured_at":"2026-09-22T10:00:00Z","results":[]}` +
		`]}`)
	got, err := ParseNetnlFindings(in, discardLogger())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got.Domains) != 1 {
		t.Fatalf("domains = %d, want 1 (error status is a real result, not a skip)", len(got.Domains))
	}
	if got.Domains[0].Status != "error" || len(got.Domains[0].Results) != 0 {
		t.Errorf("unexpected domain: %+v", got.Domains[0])
	}
}

func TestLoadNetnlFindings_MissingFileIsError(t *testing.T) {
	_, err := LoadNetnlFindings(filepath.Join("does", "not", "exist.json"), nil)
	if err == nil {
		t.Error("missing file should error")
	}
}

func assertCategoryCounts(t *testing.T, results []NetnlResult, want map[string]int) {
	t.Helper()
	got := map[string]int{}
	for _, r := range results {
		got[r.Category]++
	}
	for cat, n := range want {
		if got[cat] != n {
			t.Errorf("category %q: got %d, want %d", cat, got[cat], n)
		}
	}
	if len(got) != len(want) {
		t.Errorf("category set = %v, want %v", got, want)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(discardWriter{}, nil))
}

func TestNetnlFindings_PerTestResults(t *testing.T) {
	got, err := LoadNetnlFindings(filepath.Join("testdata", "findings-v1-web-20260922.json"), discardLogger())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	findings := NetnlFindings(got.Domains[0])
	if len(findings) != 38 {
		t.Fatalf("findings = %d, want 38 (one per test result)", len(findings))
	}
	f := findings[0]
	if f.SourceModus != models.SourceModusImport {
		t.Errorf("SourceModus = %q, want %q", f.SourceModus, models.SourceModusImport)
	}
	if f.Subject != "westerweel.work" {
		t.Errorf("Subject = %q, want westerweel.work", f.Subject)
	}
	wantProbeID := "internetnl.web.web_appsecpriv_csp"
	if f.ProbeID != wantProbeID {
		t.Errorf("ProbeID = %q, want %q", f.ProbeID, wantProbeID)
	}
	if f.Attributes["report_url"] != "https://netnl.westerweel.work/site/westerweel.work/485/" {
		t.Errorf("report_url attribute = %v", f.Attributes["report_url"])
	}
	if f.Attributes["category"] != "web_appsecpriv" || f.Attributes["test"] != "web_appsecpriv_csp" {
		t.Errorf("unexpected category/test attributes: %+v", f.Attributes)
	}
	if err := f.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

// TestNetnlFindings_ErrorStatusEmptyResults covers the design.md
// "valkuil": a domain measured with status=error and no results still
// yields evidence — a single status Finding — rather than nothing.
func TestNetnlFindings_ErrorStatusEmptyResults(t *testing.T) {
	in := strings.NewReader(`{"schema":"netnl-findings/v1","domains":[` +
		`{"domain":"broken.example.nl","type":"mail","status":"error","measured_at":"2026-09-22T10:00:00Z","results":[]}` +
		`]}`)
	got, err := ParseNetnlFindings(in, discardLogger())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	findings := NetnlFindings(got.Domains[0])
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.ProbeID != "internetnl.mail.status" {
		t.Errorf("ProbeID = %q, want internetnl.mail.status", f.ProbeID)
	}
	if f.Severity != models.SeverityConcern {
		t.Errorf("Severity = %q, want concern (a broken measurement is worth flagging)", f.Severity)
	}
	if err := f.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
