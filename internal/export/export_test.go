package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seed(t *testing.T, st *store.Store) string {
	t.Helper()
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	findings := []models.Finding{
		{ProbeID: "tls.issuer", DimensionHint: models.DimensionJuridisch, Subject: "example.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"issuer_country": []string{"US"}}, Evidence: []byte("pem-data")},
		{ProbeID: "tls.validity", DimensionHint: models.DimensionOperationeel, Subject: "example.nl", Severity: models.SeverityInfo, Attributes: map[string]any{"days_left": 90}},
		{ProbeID: "dns.mx", DimensionHint: models.DimensionDataAI, Subject: "example.nl", Severity: models.SeverityObservation, Attributes: map[string]any{"host": "mail.example.nl"}},
	}
	if err := st.AppendFindings(ctx, sc.ID, findings); err != nil {
		t.Fatalf("append: %v", err)
	}
	return sc.ID
}

func TestFindingsCSV_HasHeaderAndRows(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsCSV(context.Background(), &buf, st, store.Selectors{}); err != nil {
		t.Fatalf("write: %v", err)
	}
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) != 4 { // header + 3
		t.Fatalf("want 4 rows (header + 3), got %d", len(records))
	}
	if records[0][0] != "id" {
		t.Errorf("first column of header = %q, want id", records[0][0])
	}
	// Column count stable across rows.
	for i, r := range records {
		if len(r) != len(FindingsCSVHeader) {
			t.Errorf("row %d has %d columns, want %d", i, len(r), len(FindingsCSVHeader))
		}
	}
}

func TestFindingsCSV_EmptySetHeaderOnly(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsCSV(context.Background(), &buf, st, store.Selectors{ScanID: "s_missing"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "id,scan_id,probe_id") {
		t.Errorf("want header, got %q", buf.String()[:min(60, buf.Len())])
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("want header-only, got %d rows", len(records))
	}
}

func TestFindingsCSV_Deterministic(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var a, b bytes.Buffer
	if err := WriteFindingsCSV(context.Background(), &a, st, store.Selectors{}); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := WriteFindingsCSV(context.Background(), &b, st, store.Selectors{}); err != nil {
		t.Fatalf("write b: %v", err)
	}
	if a.String() != b.String() {
		t.Errorf("CSV output is not deterministic\nA:\n%s\nB:\n%s", a.String(), b.String())
	}
}

func TestFindingsCSV_SelectorPushdown(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsCSV(context.Background(), &buf, st, store.Selectors{ProbePref: "tls", Dimension: "juridisch"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 2 { // header + 1 row (tls.issuer is the only juridisch tls finding)
		t.Fatalf("want header+1, got %d rows", len(records))
	}
	if records[1][2] != "tls.issuer" || records[1][3] != "juridisch" {
		t.Errorf("wrong row after filter: %v", records[1])
	}
}

func TestFindingsJSONL_OneObjectPerLine(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsJSONL(context.Background(), &buf, st, store.Selectors{}, true); err != nil {
		t.Fatalf("write: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d not valid JSON: %v", i, err)
		}
	}
}

func TestFindingsJSONL_EvidenceBase64(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsJSONL(context.Background(), &buf, st, store.Selectors{ProbePref: "tls"}, true); err != nil {
		t.Fatalf("write: %v", err)
	}
	foundEvidence := false
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		var m map[string]any
		_ = json.Unmarshal([]byte(line), &m)
		if ev, ok := m["evidence"].(string); ok && ev != "" {
			foundEvidence = true
			if ev == "pem-data" {
				t.Errorf("evidence should be base64, got raw: %q", ev)
			}
		}
	}
	if !foundEvidence {
		t.Error("no line exposed base64 evidence")
	}
}

func TestFindingsJSONL_EvidenceOmitted(t *testing.T) {
	st := newTestStore(t)
	_ = seed(t, st)
	var buf bytes.Buffer
	if err := WriteFindingsJSONL(context.Background(), &buf, st, store.Selectors{}, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	if strings.Contains(buf.String(), `"evidence"`) {
		t.Errorf("evidence should be absent when includeEvidence=false; got: %s", buf.String())
	}
}

func TestScansCSV_HasCountColumn(t *testing.T) {
	st := newTestStore(t)
	scanID := seed(t, st)
	_ = scanID
	var buf bytes.Buffer
	if err := WriteScansCSV(context.Background(), &buf, st, store.Selectors{}); err != nil {
		t.Fatalf("write: %v", err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("want header+1 scan, got %d", len(records))
	}
	// finding_count column = last
	if records[1][len(ScansCSVHeader)-1] != "3" {
		t.Errorf("finding_count = %q, want 3", records[1][len(ScansCSVHeader)-1])
	}
}

func TestAssessmentsCSV_OneRowPerDimension(t *testing.T) {
	st := newTestStore(t)
	scanID := seed(t, st)
	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{
			{Dimension: models.DimensionJuridisch, Score: models.ScoreAfhankelijk, Completeness: models.CompletenessComplete},
			{Dimension: models.DimensionOperationeel, Score: models.ScoreVoldoende, Completeness: models.CompletenessComplete},
		},
	}
	if err := st.CreateAssessment(context.Background(), a); err != nil {
		t.Fatalf("create: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteAssessmentsCSV(context.Background(), &buf, st, store.Selectors{}); err != nil {
		t.Fatalf("write: %v", err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 3 { // header + 2 dimensions
		t.Fatalf("want header+2, got %d", len(records))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
