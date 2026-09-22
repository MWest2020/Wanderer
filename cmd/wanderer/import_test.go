package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestRunImportInternetnl_KnownTargetPersistsUnderImportScan(t *testing.T) {
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
	st.Close()

	rc := runImportInternetnl([]string{
		"--db", dbPath,
		filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json"),
	})
	if rc != 0 {
		t.Fatalf("runImportInternetnl exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st.Close()

	scan, err := st.LatestScanByModus(ctx, target.ID, models.SourceModusImport)
	if err != nil {
		t.Fatalf("LatestScanByModus: %v", err)
	}
	if len(scan.Findings) != 38 {
		t.Fatalf("findings = %d, want 38", len(scan.Findings))
	}
	for _, f := range scan.Findings {
		if f.SourceModus != models.SourceModusImport {
			t.Errorf("finding %s has SourceModus %q, want import", f.ProbeID, f.SourceModus)
		}
	}
}

// TestRunImportInternetnl_UnknownDomainSkippedExitsZero covers
// spec.md "Batch file for the fleet": an unknown domain is warned
// about and skipped, but the command still exits 0.
func TestRunImportInternetnl_UnknownDomainSkippedExitsZero(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	ctx := context.Background()

	st, err := store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	st.Close()

	// No target registered for westerweel.work — the fixture's only
	// domain is therefore unknown.
	rc := runImportInternetnl([]string{
		"--db", dbPath,
		filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json"),
	})
	if rc != 0 {
		t.Fatalf("runImportInternetnl exit = %d, want 0 (unknown domain is a warn+skip, not a failure)", rc)
	}
}

func TestRunImportInternetnl_SchemaMismatchAbortsWithNonZeroExit(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	badFile := filepath.Join(t.TempDir(), "bad-schema.json")
	if err := os.WriteFile(badFile, []byte(`{"schema":"netnl-findings/v2","domains":[]}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	rc := runImportInternetnl([]string{"--db", dbPath, badFile})
	if rc == 0 {
		t.Fatal("want non-zero exit for unsupported schema version")
	}
}

// TestRunImportInternetnl_ReimportSameFileIsNoOp covers spec.md
// "Re-importing an identical file SHALL be idempotent".
func TestRunImportInternetnl_ReimportSameFileIsNoOp(t *testing.T) {
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
	st.Close()

	fixture := filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json")
	if rc := runImportInternetnl([]string{"--db", dbPath, fixture}); rc != 0 {
		t.Fatalf("first import exit = %d, want 0", rc)
	}
	if rc := runImportInternetnl([]string{"--db", dbPath, fixture}); rc != 0 {
		t.Fatalf("second import exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st.Close()

	var scanCount int
	row := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM scans WHERE target_id = ?`, target.ID)
	if err := row.Scan(&scanCount); err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if scanCount != 1 {
		t.Fatalf("scan count after re-import = %d, want 1 (unchanged)", scanCount)
	}
}

func TestRunImport_UnknownSourceIsUsageError(t *testing.T) {
	rc := runImport([]string{"not-internetnl", "somefile"})
	if rc != 2 {
		t.Fatalf("exit = %d, want 2", rc)
	}
}
