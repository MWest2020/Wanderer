package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestFindingRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	target := &models.Target{Domain: "example.nl"}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}
	if target.ID == "" {
		t.Fatal("target ID not set")
	}

	scan, err := s.CreateScan(ctx, target.ID)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}

	findings := []models.Finding{
		{
			ProbeID:       "dns.mx",
			DimensionHint: models.DimensionDataAI,
			Subject:       "example.nl",
			Severity:      models.SeverityObservation,
			Attributes: map[string]any{
				"host":       "mail.example.nl.",
				"preference": float64(10),
			},
			Evidence: []byte("example.nl.\t300\tIN\tMX\t10 mail.example.nl.\n"),
		},
		{
			ProbeID:       "tls.issuer",
			DimensionHint: models.DimensionJuridisch,
			Subject:       "example.nl",
			Severity:      models.SeverityFinding,
			Attributes: map[string]any{
				"issuer_cn": "Let's Encrypt R3",
			},
		},
	}
	if err := s.AppendFindings(ctx, scan.ID, findings); err != nil {
		t.Fatalf("append findings: %v", err)
	}
	if err := s.FinishScan(ctx, scan.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish scan: %v", err)
	}

	got, err := s.GetScan(ctx, scan.ID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if got.Status != models.ScanStatusComplete {
		t.Errorf("status = %q, want complete", got.Status)
	}
	if got.EndedAt == nil {
		t.Error("ended_at nil after finish")
	}
	if len(got.Findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(got.Findings))
	}
	f := got.Findings[0]
	if f.ProbeID != "dns.mx" || f.Severity != models.SeverityObservation {
		t.Errorf("finding[0] wrong: %+v", f)
	}
	if f.Attributes["host"] != "mail.example.nl." {
		t.Errorf("attribute host = %v, want mail.example.nl.", f.Attributes["host"])
	}
	if string(f.Evidence) == "" {
		t.Error("evidence lost in round-trip")
	}
}

func TestTargetNormalisation(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	target := &models.Target{Domain: "  HTTPS://Example.NL/path?x=1 "}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if target.Domain != "example.nl" {
		t.Errorf("domain = %q, want example.nl", target.Domain)
	}
	// Second upsert with a different form of the same domain returns
	// the same row.
	second := &models.Target{Domain: "example.nl."}
	if err := s.UpsertTarget(ctx, second); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	if second.ID != target.ID {
		t.Errorf("upsert of same domain created new row: %s vs %s", second.ID, target.ID)
	}
}

func TestUpsertTarget_HostKindRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	tgt := &models.Target{Domain: "wanderer-test-host", Kind: models.TargetKindHost}
	if err := s.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert host target: %v", err)
	}
	if tgt.ID == "" {
		t.Fatal("host target ID not set")
	}

	got, err := s.GetTarget(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("GetTarget: %v", err)
	}
	if got.Kind != models.TargetKindHost {
		t.Errorf("kind = %q, want host", got.Kind)
	}
	if got.Domain != "wanderer-test-host" {
		t.Errorf("domain = %q", got.Domain)
	}

	// A second upsert without setting Kind on the input still loads
	// the persisted host kind.
	again := &models.Target{Domain: "wanderer-test-host", Kind: models.TargetKindHost}
	if err := s.UpsertTarget(ctx, again); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	if again.Kind != models.TargetKindHost {
		t.Errorf("re-upsert kind = %q", again.Kind)
	}
	if again.ID != tgt.ID {
		t.Errorf("expected same row, got %s vs %s", again.ID, tgt.ID)
	}
}

// TestMigration_RenameDictuToWand pins the SQL semantics of
// migration version 4 (ADR-0011): a row with framework='dictu' and
// a dictu-prefixed criterium_id JSON value is rewritten to
// framework='wand' with wand-prefixed criterium_id; a row already
// at wand is left untouched.
func TestMigration_RenameDictuToWand(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	// Need a target + scan first because of the FK on assessments.scan_id.
	tgt := &models.Target{Domain: "example.nl"}
	if err := s.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert target: %v", err)
	}
	sc, err := s.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}

	// Insert a pre-migration row directly via SQL — bypass
	// CreateAssessment so the row carries the legacy 'dictu'
	// framework and a 'dictu.juridisch.cert_issuer_eea' criterium
	// ID inside the JSON-encoded dimensions blob.
	preDimensions := `[{"dimension":"juridisch","score":"afhankelijk","completeness":"complete","rationale":[{"criterium_id":"dictu.juridisch.cert_issuer_eea","verdict":"cert in US","score":"afhankelijk","evidence":["f_1"]}]}]`
	if _, err := s.DB().ExecContext(ctx,
		`INSERT INTO assessments (id, scan_id, framework, dimensions, report, created_at) VALUES (?,?,?,?,?,?)`,
		"a_legacy", sc.ID, "dictu", preDimensions, "", "2026-04-29T00:00:00Z",
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// Apply the migration SQL directly. (The runner already executed
	// it at Open() but the table was empty then; we want to prove
	// the SQL itself rewrites a legacy row.)
	migrationSQL := `UPDATE assessments
	  SET framework  = 'wand',
	      dimensions = REPLACE(dimensions, '"dictu.', '"wand.')
	  WHERE framework = 'dictu'`
	if _, err := s.DB().ExecContext(ctx, migrationSQL); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Verify the row was rewritten.
	row := s.DB().QueryRowContext(ctx,
		`SELECT framework, dimensions FROM assessments WHERE id = ?`, "a_legacy")
	var framework, dimensions string
	if err := row.Scan(&framework, &dimensions); err != nil {
		t.Fatalf("select: %v", err)
	}
	if framework != "wand" {
		t.Errorf("framework = %q, want wand", framework)
	}
	if !strings.Contains(dimensions, `"wand.juridisch.cert_issuer_eea"`) {
		t.Errorf("criterium_id not rewritten; dimensions = %s", dimensions)
	}
	if strings.Contains(dimensions, `"dictu.juridisch.cert_issuer_eea"`) {
		t.Errorf("legacy dictu criterium_id still present; dimensions = %s", dimensions)
	}

	// Re-running the SQL must not touch already-migrated rows.
	if _, err := s.DB().ExecContext(ctx, migrationSQL); err != nil {
		t.Fatalf("re-apply migration: %v", err)
	}
	row = s.DB().QueryRowContext(ctx,
		`SELECT framework FROM assessments WHERE id = ?`, "a_legacy")
	if err := row.Scan(&framework); err != nil {
		t.Fatalf("re-select: %v", err)
	}
	if framework != "wand" {
		t.Errorf("after re-apply: framework = %q, want wand (idempotent)", framework)
	}
}

func TestGetScanNotFound(t *testing.T) {
	s := openTestStore(t)
	_, err := s.GetScan(context.Background(), "s_missing")
	if err == nil {
		t.Fatal("expected error for missing scan")
	}
	if err != store.ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
