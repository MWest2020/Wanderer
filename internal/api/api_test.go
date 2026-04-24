package api_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubProbe struct{}

func (stubProbe) ID() string { return "stub" }
func (stubProbe) Run(_ context.Context, t models.Target, _ probe.Config) ([]models.Finding, error) {
	return []models.Finding{{
		ProbeID:    "stub.hello",
		Subject:    t.Domain,
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{"ok": true},
	}}, nil
}

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	h := api.Router(st, sc, sc.Logger)
	srv := httptest.NewServer(h)
	t.Cleanup(func() {
		srv.Close()
		_ = st.Close()
	})
	return srv, st
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestPostScanAndGet(t *testing.T) {
	srv, _ := newTestServer(t)

	body := strings.NewReader(`{"domain":"example.nl"}`)
	resp, err := http.Post(srv.URL+"/scans", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	var scan models.Scan
	if err := json.NewDecoder(resp.Body).Decode(&scan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if scan.ID == "" {
		t.Fatal("empty scan ID")
	}

	resp2, err := http.Get(srv.URL + "/scans/" + scan.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp2.StatusCode)
	}
}

func TestGetScanNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/scans/s_missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] == nil {
		t.Error("error field missing")
	}
}

func TestPostScanBadJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Post(srv.URL+"/scans", "application/json", strings.NewReader("not-json"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPostScanMissingDomain(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Post(srv.URL+"/scans", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestAssessmentLifecycle drives POST /scans → POST /scans/{id}/assessments
// → GET /assessments/{id} end-to-end against the stub probe.
func TestAssessmentLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)

	// First create a scan so we have an ID to assess.
	resp, err := http.Post(srv.URL+"/scans", "application/json", strings.NewReader(`{"domain":"example.nl"}`))
	if err != nil {
		t.Fatalf("scan post: %v", err)
	}
	var scan models.Scan
	if err := json.NewDecoder(resp.Body).Decode(&scan); err != nil {
		t.Fatalf("decode scan: %v", err)
	}
	resp.Body.Close()
	if scan.ID == "" {
		t.Fatal("empty scan ID")
	}

	// Request an assessment.
	aresp, err := http.Post(srv.URL+"/scans/"+scan.ID+"/assessments", "application/json", strings.NewReader(``))
	if err != nil {
		t.Fatalf("assess post: %v", err)
	}
	defer aresp.Body.Close()
	if aresp.StatusCode != 201 {
		t.Fatalf("status = %d, want 201", aresp.StatusCode)
	}
	var a models.Assessment
	if err := json.NewDecoder(aresp.Body).Decode(&a); err != nil {
		t.Fatalf("decode assessment: %v", err)
	}
	if a.ID == "" {
		t.Fatal("empty assessment ID")
	}
	if len(a.Dimensions) != 5 {
		t.Errorf("want 5 dimensions, got %d", len(a.Dimensions))
	}
	if a.Report == "" {
		t.Error("markdown report missing")
	}

	// Retrieve the persisted assessment.
	gresp, err := http.Get(srv.URL + "/assessments/" + a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gresp.Body.Close()
	if gresp.StatusCode != 200 {
		t.Fatalf("get status = %d", gresp.StatusCode)
	}
	var got models.Assessment
	if err := json.NewDecoder(gresp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != a.ID || got.ScanID != a.ScanID {
		t.Errorf("round trip diverged: %+v vs %+v", got, a)
	}
}

func TestPostAssessmentMissingScan(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Post(srv.URL+"/scans/s_missing/assessments", "application/json", strings.NewReader(``))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetAssessmentNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/assessments/a_missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
