package store

import (
	"context"
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
