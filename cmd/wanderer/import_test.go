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

// TestRunImportInternetnl_TargetAddedAfterUnknownDomainImportsOnRetry
// covers habitat run 02b: the first run warns and skips because no
// target exists yet for the fixture's domain; once the target is
// added, re-running the same file must actually persist that
// domain's findings instead of being treated as a no-op purely
// because the file hash was already seen.
func TestRunImportInternetnl_TargetAddedAfterUnknownDomainImportsOnRetry(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	ctx := context.Background()
	fixture := filepath.Join("..", "..", "internal", "scanner", "testdata", "findings-v1-web-20260922.json")

	st, err := store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	st.Close()

	// No target registered yet — the fixture's domain is unknown.
	if rc := runImportInternetnl([]string{"--db", dbPath, fixture}); rc != 0 {
		t.Fatalf("first import (unknown domain) exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	target := &models.Target{Domain: "westerweel.work"}
	if err := st.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}
	st.Close()

	// Same file, now that the target exists: this must import, not
	// be swallowed as "already imported (file unchanged)".
	if rc := runImportInternetnl([]string{"--db", dbPath, fixture}); rc != 0 {
		t.Fatalf("second import (target now exists) exit = %d, want 0", rc)
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
		t.Fatalf("findings = %d, want 38 — the retry after adding the target must actually persist them", len(scan.Findings))
	}
}

// TestRunImportInternetnl_MixedKnownAndUnknownDomains covers habitat
// run 02b's third scenario: a file with two domains where only one
// has a target. The known domain imports on the first run; the other
// only imports once its target is created, and re-running afterwards
// must not duplicate the domain that already came in.
func TestRunImportInternetnl_MixedKnownAndUnknownDomains(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	ctx := context.Background()

	file := filepath.Join(t.TempDir(), "mixed.json")
	const body = `{
  "schema": "netnl-findings/v1",
  "domains": [
    {
      "domain": "known.example.nl",
      "type": "web",
      "status": "ok",
      "measured_at": "2026-09-22T10:25:16Z",
      "score_percent": 95,
      "report_url": "https://netnl.example/site/known.example.nl/1/",
      "results": []
    },
    {
      "domain": "unknown.example.nl",
      "type": "web",
      "status": "ok",
      "measured_at": "2026-09-22T10:25:16Z",
      "score_percent": 50,
      "report_url": "https://netnl.example/site/unknown.example.nl/2/",
      "results": []
    }
  ]
}`
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	st, err := store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	known := &models.Target{Domain: "known.example.nl"}
	if err := st.UpsertTarget(ctx, known); err != nil {
		t.Fatalf("upsert known target: %v", err)
	}
	st.Close()

	if rc := runImportInternetnl([]string{"--db", dbPath, file}); rc != 0 {
		t.Fatalf("first import exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	if _, err := st.LatestScanByModus(ctx, known.ID, models.SourceModusImport); err != nil {
		t.Fatalf("LatestScanByModus(known) after first import: %v", err)
	}
	unknown := &models.Target{Domain: "unknown.example.nl"}
	if err := st.UpsertTarget(ctx, unknown); err != nil {
		t.Fatalf("upsert previously-unknown target: %v", err)
	}
	st.Close()

	if rc := runImportInternetnl([]string{"--db", dbPath, file}); rc != 0 {
		t.Fatalf("second import exit = %d, want 0", rc)
	}

	st, err = store.Open(ctx, "file:"+dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st.Close()

	if _, err := st.LatestScanByModus(ctx, unknown.ID, models.SourceModusImport); err != nil {
		t.Fatalf("LatestScanByModus(unknown) after target was created: %v", err)
	}

	var knownScanCount int
	row := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM scans WHERE target_id = ?`, known.ID)
	if err := row.Scan(&knownScanCount); err != nil {
		t.Fatalf("count scans for known target: %v", err)
	}
	if knownScanCount != 1 {
		t.Fatalf("known target scan count after second import = %d, want 1 (not re-imported)", knownScanCount)
	}
}

func TestRunImport_UnknownSourceIsUsageError(t *testing.T) {
	rc := runImport([]string{"not-internetnl", "somefile"})
	if rc != 2 {
		t.Fatalf("exit = %d, want 2", rc)
	}
}
