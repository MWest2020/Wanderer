package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// loadFreshNetnlFindings loads a netnl-findings/v1 golden fixture
// (task 1.3's testdata copies, measured against api.westerweel.work
// on 2026-09-22 — design.md "Design gate outcome") and rewrites every
// measured_at to now, so this test stays stable regardless of when it
// runs. Mirrors internal/assessor/wand's loadGoldenStandardsFindings /
// freshenMeasuredAt, duplicated here rather than exported across the
// package boundary for one helper.
func loadFreshNetnlFindings(t *testing.T, name string) []models.Finding {
	t.Helper()
	path := filepath.Join("..", "scanner", "testdata", name)
	f, err := scanner.LoadNetnlFindings(path, nil)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var out []models.Finding
	for _, d := range f.Domains {
		for _, fnd := range scanner.NetnlFindings(d) {
			fnd.Attributes["measured_at"] = now
			out = append(out, fnd)
		}
	}
	return out
}

// importScan persists findings as one completed import-kind scan for
// targetID and returns it re-read from the store (with Findings and
// their assigned IDs populated), the same shape assessor.Assess
// consumes in production.
func importScan(ctx context.Context, t *testing.T, s *store.Store, targetID string, findings []models.Finding) *models.Scan {
	t.Helper()
	sc, err := s.CreateScan(ctx, targetID)
	if err != nil {
		t.Fatalf("create import scan: %v", err)
	}
	if err := s.AppendFindings(ctx, sc.ID, findings); err != nil {
		t.Fatalf("append import findings: %v", err)
	}
	if err := s.FinishScan(ctx, sc.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish import scan: %v", err)
	}
	got, err := s.GetScan(ctx, sc.ID)
	if err != nil {
		t.Fatalf("re-read import scan: %v", err)
	}
	return got
}

// dimensionRule finds ruleID's Rationale entry inside dims without
// the caller needing to know which models.DimensionHint the rule
// lives under.
func dimensionRule(t *testing.T, dims []models.DimensionScore, ruleID string) models.Rationale {
	t.Helper()
	for _, d := range dims {
		for _, r := range d.Rationale {
			if r.CriteriumID == ruleID {
				return r
			}
		}
	}
	t.Fatalf("no rationale for rule %q", ruleID)
	return models.Rationale{}
}

// TestFindingsForAssessment_CorrelatesWebAndMailAcrossScans pins
// habitat run 03b's regression (runs/03b-correlatie-over-scansoorten.md):
// a target with a perimeter scan plus separate web- and mail-import
// scans (two netnl-findings files, per design.md "one file may carry
// many domains... one import call handles the whole fleet" — web and
// mail are separate files and therefore separate scans) must score
// the `standards` dimension identically no matter which of the three
// scans is assessed — design.md "Import semantics": "a perimeter scan
// never erases imported findings and vice versa." Before the fix,
// every caller scored scan.Findings alone: the perimeter scan saw
// zero internetnl.* evidence and each import scan saw only its own
// half (e.g. ipv6 soeverein on the web scan, voldoende on the mail
// scan, self-contradictory for one rule on one target).
func TestFindingsForAssessment_CorrelatesWebAndMailAcrossScans(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	target := &models.Target{Domain: "westerweel.work"}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	webFindings := loadFreshNetnlFindings(t, "findings-v1-web-20260922.json")
	mailFindings := loadFreshNetnlFindings(t, "findings-v1-mail-20260922.json")

	perimeterScan, err := s.CreateScan(ctx, target.ID)
	if err != nil {
		t.Fatalf("create perimeter scan: %v", err)
	}
	if err := s.AppendFindings(ctx, perimeterScan.ID, []models.Finding{
		{ProbeID: "dns.mx", Subject: target.Domain, Severity: models.SeverityInfo, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append perimeter finding: %v", err)
	}
	if err := s.FinishScan(ctx, perimeterScan.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish perimeter scan: %v", err)
	}
	perimeterScan, err = s.GetScan(ctx, perimeterScan.ID)
	if err != nil {
		t.Fatalf("re-read perimeter scan: %v", err)
	}

	webScan := importScan(ctx, t, s, target.ID, webFindings)
	mailScan := importScan(ctx, t, s, target.ID, mailFindings)

	rules := wand.DefaultRules()
	for _, tc := range []struct {
		name string
		scan *models.Scan
	}{
		{"perimeter", perimeterScan},
		{"web-import", webScan},
		{"mail-import", mailScan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := s.FindingsForAssessment(ctx, tc.scan)
			if err != nil {
				t.Fatalf("FindingsForAssessment: %v", err)
			}
			dims := assessor.Assess(findings, rules)

			ipv6 := dimensionRule(t, dims, "wand.standards.ipv6")
			if len(ipv6.Evidence) != 9 {
				t.Errorf("ipv6 evidence = %d, want 9 (design.md: web's 5 + mail's 4, 8 passed + 1 failed)", len(ipv6.Evidence))
			}
			if ipv6.Score != models.ScoreVoldoende {
				t.Errorf("ipv6 score = %s, want voldoende", ipv6.Score)
			}

			rpki := dimensionRule(t, dims, "wand.standards.rpki")
			if len(rpki.Evidence) != 10 {
				t.Errorf("rpki evidence = %d, want 10 (design.md: web_rpki + mail_rpki incl. nameserver subtests)", len(rpki.Evidence))
			}
			if rpki.Score != models.ScoreSoeverein {
				t.Errorf("rpki score = %s, want soeverein", rpki.Score)
			}
		})
	}
}

// TestFindingsForAssessment_FreshMailImportReplacesOnlyMail pins the
// other half of design.md "Import semantics": a second import of one
// type supersedes only that type's findings, never the other's, and
// this holds however the assessor later reaches the target (here,
// via the newest mail scan itself).
func TestFindingsForAssessment_FreshMailImportReplacesOnlyMail(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	target := &models.Target{Domain: "westerweel.work"}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	webFindings := loadFreshNetnlFindings(t, "findings-v1-web-20260922.json")
	mailFindingsOld := loadFreshNetnlFindings(t, "findings-v1-mail-20260922.json")

	importScan(ctx, t, s, target.ID, webFindings)
	importScan(ctx, t, s, target.ID, mailFindingsOld)

	// A fresh mail measurement: every mail_ipv6 test now passes (the
	// golden fixture has one failing), so wand.standards.ipv6 flips
	// from voldoende to soeverein if — and only if — this import
	// actually supersedes the old mail scan's findings rather than
	// adding to them.
	freshMail := loadFreshNetnlFindings(t, "findings-v1-mail-20260922.json")
	for i := range freshMail {
		if cat, ok := freshMail[i].Attributes["category"].(string); ok && cat == "mail_ipv6" {
			freshMail[i].Attributes["status"] = "passed"
		}
	}
	newMailScan := importScan(ctx, t, s, target.ID, freshMail)

	rules := wand.DefaultRules()
	findings, err := s.FindingsForAssessment(ctx, newMailScan)
	if err != nil {
		t.Fatalf("FindingsForAssessment: %v", err)
	}
	dims := assessor.Assess(findings, rules)

	ipv6 := dimensionRule(t, dims, "wand.standards.ipv6")
	if ipv6.Score != models.ScoreSoeverein {
		t.Errorf("ipv6 score after fresh mail import = %s, want soeverein (old mail_ipv6 failure must be gone, not doubled up with the new pass)", ipv6.Score)
	}
	if len(ipv6.Evidence) != 9 {
		t.Errorf("ipv6 evidence = %d, want 9 (still web's 5 + mail's 4 — replaced, not appended)", len(ipv6.Evidence))
	}

	// dnssec is untouched by either mail import: proof the web side
	// survives a second mail import unmodified.
	dnssec := dimensionRule(t, dims, "wand.standards.dnssec")
	if len(dnssec.Evidence) != 6 || dnssec.Score != models.ScoreSoeverein {
		t.Errorf("dnssec = %d tests / %s, want 6 / soeverein (web untouched by mail re-import)", len(dnssec.Evidence), dnssec.Score)
	}
}
