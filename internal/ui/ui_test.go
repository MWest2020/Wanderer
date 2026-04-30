package ui_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/internal/ui"
	"github.com/MWest2020/wanderer/pkg/models"

	"golang.org/x/crypto/bcrypt"
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

func seed(t *testing.T, st *store.Store) (targetID, scanID string) {
	t.Helper()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if err := st.AppendFindings(context.Background(), sc.ID, []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Severity: models.SeverityFinding, Attributes: map[string]any{"issuer_country": []string{"NL"}}},
	}); err != nil {
		t.Fatalf("findings: %v", err)
	}
	return tgt.ID, sc.ID
}

func newServer(t *testing.T, htpasswd string) (*httptest.Server, *store.Store) {
	t.Helper()
	st := newTestStore(t)
	h, err := ui.Handler(st, htpasswd)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, st
}

func TestIndex_RendersTargetRow(t *testing.T) {
	srv, st := newServer(t, "")
	_, _ = seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "example.nl") {
		t.Errorf("index missing target domain; body:\n%s", string(body))
	}
	if !strings.Contains(string(body), "Wanderer targets") {
		t.Errorf("index header missing")
	}
}

func TestScanPage_RendersFindings(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "tls.issuer") {
		t.Errorf("scan page missing finding ProbeID; body:\n%s", string(body))
	}
}

func TestScanPage_NotFound(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/scans/s_missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// seedAssessment persists a DICTU and (optionally) EUCSF assessment
// for the given scan. The assessments cover one populated dimension
// each so the assessment-page tests can assert per-rule rendering.
func seedAssessment(t *testing.T, st *store.Store, scanID string, frameworks ...string) {
	t.Helper()
	if len(frameworks) == 0 {
		frameworks = []string{"dictu"}
	}
	for _, fw := range frameworks {
		var rationale models.Rationale
		switch fw {
		case "dictu":
			rationale = models.Rationale{
				CriteriumID: "dictu.juridisch.cert_issuer_eea",
				Verdict:     "cert issued in NL (EEA)",
				Score:       models.ScoreSoeverein,
				Evidence:    []string{"f_test"},
			}
		case "eucsf":
			rationale = models.Rationale{
				CriteriumID: "eucsf.sov2.cert_issuer_eu",
				Verdict:     "cert issued in NL",
				Score:       models.ScoreSoeverein,
				Evidence:    []string{"f_test"},
			}
		}
		a := &models.Assessment{
			ScanID:    scanID,
			Framework: fw,
			Dimensions: []models.DimensionScore{{
				Dimension:    models.DimensionJuridisch,
				Score:        models.ScoreSoeverein,
				Completeness: models.CompletenessComplete,
				Rationale:    []models.Rationale{rationale},
			}},
		}
		if err := st.CreateAssessment(context.Background(), a); err != nil {
			t.Fatalf("create assessment %s: %v", fw, err)
		}
	}
}

func TestAssessmentPage_RendersDimensionAndRule(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "dictu", "eucsf")
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/assessment")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		"Assessment for scan",
		"dictu",
		"eucsf",
		"juridisch",
		"dictu.juridisch.cert_issuer_eea",
		"eucsf.sov2.cert_issuer_eu",
		"score-soeverein",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("assessment page missing %q; body:\n%s", want, bodyStr)
		}
	}
}

func TestAssessmentPage_EmptyShowsHint(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/assessment")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "No assessment has been produced") {
		t.Errorf("expected empty-state hint; body:\n%s", string(body))
	}
	if !strings.Contains(string(body), "wanderer assess") {
		t.Errorf("expected hint to mention `wanderer assess`")
	}
}

func TestAssessmentPage_RetiredRuleDegrades(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "dictu",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreOnbekend,
			Completeness: models.CompletenessIncomplete,
			Rationale: []models.Rationale{{
				CriteriumID: "dictu.juridisch.no_such_rule_anymore",
				Verdict:     "historical verdict",
				Score:       models.ScoreOnbekend,
			}},
		}},
	}
	if err := st.CreateAssessment(context.Background(), a); err != nil {
		t.Fatalf("create assessment: %v", err)
	}
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID + "/assessment")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "rule retired") {
		t.Errorf("expected 'rule retired' marker for unknown CriteriumID; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "historical verdict") {
		t.Errorf("expected historical verdict to remain visible; body:\n%s", bodyStr)
	}
}

func TestScanPage_LinksToAssessmentWhenAssessed(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "dictu")
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Open assessment") {
		t.Errorf("scan page should link to assessment; body:\n%s", string(body))
	}
}

func TestScanPage_NoAssessmentLinkWhenAbsent(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/scans/" + scanID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "Open assessment") {
		t.Errorf("scan page should NOT link to assessment when none exists; body:\n%s", string(body))
	}
}

func TestDriftPage_RendersEmptyState(t *testing.T) {
	srv, st := newServer(t, "")
	tgtID, _ := seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/targets/" + tgtID + "/drift")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "No drift findings") {
		t.Errorf("drift page missing empty-state; body:\n%s", string(body))
	}
}

func TestBasicAuth_RejectsAndAccepts(t *testing.T) {
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	srv, st := newServer(t, htpasswd)
	_, _ = seed(t, st)

	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("unauthenticated: status = %d, want 401", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); got == "" {
		t.Errorf("missing WWW-Authenticate header")
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/ui/", nil)
	req.SetBasicAuth("op", "correct horse battery staple")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("auth do: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("authed: status = %d, want 200", resp2.StatusCode)
	}

	req3, _ := http.NewRequest(http.MethodGet, srv.URL+"/ui/", nil)
	req3.SetBasicAuth("op", "wrong")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("bad-pw do: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != 401 {
		t.Errorf("wrong password: status = %d, want 401", resp3.StatusCode)
	}
}

func TestLoadHtpasswd_RejectsMD5(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "md5.htpasswd")
	if err := os.WriteFile(path, []byte("admin:$apr1$abc$xyz\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ui.LoadHtpasswd(path); err == nil {
		t.Errorf("expected error on MD5 entry")
	} else if !strings.Contains(err.Error(), "MD5") {
		t.Errorf("error should name MD5; got %v", err)
	}
}

func TestNoMutatingHandlersInPackage(t *testing.T) {
	// Static-analysis check per goal #7 spec: the ui package
	// declares no chi.Router or http.ServeMux registration with
	// methods other than GET. We grep our own source for
	// `r.Post|r.Put|r.Patch|r.Delete` patterns.
	src, err := os.ReadFile("ui.go")
	if err != nil {
		// The test runs from internal/ui so the relative path
		// works; fall back to absolute via the module root if
		// someone runs go test from elsewhere.
		src, err = os.ReadFile(filepath.Join("internal", "ui", "ui.go"))
		if err != nil {
			t.Fatalf("read ui.go: %v", err)
		}
	}
	for _, banned := range []string{"r.Post(", "r.Put(", "r.Patch(", "r.Delete("} {
		if strings.Contains(string(src), banned) {
			t.Errorf("ui package contains mutating handler registration: %s", banned)
		}
	}
}
