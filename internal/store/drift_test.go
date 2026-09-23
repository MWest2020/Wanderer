package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestPreviousScanForTarget(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// No prior scan: ErrNotFound.
	_, err := st.PreviousScanForTarget(ctx, tgt.ID, time.Now())
	if err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}

	// Two scans, fetch the previous of the newer.
	first, _ := st.CreateScan(ctx, tgt.ID)
	time.Sleep(5 * time.Millisecond)
	second, _ := st.CreateScan(ctx, tgt.ID)

	prev, err := st.PreviousScanForTarget(ctx, tgt.ID, second.StartedAt)
	if err != nil {
		t.Fatalf("previous: %v", err)
	}
	if prev.ID != first.ID {
		t.Errorf("want previous = %s, got %s", first.ID, prev.ID)
	}
}

// TestPreviousScanForTarget_SkipsImportScanBetweenPerimeterScans pins
// specs/scheduling/spec.md "Import between two perimeter scans": an
// import scan landing between two perimeter scans must never be
// picked as "the previous scan" — habitat run 02 found the drift
// engine diffing a perimeter scan against an import this way,
// reporting every perimeter Finding as added and every imported one
// as removed.
func TestPreviousScanForTarget_SkipsImportScanBetweenPerimeterScans(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	firstPerimeter, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("first perimeter scan: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	importScan, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("import scan: %v", err)
	}
	if err := st.AppendFindings(ctx, importScan.ID, []models.Finding{
		{ProbeID: "internetnl.dnssec", Subject: tgt.Domain, Severity: models.SeverityInfo, SourceModus: models.SourceModusImport, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append import finding: %v", err)
	}
	if err := st.FinishScan(ctx, importScan.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish import scan: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	secondPerimeter, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("second perimeter scan: %v", err)
	}

	prev, err := st.PreviousScanForTarget(ctx, tgt.ID, secondPerimeter.StartedAt)
	if err != nil {
		t.Fatalf("previous: %v", err)
	}
	if prev.ID != firstPerimeter.ID {
		t.Errorf("previous scan = %s, want the perimeter scan %s (the import scan must be skipped)", prev.ID, firstPerimeter.ID)
	}
}

// TestPreviousScanForTarget_OnlyImportBeforeIsTreatedAsNoBaseline pins
// specs/scheduling/spec.md "Only an import before the first perimeter
// scan": when the only earlier scan for a target is an import scan,
// the first perimeter scan must be treated as a baseline
// (ErrNotFound), not diffed against the import.
func TestPreviousScanForTarget_OnlyImportBeforeIsTreatedAsNoBaseline(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	importScan, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("import scan: %v", err)
	}
	if err := st.AppendFindings(ctx, importScan.ID, []models.Finding{
		{ProbeID: "internetnl.dnssec", Subject: tgt.Domain, Severity: models.SeverityInfo, SourceModus: models.SourceModusImport, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append import finding: %v", err)
	}
	if err := st.FinishScan(ctx, importScan.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish import scan: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	perimeter, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("perimeter scan: %v", err)
	}

	_, err = st.PreviousScanForTarget(ctx, tgt.ID, perimeter.StartedAt)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("previous scan err = %v, want ErrNotFound (import-only history is a baseline, not a previous scan)", err)
	}
}

func TestListDriftForTarget(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	scan, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	findings := []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"issuer_cn": "ZeroSSL"}},
		{ProbeID: "drift.tls.issuer_changed", Subject: "example.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"source_modus": "drift"}},
		{ProbeID: "drift.dns.mx_set_changed", Subject: "example.nl", Severity: models.SeverityObservation, Attributes: map[string]any{"source_modus": "drift"}},
	}
	if err := st.AppendFindings(ctx, scan.ID, findings); err != nil {
		t.Fatalf("append: %v", err)
	}

	drift, err := st.ListDriftForTarget(ctx, tgt.ID, time.Time{})
	if err != nil {
		t.Fatalf("list drift: %v", err)
	}
	if len(drift) != 2 {
		t.Errorf("want 2 drift findings, got %d", len(drift))
	}
	for _, f := range drift {
		if f.ProbeID[:6] != "drift." {
			t.Errorf("non-drift finding leaked: %s", f.ProbeID)
		}
	}

	// Since-filter past the drift creation time → zero results.
	future, err := st.ListDriftForTarget(ctx, tgt.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("list drift: %v", err)
	}
	if len(future) != 0 {
		t.Errorf("future since: want 0, got %d", len(future))
	}
}
