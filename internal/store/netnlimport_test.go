package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestGetTargetByDomain(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	target := &models.Target{Domain: "westerweel.work"}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	got, err := s.GetTargetByDomain(ctx, "westerweel.work")
	if err != nil {
		t.Fatalf("GetTargetByDomain: %v", err)
	}
	if got.ID != target.ID {
		t.Errorf("ID = %q, want %q", got.ID, target.ID)
	}

	if _, err := s.GetTargetByDomain(ctx, "unknown.example.nl"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown domain: err = %v, want ErrNotFound", err)
	}
}

func TestNetnlImportIdempotency(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	recorded, err := s.NetnlImportRecorded(ctx, "deadbeef", "westerweel.work")
	if err != nil {
		t.Fatalf("NetnlImportRecorded (before): %v", err)
	}
	if recorded {
		t.Fatal("unimported (hash, domain) reported as recorded")
	}

	if err := s.RecordNetnlImport(ctx, "deadbeef", "westerweel.work"); err != nil {
		t.Fatalf("RecordNetnlImport: %v", err)
	}

	recorded, err = s.NetnlImportRecorded(ctx, "deadbeef", "westerweel.work")
	if err != nil {
		t.Fatalf("NetnlImportRecorded (after): %v", err)
	}
	if !recorded {
		t.Fatal("recorded (hash, domain) reported as unimported")
	}

	// A different file hash is a distinct import, unaffected by the
	// first.
	recorded, err = s.NetnlImportRecorded(ctx, "otherhash", "westerweel.work")
	if err != nil {
		t.Fatalf("NetnlImportRecorded (other hash): %v", err)
	}
	if recorded {
		t.Fatal("unrelated hash reported as recorded")
	}

	// Same file hash, different domain: recording one domain from a
	// file must not mark another domain from that same file as done
	// (habitat run 02b — the key is (file_hash, domain), not file_hash
	// alone).
	recorded, err = s.NetnlImportRecorded(ctx, "deadbeef", "other.example.nl")
	if err != nil {
		t.Fatalf("NetnlImportRecorded (other domain): %v", err)
	}
	if recorded {
		t.Fatal("unrelated domain in the same file reported as recorded")
	}
}

func TestLatestScanByModus(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	target := &models.Target{Domain: "westerweel.work"}
	if err := s.UpsertTarget(ctx, target); err != nil {
		t.Fatalf("upsert target: %v", err)
	}

	// No scans at all yet.
	if _, err := s.LatestScanByModus(ctx, target.ID, models.SourceModusImport); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("before any scan: err = %v, want ErrNotFound", err)
	}

	perimeterScan, err := s.CreateScan(ctx, target.ID)
	if err != nil {
		t.Fatalf("create perimeter scan: %v", err)
	}
	if err := s.AppendFindings(ctx, perimeterScan.ID, []models.Finding{
		{ProbeID: "dns.mx", Subject: target.Domain, Severity: models.SeverityInfo, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append perimeter finding: %v", err)
	}

	// A perimeter scan exists, but no import scan yet.
	if _, err := s.LatestScanByModus(ctx, target.ID, models.SourceModusImport); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("perimeter-only: err = %v, want ErrNotFound for import modus", err)
	}
	got, err := s.LatestScanByModus(ctx, target.ID, models.SourceModusPerimeter)
	if err != nil {
		t.Fatalf("LatestScanByModus(perimeter): %v", err)
	}
	if got.ID != perimeterScan.ID {
		t.Errorf("perimeter scan ID = %q, want %q", got.ID, perimeterScan.ID)
	}

	importScan, err := s.CreateScan(ctx, target.ID)
	if err != nil {
		t.Fatalf("create import scan: %v", err)
	}
	if err := s.AppendFindings(ctx, importScan.ID, []models.Finding{
		{
			ProbeID:     "internetnl.web.web_dnssec_exist",
			SourceModus: models.SourceModusImport,
			Subject:     target.Domain,
			Severity:    models.SeverityInfo,
			Attributes:  map[string]any{},
		},
	}); err != nil {
		t.Fatalf("append import finding: %v", err)
	}

	// Both kinds coexist; each modus resolves to its own scan,
	// independent of which was created (or would sort) more recently
	// overall.
	gotImport, err := s.LatestScanByModus(ctx, target.ID, models.SourceModusImport)
	if err != nil {
		t.Fatalf("LatestScanByModus(import): %v", err)
	}
	if gotImport.ID != importScan.ID {
		t.Errorf("import scan ID = %q, want %q", gotImport.ID, importScan.ID)
	}
	gotPerimeter, err := s.LatestScanByModus(ctx, target.ID, models.SourceModusPerimeter)
	if err != nil {
		t.Fatalf("LatestScanByModus(perimeter) after import: %v", err)
	}
	if gotPerimeter.ID != perimeterScan.ID {
		t.Errorf("perimeter scan unchanged by later import: got %q, want %q", gotPerimeter.ID, perimeterScan.ID)
	}
}
