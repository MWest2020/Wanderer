package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// TestRunAssess_CorrelatesImportedStandardsFindings pins the CLI half
// of the FindingsForAssessment correlation (tasks/2026-09-22-cli-assess-correlatie.md):
// `wanderer assess` on a perimeter scan must see internetnl.* findings
// imported separately for web and mail, the same way the API, UI,
// scheduler and MCP callers already do. Before the fix, `assess.go`
// scored scan.Findings alone, so a perimeter scan always reported the
// standards dimension as "not measured" even with both imports present
// in the same database.
func TestRunAssess_CorrelatesImportedStandardsFindings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	ctx := context.Background()

	st, err := store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	perimeterScan, err := st.CreateScan(ctx, target.ID)
	if err != nil {
		t.Fatalf("create perimeter scan: %v", err)
	}
	if err := st.AppendFindings(ctx, perimeterScan.ID, []models.Finding{
		{ProbeID: "dns.mx", Subject: target.Domain, Severity: models.SeverityInfo, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append perimeter finding: %v", err)
	}
	if err := st.FinishScan(ctx, perimeterScan.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish perimeter scan: %v", err)
	}
	st.Close()

	webFixture := filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json")
	mailFixture := filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-mail-20260922.json")
	if rc := runImportInternetnl([]string{"--db", dbPath, webFixture}); rc != 0 {
		t.Fatalf("import web exit = %d, want 0", rc)
	}
	if rc := runImportInternetnl([]string{"--db", dbPath, mailFixture}); rc != 0 {
		t.Fatalf("import mail exit = %d, want 0", rc)
	}

	if rc := runAssess([]string{"--db", dbPath, perimeterScan.ID}); rc != 0 {
		t.Fatalf("runAssess exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st.Close()

	assessments, err := st.ListAssessmentsForScan(ctx, perimeterScan.ID)
	if err != nil {
		t.Fatalf("list assessments: %v", err)
	}
	var wandAssessment *models.Assessment
	for i := range assessments {
		if assessments[i].Framework == "wand" {
			wandAssessment = &assessments[i]
			break
		}
	}
	if wandAssessment == nil {
		t.Fatal("no wand assessment persisted for the perimeter scan")
	}

	rpki := dimensionRationale(t, wandAssessment.Dimensions, "wand.standards.rpki")
	if len(rpki.Evidence) != 10 {
		t.Errorf("rpki evidence = %d, want 10 (web + mail imports, incl. nameserver subtests)", len(rpki.Evidence))
	}
	ipv6 := dimensionRationale(t, wandAssessment.Dimensions, "wand.standards.ipv6")
	if len(ipv6.Evidence) != 9 {
		t.Errorf("ipv6 evidence = %d, want 9 (web's 5 + mail's 4)", len(ipv6.Evidence))
	}
}

// dimensionRationale finds ruleID's Rationale entry inside dims
// without the caller needing to know which models.DimensionHint the
// rule lives under. Mirrors internal/store's standards_correlation_test.go
// helper of the same shape.
func dimensionRationale(t *testing.T, dims []models.DimensionScore, ruleID string) models.Rationale {
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
