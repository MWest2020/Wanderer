package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/agent"
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

// findingsIngestServer wires a router with a single registered
// agent secret for the test cases below.
func findingsIngestServer(t *testing.T) (*httptest.Server, *store.Store, []byte) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	secret := []byte("agent-secret-fixture")
	secrets := api.NewStaticAgentSecrets(map[string][]byte{"webapp-01": secret})
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	srv := httptest.NewServer(api.RouterWithSecrets(st, sc, sc.Logger, secrets))
	t.Cleanup(srv.Close)
	return srv, st, secret
}

func seedScan(t *testing.T, st *store.Store) string {
	t.Helper()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	scan, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return scan.ID
}

func TestFindingsIngest_HappyPath(t *testing.T) {
	srv, st, secret := findingsIngestServer(t)
	scanID := seedScan(t, st)
	body := []byte(`{"findings":[{"probe_id":"inventory.systemd.service","subject":"sshd.service","severity":"info","attributes":{"active_state":"active"}}]}`)
	ts, sig := agent.Sign(secret, body, time.Now().UTC())
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "webapp-01")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	scan, err := st.GetScan(context.Background(), scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if len(scan.Findings) != 1 {
		t.Fatalf("want 1 persisted finding, got %d", len(scan.Findings))
	}
	if scan.Findings[0].SourceModus != models.SourceModusInventory {
		t.Errorf("source_modus = %s, want inventory", scan.Findings[0].SourceModus)
	}
}

func TestFindingsIngest_Replay(t *testing.T) {
	srv, st, secret := findingsIngestServer(t)
	scanID := seedScan(t, st)
	body := []byte(`{"findings":[]}`)
	ts, sig := agent.Sign(secret, body, time.Now().UTC().Add(-10*time.Minute))
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "webapp-01")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("replay accepted; status = %d", resp.StatusCode)
	}
}

func TestFindingsIngest_WrongSecret(t *testing.T) {
	srv, _, _ := findingsIngestServer(t)
	scanID := "anything"
	body := []byte(`{"findings":[]}`)
	ts, sig := agent.Sign([]byte("wrong"), body, time.Now().UTC())
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/"+scanID+"/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "webapp-01")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestFindingsIngest_UnknownHostname(t *testing.T) {
	srv, _, secret := findingsIngestServer(t)
	body := []byte(`{}`)
	ts, sig := agent.Sign(secret, body, time.Now().UTC())
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scans/something/findings", bytes.NewReader(body))
	req.Header.Set(agent.HeaderHostname, "stranger-host")
	req.Header.Set(agent.HeaderTimestamp, ts)
	req.Header.Set(agent.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGetTargetDrift_EmptyAndBadSince(t *testing.T) {
	srv, _ := newTestServer(t)

	// Bad since: 400.
	resp, err := http.Get(srv.URL + "/targets/t_anything/drift?since=garbage")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	// Unknown target with no since: 200 with empty findings.
	resp2, err := http.Get(srv.URL + "/targets/t_missing/drift")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp2.StatusCode)
	}
	var body struct {
		TargetID string `json:"target_id"`
		Findings []any  `json:"findings"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.TargetID != "t_missing" {
		t.Errorf("target_id = %q", body.TargetID)
	}
	if len(body.Findings) != 0 {
		t.Errorf("want 0 findings, got %d", len(body.Findings))
	}
}
