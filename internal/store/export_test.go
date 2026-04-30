package store

import (
	"context"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

func seedFindings(t *testing.T, st *Store, scanID string) {
	t.Helper()
	ctx := context.Background()
	findings := []models.Finding{
		{ProbeID: "tls.issuer", DimensionHint: models.DimensionJuridisch, Subject: "example.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"issuer_country": []string{"US"}}},
		{ProbeID: "tls.validity", DimensionHint: models.DimensionOperationeel, Subject: "example.nl", Severity: models.SeverityInfo, Attributes: map[string]any{"days_left": 90}},
		{ProbeID: "dns.mx", DimensionHint: models.DimensionDataAI, Subject: "example.nl", Severity: models.SeverityObservation, Attributes: map[string]any{"host": "mail.example.nl"}},
	}
	if err := st.AppendFindings(ctx, scanID, findings); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func drain(t *testing.T, it *FindingsIter) []models.Finding {
	t.Helper()
	defer it.Close()
	var out []models.Finding
	for it.Next() {
		f, err := it.Finding()
		if err != nil {
			t.Fatalf("finding: %v", err)
		}
		out = append(out, f)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("err: %v", err)
	}
	return out
}

func TestListFindings_AllAndScanFilter(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)
	seedFindings(t, st, scanID)

	// No filter: three findings.
	it, err := st.ListFindings(context.Background(), Selectors{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := drain(t, it)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}

	// Scan filter: same three.
	it, err = st.ListFindings(context.Background(), Selectors{ScanID: scanID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got = drain(t, it)
	if len(got) != 3 {
		t.Fatalf("scan filter: want 3, got %d", len(got))
	}

	// Scan filter mismatch: zero.
	it, err = st.ListFindings(context.Background(), Selectors{ScanID: "s_missing"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got = drain(t, it)
	if len(got) != 0 {
		t.Fatalf("bogus scan: want 0, got %d", len(got))
	}
}

func TestListFindings_ProbePrefix(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)
	seedFindings(t, st, scanID)

	it, err := st.ListFindings(context.Background(), Selectors{ProbePref: "tls"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := drain(t, it)
	if len(got) != 2 {
		t.Fatalf("tls prefix: want 2, got %d", len(got))
	}
	for _, f := range got {
		if f.ProbeID[:4] != "tls." {
			t.Errorf("unexpected probe %s in tls-prefix result", f.ProbeID)
		}
	}
}

func TestListFindings_DimensionFilter(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)
	seedFindings(t, st, scanID)

	it, err := st.ListFindings(context.Background(), Selectors{Dimension: "juridisch"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := drain(t, it)
	if len(got) != 1 {
		t.Fatalf("juridisch: want 1, got %d", len(got))
	}
	if got[0].DimensionHint != models.DimensionJuridisch {
		t.Errorf("dimension mismatch: %s", got[0].DimensionHint)
	}
}

func TestListFindings_SinceUntil(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)
	seedFindings(t, st, scanID)

	// Since in the past: all rows.
	since := time.Now().Add(-time.Hour)
	it, err := st.ListFindings(context.Background(), Selectors{Since: since})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := drain(t, it)
	if len(got) != 3 {
		t.Fatalf("since past: want 3, got %d", len(got))
	}

	// Until in the past: none.
	until := time.Now().Add(-time.Hour)
	it, err = st.ListFindings(context.Background(), Selectors{Until: until})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got = drain(t, it)
	if len(got) != 0 {
		t.Fatalf("until past: want 0, got %d", len(got))
	}
}

func TestListScans_IncludesFindingCount(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)
	seedFindings(t, st, scanID)

	rows, err := st.ListScans(context.Background(), Selectors{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 scan row, got %d", len(rows))
	}
	if rows[0].FindingCount != 3 {
		t.Errorf("finding count = %d, want 3", rows[0].FindingCount)
	}
	if rows[0].Domain != "example.nl" {
		t.Errorf("domain = %q", rows[0].Domain)
	}
}

func TestListAssessments(t *testing.T) {
	st := openTestStore(t)
	scanID := seedScan(t, st)

	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{
			{Dimension: models.DimensionJuridisch, Score: models.ScoreVoldoende, Completeness: models.CompletenessComplete},
		},
	}
	if err := st.CreateAssessment(context.Background(), a); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := st.ListAssessments(context.Background(), Selectors{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1, got %d", len(list))
	}
	if list[0].ScanID != scanID {
		t.Errorf("scan ID mismatch")
	}
}
