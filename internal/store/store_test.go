package store_test

import (
	"context"
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
