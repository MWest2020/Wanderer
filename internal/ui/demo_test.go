package ui_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/internal/ui"
	"github.com/MWest2020/wanderer/pkg/models"
)

func newDemoServer(t *testing.T, st *store.Store, target string) *httptest.Server {
	t.Helper()
	tmpl, err := ui.Templates()
	if err != nil {
		t.Fatalf("templates: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/demo", ui.DemoHandler(st, tmpl, target))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// seedCompleteScan creates a target + a finished scan (status
// complete) with one finding and one wand assessment carrying a
// sovereignty-flow rationale, so BuildAnswerVerdict has something to
// say. Mirrors seedAssessment/seed above but finishes the scan, since
// the demo route only ever shows a "voltooide scan".
func seedCompleteScan(t *testing.T, st *store.Store, domain string) (targetID, scanID string) {
	t.Helper()
	ctx := context.Background()
	tgt := &models.Target{Domain: domain}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	findingID := "f_test_" + domain
	if err := st.AppendFindings(ctx, sc.ID, []models.Finding{
		{ID: findingID, ProbeID: "tls.issuer", Subject: domain, Severity: models.SeverityFinding},
	}); err != nil {
		t.Fatalf("findings: %v", err)
	}
	if err := st.FinishScan(ctx, sc.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish: %v", err)
	}
	a := &models.Assessment{
		ScanID:    sc.ID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreSoeverein,
			Completeness: models.CompletenessComplete,
			Rationale: []models.Rationale{{
				CriteriumID: "wand.juridisch.cert_issuer_eea",
				Verdict:     "cert issued in NL (EEA)",
				Score:       models.ScoreSoeverein,
				Evidence:    []string{findingID},
			}},
		}},
	}
	if err := st.CreateAssessment(ctx, a); err != nil {
		t.Fatalf("assessment: %v", err)
	}
	return tgt.ID, sc.ID
}

func TestDemoHandler_NoScanYetSaysSo(t *testing.T) {
	st := newTestStore(t)
	srv := newDemoServer(t, st, "westerweel.work")
	resp, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Nog geen meting") {
		t.Errorf("expected 'Nog geen meting' hint; body:\n%s", string(body))
	}
	if strings.Contains(string(body), "Ja —") || strings.Contains(string(body), "Nee —") {
		t.Errorf("expected no verdict headline without a scan; body:\n%s", string(body))
	}
}

func TestDemoHandler_CompletedScanShowsVerdictAndDate(t *testing.T) {
	st := newTestStore(t)
	seedCompleteScan(t, st, "westerweel.work")
	srv := newDemoServer(t, st, "westerweel.work")
	resp, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Ja —") {
		t.Errorf("expected the ja/nee/onbekend headline; body:\n%s", string(body))
	}
	if !strings.Contains(string(body), "Gescand op") {
		t.Errorf("expected the scan date; body:\n%s", string(body))
	}
	if !strings.Contains(string(body), "het certificaat is uitgegeven binnen de EER") {
		t.Errorf("expected the Dutch flow onderbouwing; body:\n%s", string(body))
	}
}

func TestDemoHandler_NeverMentionsAnotherTarget(t *testing.T) {
	st := newTestStore(t)
	seedCompleteScan(t, st, "westerweel.work")
	seedCompleteScan(t, st, "some-other-customer.example")
	srv := newDemoServer(t, st, "westerweel.work")
	resp, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "some-other-customer.example") {
		t.Errorf("demo page leaked another target's name; body:\n%s", string(body))
	}
}

func TestDemoHandler_PostDoesNotStartAScan(t *testing.T) {
	// DemoHandler is never given a Scanner, so a POST can only render
	// the same read-only view — there is no code path here that could
	// enqueue work (tasks.md 1.4).
	st := newTestStore(t)
	srv := newDemoServer(t, st, "westerweel.work")
	resp, err := http.Post(srv.URL+"/demo", "application/x-www-form-urlencoded", strings.NewReader("domain=westerweel.work"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	scans, err := st.ListScans(context.Background(), store.Selectors{})
	if err != nil {
		t.Fatalf("list scans: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("expected no scans to exist, got %d", len(scans))
	}
}
