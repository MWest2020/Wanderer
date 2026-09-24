package ui_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/scheduler"
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
	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, st
}

func TestTargetsRoute_RendersTargetRow(t *testing.T) {
	srv, st := newServer(t, "")
	_, _ = seed(t, st)
	resp, err := http.Get(srv.URL + "/ui/targets")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "example.nl") {
		t.Errorf("targets page missing target domain; body:\n%s", string(body))
	}
	if !strings.Contains(string(body), "Wanderer targets") {
		t.Errorf("targets header missing")
	}
}

// TestNav_IsOneLanguage covers run 05 task 4.5: the nav bar mixed
// "Overview · Vloot · Trends" — an English tab next to two Dutch ones.
func TestNav_IsOneLanguage(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `>Overzicht</a>`) {
		t.Errorf("expected the Dutch nav label \"Overzicht\"; body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, `>Overview</a>`) {
		t.Errorf("nav must not read \"Overview\" next to the Dutch tabs; body:\n%s", bodyStr)
	}
}

func TestDashboard_EmptyStoreRendersEmptyHint(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "wanderer scan") {
		t.Errorf("expected empty-state hint with `wanderer scan`; body:\n%s", string(body))
	}
}

func TestTrends_VerdictPill_PerFramework(t *testing.T) {
	// After the answer-first restructure, the fleet table and the
	// per-framework verdict pills (worst-score) live on /ui/trends —
	// no posture distribution blocks, no Top concerns table, no
	// Recent activity table.
	srv, st := newServer(t, "")
	for i, score := range []models.Score{models.ScoreSoeverein, models.ScoreAfhankelijk, models.ScoreOnbekend} {
		domain := []string{"a.example", "b.example", "c.example"}[i]
		tgt := &models.Target{Domain: domain}
		if err := st.UpsertTarget(context.Background(), tgt); err != nil {
			t.Fatal(err)
		}
		sc, err := st.CreateScan(context.Background(), tgt.ID)
		if err != nil {
			t.Fatal(err)
		}
		comp := models.CompletenessComplete
		if score == models.ScoreOnbekend {
			comp = models.CompletenessIncomplete
		}
		a := &models.Assessment{
			ScanID:    sc.ID,
			Framework: "wand",
			Dimensions: []models.DimensionScore{{
				Dimension:    models.DimensionJuridisch,
				Score:        score,
				Completeness: comp,
			}},
		}
		if err := st.CreateAssessment(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := http.Get(srv.URL + "/ui/trends")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		"Verdict",
		"verdict-pill",
		"score-afhankelijk", // worst across {soeverein, afhankelijk, onbekend}
		// Fleet table — the per-target Tourist view: every target,
		// its verdict, and a link to that scan's report.
		"targets-fleet",
		"a.example",
		"b.example",
		"c.example",
		"report →",
		"/assessment",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("trends page missing %q", want)
		}
	}
	for _, mustNotHave := range []string{
		"External posture",
		"Internal posture",
		"Top concerns",
		"Recent activity",
	} {
		if strings.Contains(bodyStr, mustNotHave) {
			t.Errorf("dashboard MUST NOT contain %q after the 2026-05-10 layer restructure", mustNotHave)
		}
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
		frameworks = []string{"wand"}
	}
	for _, fw := range frameworks {
		var rationale models.Rationale
		switch fw {
		case "wand":
			rationale = models.Rationale{
				CriteriumID: "wand.juridisch.cert_issuer_eea",
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
	seedAssessment(t, st, scanID, "wand", "eucsf")
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
		"example.nl", // domain is the heading, not the opaque scan ID
		"wand",
		"eucsf",
		"juridisch",
		"wand.juridisch.cert_issuer_eea",
		"eucsf.sov2.cert_issuer_eu",
		"score-soeverein",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("assessment page missing %q; body:\n%s", want, bodyStr)
		}
	}
}

// TestAssessmentPage_FlowsRenderAsAnswerSheetWithRuleIDsInEvidenceOnly
// covers run 04 tasks 4.1/4.2: the seven sovereignty flows render as
// answer-sheet questions (reusing the accountability pattern), a flow
// rule's ID no longer appears in the generic per-dimension table (it
// would be a second, open-air view of the same fact), and a non-flow
// rule in the same dimension is untouched.
func TestAssessmentPage_FlowsRenderAsAnswerSheetWithRuleIDsInEvidenceOnly(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreSoeverein,
			Completeness: models.CompletenessComplete,
			Rationale: []models.Rationale{
				{CriteriumID: "wand.juridisch.cert_issuer_eea", Verdict: "cert issued in NL (EEA)", Score: models.ScoreSoeverein},
				{CriteriumID: "wand.juridisch.registrar_jurisdiction", Verdict: "registrant in NL (EEA)", Score: models.ScoreSoeverein},
			},
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
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Waar is het certificaat uitgegeven?") {
		t.Errorf("expected the Certificate flow's plain-language question; body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, "<code>wand.juridisch.cert_issuer_eea</code>\n            <details class=\"why\">") {
		t.Errorf("cert_issuer_eea must not also render in the generic dimension table; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "wand.juridisch.cert_issuer_eea") {
		t.Errorf("cert_issuer_eea should still appear, collapsed inside the flow's evidence; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "wand.juridisch.registrar_jurisdiction") {
		t.Errorf("a non-flow rule in the same dimension must still render in the generic table; body:\n%s", bodyStr)
	}
	// One link back to the answer page (spec.md "one click from the
	// answer"; run 04 task 4.3).
	if !strings.Contains(bodyStr, `href="/ui/scans/`+scanID+`/answer"`) {
		t.Errorf("assessment page must link back to the answer page; body:\n%s", bodyStr)
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
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreOnbekend,
			Completeness: models.CompletenessIncomplete,
			Rationale: []models.Rationale{{
				CriteriumID: "wand.juridisch.no_such_rule_anymore",
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

// TestAssessmentPage_StandardsReportURLIsClickable covers run 04 task
// 4.1: the Internet.nl report URL a standards rule's Verdict carries
// (design.md "Design gate outcome" #5 — opaque, self-hosted, never
// built by Wanderer) must render as a real link on the existing rule
// table, not as inert text.
func TestAssessmentPage_StandardsReportURLIsClickable(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	reportURL := "https://netnl.westerweel.work/site/westerweel.work/485/"
	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionStandards,
			Score:        models.ScoreSoeverein,
			Completeness: models.CompletenessComplete,
			Rationale: []models.Rationale{{
				CriteriumID: "wand.standards.dnssec",
				Verdict:     "all 6 tested DNSSEC test(s) passed — see " + reportURL,
				Score:       models.ScoreSoeverein,
				Evidence:    []string{"finding-1"},
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
	want := `<a href="` + reportURL + `" rel="noopener noreferrer" target="_blank">` + reportURL + `</a>`
	if !strings.Contains(bodyStr, want) {
		t.Errorf("expected a clickable report link %q; body:\n%s", want, bodyStr)
	}
}

// TestAssessmentPage_StandardsWithoutImportShowsNotMeasured covers run
// 04 task 4.1: a target with no `wanderer import internetnl` run at
// all must show "not measured" text with the onbekend badge on each
// standards rule — never an empty row and never a score that reads as
// good.
func TestAssessmentPage_StandardsWithoutImportShowsNotMeasured(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionStandards,
			Score:        models.ScoreOnbekend,
			Completeness: models.CompletenessIncomplete,
			Rationale: []models.Rationale{{
				CriteriumID: "wand.standards.dnssec",
				Verdict:     "no internetnl.* findings imported for DNSSEC — not measured",
				Score:       models.ScoreOnbekend,
				Reason:      "not_measured",
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
	if !strings.Contains(bodyStr, "not measured") {
		t.Errorf("expected \"not measured\" text for an unimported standards rule; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, `score-onbekend">onbekend</span>`) {
		t.Errorf("expected the onbekend badge on the not-measured rule, not an empty or good-looking box; body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, `score-soeverein`) {
		t.Errorf("a target with no import must never render a soeverein-looking standards badge; body:\n%s", bodyStr)
	}
}

// TestAssessmentPage_FrameworkTablesCollapseBehindOneClick covers run 05
// task 5.1: the onderbouwing page opens with the seven flow questions,
// and the two raw framework tables underneath sit behind a single
// <details> whose <summary> names what's underneath and how many rows
// — not a bare "details".
func TestAssessmentPage_FrameworkTablesCollapseBehindOneClick(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand", "eucsf")
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

	iFlows := strings.Index(bodyStr, `class="sovereignty-overview"`)
	iDetails := strings.Index(bodyStr, `<details class="framework-details">`)
	iFrameworks := strings.Index(bodyStr, `<div class="frameworks">`)
	iDetailsClose := strings.LastIndex(bodyStr, "</details>")
	if iFlows == -1 || iDetails == -1 || iFrameworks == -1 || iDetailsClose == -1 {
		t.Fatalf("expected the seven-questions section followed by a collapsed framework-details block; body:\n%s", bodyStr)
	}
	if !(iFlows < iDetails && iDetails < iFrameworks && iFrameworks < iDetailsClose) {
		t.Errorf("expected order: sovereignty-overview, then <details>, then the frameworks div inside it; body:\n%s", bodyStr)
	}

	summaryRe := regexp.MustCompile(`<summary>[^<]*\d+[^<]*regel[^<]*</summary>`)
	if !summaryRe.MatchString(bodyStr) {
		t.Errorf("expected a <summary> naming what's underneath and how many regels, not a bare \"details\"; body:\n%s", bodyStr)
	}
}

func TestScanPage_LinksToAssessmentWhenAssessed(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand")
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
	// PUT/PATCH/DELETE are never allowed. POST is allowed only for the
	// sanctioned, signed-in-only routes below. Any other POST (or any
	// of the others) is a regression.
	for _, banned := range []string{"r.Put(", "r.Patch(", "r.Delete("} {
		if strings.Contains(string(src), banned) {
			t.Errorf("ui package contains mutating handler registration: %s", banned)
		}
	}
	// Sanctioned mutating routes, each mounted only for a signed-in
	// user (never a dev-mode flag alone):
	//   /scan                                   — dev-mode scan trigger
	//   /orgs/{slug}/fleet/domains               — add a domain to the
	//     org's fleet without scanning (vloot-en-regels run 02)
	//   /orgs/{slug}/fleet/domains/{domain}/remove — take a domain out
	//     of the fleet; scans/oordelen stay queryable, only the
	//     overview listing changes (same run)
	sanctioned := map[string]bool{
		"/scan":                      true,
		"/orgs/{slug}/fleet/domains": true,
		"/orgs/{slug}/fleet/domains/{domain}/remove": true,
	}
	posts := regexp.MustCompile(`r\.Post\("([^"]*)"`).FindAllStringSubmatch(string(src), -1)
	for _, m := range posts {
		if !sanctioned[m[1]] {
			t.Errorf("unexpected mutating POST route %q — not in the sanctioned set", m[1])
		}
	}
}

func TestTrends_ConsolidatesCatalogueAndMatrix(t *testing.T) {
	// After the Tourist/Explorer/Farmer restructure, the rule
	// catalogue and the rule × score matrix live together on
	// /ui/trends — the single Farmer surface.
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand", "eucsf")
	resp, err := http.Get(srv.URL + "/ui/trends")
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
		"Trends · rules across your fleet",
		"Rule catalogue",                 // catalogue section
		"Score matrix",                   // matrix section
		"wand.juridisch.cert_issuer_eea", // rules from both packs
		"eucsf.sov2.cert_issuer_eu",
		"soeverein", // the matrix carries score data
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("trends page missing %q", want)
		}
	}
}

func TestLegacyAnalysisReporting_RedirectToTrends(t *testing.T) {
	// /ui/analysis and /ui/reporting consolidated into /ui/trends;
	// the old routes redirect, preserving the org scope.
	srv, _ := newServer(t, "")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	cases := map[string]string{
		"/ui/analysis":           "/ui/trends",
		"/ui/reporting":          "/ui/trends",
		"/ui/analysis?org=acme":  "/ui/trends?org=acme",
		"/ui/reporting?org=acme": "/ui/trends?org=acme",
	}
	for from, wantLoc := range cases {
		resp, err := client.Get(srv.URL + from)
		if err != nil {
			t.Fatalf("get %s: %v", from, err)
		}
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Errorf("%s: status = %d, want 302", from, resp.StatusCode)
		}
		if loc != wantLoc {
			t.Errorf("%s: Location = %q, want %q", from, loc, wantLoc)
		}
	}
}

func TestReporting_Rule_RendersTargets(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand")
	resp, err := http.Get(srv.URL + "/ui/reporting/wand/wand.juridisch.cert_issuer_eea")
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
		"wand.juridisch.cert_issuer_eea",
		"cert issued in NL (EEA)", // verdict text from seedAssessment
		"score-soeverein",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("reporting rule page missing %q", want)
		}
	}
}

func TestReporting_Rule_UnknownReturns404(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/reporting/wand/wand.juridisch.does_not_exist")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestDashboard_Org_PerOrgPageRebadgesHeadline(t *testing.T) {
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	tgt := &models.Target{Domain: "a.example", OrganisationID: o.ID}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/orgs/acme")
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
		"ACME B.V.",         // headline rebadged
		"all organisations", // back-link to instance-wide
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("per-org page missing %q", want)
		}
	}
}

func TestDashboard_Org_UnknownReturns404(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/orgs/nope")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestNav_PerOrgPageThreadsScopeIntoNavLinks(t *testing.T) {
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/orgs/acme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		// The two-tab nav (Overview / Trends) threads the slug across
		// both tabs; Trends is scope-aware (its status column counts
		// distinct targets in scope).
		`href="/ui/trends?org=acme"`,
		`href="/ui/orgs/acme"`,
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("per-org page nav missing %q", want)
		}
	}
	// The retired tabs must be gone from the nav.
	for _, gone := range []string{`href="/ui/analysis`, `href="/ui/reporting?`} {
		if strings.Contains(bodyStr, gone) {
			t.Errorf("per-org nav still references a retired tab %q", gone)
		}
	}
}

func TestNav_PagesIncludeTrendsTab(t *testing.T) {
	// Every page's nav offers the Trends tab (HasReporting=true).
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	for _, path := range []string{"/ui/targets", "/ui/scans/" + scanID, "/ui/scans/" + scanID + "/assessment"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), `href="/ui/trends`) {
			t.Errorf("%s: Trends nav link missing", path)
		}
		// The old DAR tabs are gone.
		if strings.Contains(string(body), `>Analysis<`) || strings.Contains(string(body), `>Reporting<`) {
			t.Errorf("%s: nav still shows a retired Analysis/Reporting tab", path)
		}
	}
}

func TestAnalysis_ShowsScopeLabelWhenFiltered(t *testing.T) {
	// The scope label moved with the matrix from /ui/reporting to
	// /ui/analysis (2026-05-10 layer restructure).
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/analysis?org=acme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		`Scope:`,
		`ACME B.V.`,
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("analysis page missing scope label %q", want)
		}
	}
}

func TestTargets_Filtered_ByOrg(t *testing.T) {
	srv, st := newServer(t, "")
	acme := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), acme); err != nil {
		t.Fatal(err)
	}
	tgtAcme := &models.Target{Domain: "a.example", OrganisationID: acme.ID}
	if err := st.UpsertTarget(context.Background(), tgtAcme); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateScan(context.Background(), tgtAcme.ID); err != nil {
		t.Fatal(err)
	}
	tgtOther := &models.Target{Domain: "b.example"} // default org
	if err := st.UpsertTarget(context.Background(), tgtOther); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateScan(context.Background(), tgtOther.ID); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/targets?org=acme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "a.example") {
		t.Errorf("acme target a.example not in filtered list")
	}
	if strings.Contains(bodyStr, "b.example") {
		t.Errorf("default-org target b.example leaked into acme view")
	}
}

func TestTrends_Global_ListsOrganisationsWhenMultiple(t *testing.T) {
	srv, st := newServer(t, "")
	for _, slug := range []string{"acme", "beta"} {
		o := &models.Organisation{Slug: slug, Name: slug}
		if err := st.UpsertOrganisation(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}
	tgt := &models.Target{Domain: "a.example"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateScan(context.Background(), tgt.ID); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/trends")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		"<h2>Organisations</h2>",
		"/ui/orgs/acme",
		"/ui/orgs/beta",
		"all organisations",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("trends page missing %q", want)
		}
	}
}

// stubScanner persists a minimal scan so the scan-trigger handler can
// assess it and redirect. Satisfies ui.ScanTrigger.
type stubScanner struct{ st *store.Store }

func (s stubScanner) Scan(ctx context.Context, target models.Target) (*models.Scan, error) {
	if err := s.st.UpsertTarget(ctx, &target); err != nil {
		return nil, err
	}
	sc, err := s.st.CreateScan(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	_ = s.st.FinishScan(ctx, sc.ID, models.ScanStatusComplete, "")
	return sc, nil
}

// basicAuthClient returns an http.Client that attaches Basic auth
// credentials to every request — the break-glass "signed-in user" for
// tests that don't drive the full OIDC dance.
func basicAuthClient(user, pass string) *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.SetBasicAuth(user, pass)
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUIScan_SignedInUser_TriggersAndRedirects(t *testing.T) {
	// spec.md "The entry surface asks for a domain and answers it":
	// scanning requires a signed-in user — htpasswd Basic auth here,
	// OIDC in production.
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := newTestStore(t)
	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd, Scanner: stubScanner{st: st}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := basicAuthClient("op", "correct horse battery staple")
	resp, err := client.PostForm(srv.URL+"/ui/scan", url.Values{"domain": {"example.nl"}})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", resp.StatusCode)
	}
	// The scan runs in the background; the POST bounces to a
	// domain-keyed status page that polls until the result lands.
	loc := resp.Header.Get("Location")
	if loc != "/ui/scan-status?domain=example.nl" {
		t.Fatalf("Location = %q, want /ui/scan-status?domain=example.nl", loc)
	}
	// Poll the status page; once the background scan has a row it
	// redirects to that scan's answer page — it does not wait for the
	// (also background) assessment to land.
	var answerLoc string
	for i := 0; i < 50; i++ {
		sr, serr := client.Get(srv.URL + loc)
		if serr != nil {
			t.Fatalf("get status: %v", serr)
		}
		l := sr.Header.Get("Location")
		sr.Body.Close()
		if sr.StatusCode == http.StatusSeeOther && strings.HasPrefix(l, "/ui/scans/") && strings.HasSuffix(l, "/answer") {
			answerLoc = l
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if answerLoc == "" {
		t.Errorf("status page never redirected to an answer page")
	}
}

func TestUIScan_ReadOnlyByDefault_NoScanRoute(t *testing.T) {
	// Without Options.Scanner the /ui/scan route is not mounted.
	srv, _ := newServer(t, "")
	resp, err := http.Post(srv.URL+"/ui/scan", "application/x-www-form-urlencoded", strings.NewReader("domain=example.nl"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther {
		t.Errorf("read-only UI must not trigger scans (got 303)")
	}
}

func TestDoor_ShowsFleetScoreAndAgentHost(t *testing.T) {
	// openspec change 2026-09-22-drie-lagen-ciso, spec.md "De vloot is
	// de eerste laag": the door opens with the fleet's x/n score and
	// its domains, worst first, each linking to its own answer page —
	// and still offers the scan input, with an enrolled agent host
	// selectable from the same input (the original entry-surface
	// requirement), as an action inside the page rather than its
	// headline.
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := newTestStore(t)
	if _, _, err := st.EnrolAgent(context.Background(), mustEnrolmentToken(t, st), "webapp-01"); err != nil {
		t.Fatalf("enrol: %v", err)
	}
	_, scanID := seed(t, st)
	if err := st.FinishScan(context.Background(), scanID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	seedAssessment(t, st, scanID, "wand")

	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd, Scanner: stubScanner{st: st}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := basicAuthClient("op", "correct horse battery staple")
	resp, err := client.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{
		"1/1", // the fleet score: the one soeverein rationale seedAssessment creates, out of one answerable
		"example.nl",
		scanID + "/answer", // the domain links straight to its answer page
		"webapp-01",        // the agent host, still selectable from the datalist
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("door missing %q; body:\n%s", want, bodyStr)
		}
	}
}

// seedDoorDomain adds domain to org's fleet with a completed scan and a
// single-flow assessment — the door page (buildDoorDomains) only counts
// a domain toward the vlootscore once its latest scan is
// Complete/Partial, unlike the fleet page which scores whatever scan it
// finds regardless of status.
func seedDoorDomain(t *testing.T, st *store.Store, orgID, domain string, rs ...models.Rationale) {
	t.Helper()
	tgt, err := st.AddFleetDomain(context.Background(), orgID, domain)
	if err != nil {
		t.Fatalf("AddFleetDomain(%s): %v", domain, err)
	}
	sc, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	if err := st.FinishScan(context.Background(), sc.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	a := &models.Assessment{
		ScanID:    sc.ID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreSoeverein,
			Completeness: models.CompletenessComplete,
			Rationale:    rs,
		}},
	}
	if err := st.CreateAssessment(context.Background(), a); err != nil {
		t.Fatalf("create assessment: %v", err)
	}
}

// TestDoor_CISOSeesOneScoreAndWorstDomainFirst pins spec.md's scenario
// "CISO opent de tool": a organisation with five scanned domains shows
// one score for the fleet, how many domains are not sovereign, and the
// worst domain on top.
func TestDoor_CISOSeesOneScoreAndWorstDomainFirst(t *testing.T) {
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		seedDoorDomain(t, st, o.ID, "acme-"+string(rune('0'+i))+".nl",
			models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein})
	}
	seedDoorDomain(t, st, o.ID, "worst.nl",
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in US", Score: models.ScoreAfhankelijk})

	resp, err := http.Get(srv.URL + "/ui/orgs/acme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "4/5") {
		t.Errorf("expected the fleet-wide 4/5 score; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "1 domein niet soeverein") {
		t.Errorf("expected '1 domein niet soeverein'; body:\n%s", bodyStr)
	}
	if i, j := strings.Index(bodyStr, "worst.nl"), strings.Index(bodyStr, "acme-1.nl"); i == -1 || j == -1 || i > j {
		t.Errorf("worst.nl (index %d) must render before acme-1.nl (index %d); body:\n%s", i, j, bodyStr)
	}
}

// TestDoor_GoodScoreNeverHidesFailingDomain pins spec.md's scenario
// "Een goede score verbergt geen slecht domein": nine sovereign domains
// and one failing domain must still surface the one not-sovereign count
// and put the failing domain on top.
func TestDoor_GoodScoreNeverHidesFailingDomain(t *testing.T) {
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 9; i++ {
		seedDoorDomain(t, st, o.ID, "goed-"+string(rune('0'+i))+".nl",
			models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein})
	}
	seedDoorDomain(t, st, o.ID, "erger.nl",
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in US", Score: models.ScoreAfhankelijk})

	resp, err := http.Get(srv.URL + "/ui/orgs/acme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "1 domein niet soeverein") {
		t.Errorf("a good fleet score must still name the one failing domain; body:\n%s", bodyStr)
	}
	if i, j := strings.Index(bodyStr, "erger.nl"), strings.Index(bodyStr, "goed-1.nl"); i == -1 || j == -1 || i > j {
		t.Errorf("erger.nl (index %d) must render before goed-1.nl (index %d) — worst first; body:\n%s", i, j, bodyStr)
	}
}

// TestDoor_ScanFieldPrecedesFleetScore pins spec.md's scenario "Het
// invoerveld staat bovenaan": a signed-in user's scan input SHALL
// render before the vlootscore in the page.
func TestDoor_ScanFieldPrecedesFleetScore(t *testing.T) {
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := newTestStore(t)
	_, scanID := seed(t, st)
	if err := st.FinishScan(context.Background(), scanID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	seedAssessment(t, st, scanID, "wand")

	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd, Scanner: stubScanner{st: st}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := basicAuthClient("op", "correct horse battery staple")
	resp, err := client.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	scanIdx := strings.Index(bodyStr, `class="scan-form"`)
	fleetIdx := strings.Index(bodyStr, `class="dashboard-section fleet-summary"`)
	if scanIdx == -1 || fleetIdx == -1 || scanIdx > fleetIdx {
		t.Errorf("scan-form (index %d) must precede the fleet-summary (index %d); body:\n%s", scanIdx, fleetIdx, bodyStr)
	}
}

// TestDoor_OrganisationsTableHiddenWithOneOrganisation pins task 1.6:
// the Organisations table leaves the Tourist layer when there is only
// one organisation (the seeded "default" one) — nothing to pick
// between, so it would be pure noise.
func TestDoor_OrganisationsTableHiddenWithOneOrganisation(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "<h2>Organisations</h2>") {
		t.Errorf("Organisations table must not render with a single organisation; body:\n%s", string(body))
	}
}

// TestDoor_OrganisationsTableShownWithMultipleOrganisations is the
// complement: once a second organisation exists, the door still needs a
// way to reach it.
func TestDoor_OrganisationsTableShownWithMultipleOrganisations(t *testing.T) {
	srv, st := newServer(t, "")
	o := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<h2>Organisations</h2>") {
		t.Errorf("Organisations table must render with more than one organisation; body:\n%s", string(body))
	}
}

// mustEnrolmentToken issues a fresh enrolment token for the enrol
// helper tests; the plain token is the only return value they need.
func mustEnrolmentToken(t *testing.T, st *store.Store) string {
	t.Helper()
	plain, _, err := st.CreateEnrolmentToken(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("enrolment token: %v", err)
	}
	return plain
}

func TestUIScan_RefusedWithoutAuth_EvenWithScanner(t *testing.T) {
	// spec.md "No authentication configured": a Scanner alone does not
	// open the route — an instance with no htpasswd and no OIDC has no
	// way to gate "signed in", so the route must not mount.
	st := newTestStore(t)
	h, err := ui.Handler(st, ui.Options{Scanner: stubScanner{st: st}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/ui/scan", "application/x-www-form-urlencoded", strings.NewReader("domain=example.nl"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther {
		t.Errorf("scan route must be refused with no authentication configured (got 303)")
	}
}

func TestFleetPage_ShowsUnscannedDomainAsNotScanned(t *testing.T) {
	// spec.md scenario "Domein toevoegen zonder te scannen": a fleet
	// domain with no scan yet renders "nog niet gescand", not a blank
	// or an error.
	srv, st := newServer(t, "")
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "voorbeeld.nl") {
		t.Errorf("fleet page missing the domain; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "nog niet gescand") {
		t.Errorf("fleet page missing \"nog niet gescand\" for an unscanned domain; body:\n%s", bodyStr)
	}
}

func TestFleetPage_UnknownOrgReturns404(t *testing.T) {
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/orgs/nope/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestFleetAdd_SignedInUser_AddsDomainAndRedirects(t *testing.T) {
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := newTestStore(t)
	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := basicAuthClient("op", "correct horse battery staple")
	resp, err := client.PostForm(srv.URL+"/ui/orgs/default/fleet/domains", url.Values{"domain": {"voorbeeld.nl"}})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/ui/orgs/default/fleet" {
		t.Errorf("Location = %q, want /ui/orgs/default/fleet", loc)
	}
	list, err := st.ListFleetDomains(context.Background(), models.DefaultOrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Domain != "voorbeeld.nl" {
		t.Errorf("ListFleetDomains = %v, want [voorbeeld.nl]", list)
	}
}

func TestFleetAdd_RefusedWithoutAuth(t *testing.T) {
	// Without htpasswd/OIDC configured, the mutating fleet routes must
	// not mount — same gate as /ui/scan.
	srv, st := newServer(t, "")
	resp, err := http.PostForm(srv.URL+"/ui/orgs/default/fleet/domains", url.Values{"domain": {"voorbeeld.nl"}})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther {
		t.Errorf("read-only UI must not accept fleet mutations (got 303)")
	}
	list, err := st.ListFleetDomains(context.Background(), models.DefaultOrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("domain must not have been added without auth: %v", list)
	}
}

func TestFleetRemove_SignedInUser_RemovesDomainButKeepsHistory(t *testing.T) {
	// spec.md scenario "Verwijderen laat de geschiedenis staan": the
	// domain disappears from the fleet listing but its scan is still
	// reachable.
	dir := t.TempDir()
	htpasswd := filepath.Join(dir, "passwd")
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err := os.WriteFile(htpasswd, []byte("op:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := newTestStore(t)
	_, scanID := seed(t, st) // creates the "example.nl" target + one scan, default org
	h, err := ui.Handler(st, ui.Options{HtpasswdPath: htpasswd})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := basicAuthClient("op", "correct horse battery staple")
	resp, err := client.PostForm(srv.URL+"/ui/orgs/default/fleet/domains/example.nl/remove", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", resp.StatusCode)
	}

	list, err := st.ListFleetDomains(context.Background(), models.DefaultOrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tg := range list {
		if tg.Domain == "example.nl" {
			t.Errorf("removed domain still in fleet listing")
		}
	}

	scanResp, err := client.Get(srv.URL + "/ui/scans/" + scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	defer scanResp.Body.Close()
	if scanResp.StatusCode != 200 {
		t.Errorf("scan history no longer reachable after removal: status = %d", scanResp.StatusCode)
	}
}

func TestFleetPage_ShowsSchedule(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	h, err := ui.Handler(st, ui.Options{Schedules: stubSchedules{{
		Name:   "nightly",
		Cron:   "0 3 * * *",
		Target: scheduler.Target{Domain: "voorbeeld.nl"},
	}}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "nightly") {
		t.Errorf("fleet page missing schedule name; body:\n%s", string(body))
	}
}

// seedFleetScan creates one scan + wand assessment for targetID with
// the given rationales, so fleet-screen tests can drive
// BuildFleetScore/BuildFleetDelta through the real HTTP handler
// instead of re-testing those functions' logic (already covered by
// fleet_score_test.go / fleet_delta_test.go).
func seedFleetScan(t *testing.T, st *store.Store, targetID string, rs ...models.Rationale) (scanID string, startedAt time.Time) {
	t.Helper()
	sc, err := st.CreateScan(context.Background(), targetID)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	a := &models.Assessment{
		ScanID:    sc.ID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{{
			Dimension:    models.DimensionJuridisch,
			Score:        models.ScoreSoeverein,
			Completeness: models.CompletenessComplete,
			Rationale:    rs,
		}},
	}
	if err := st.CreateAssessment(context.Background(), a); err != nil {
		t.Fatalf("create assessment: %v", err)
	}
	return sc.ID, sc.StartedAt
}

func TestFleetPage_ShowsScoreAndWorstFinding(t *testing.T) {
	// spec.md "Het vlootscherm scoort x van n, niet ja of nee": the row
	// shows x/n plus the heaviest open finding (BuildFleetScore).
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
		models.Rationale{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "mx hosts in US (outside EEA)", Score: models.ScoreAfhankelijk},
	)
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{"1/2", "Mail", "De mail wordt buiten de EER gerouteerd."} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("fleet page missing %q; body:\n%s", want, bodyStr)
		}
	}
	if strings.Contains(bodyStr, "mx hosts in US") {
		t.Errorf("fleet page leaks the rule's raw English verdict; body:\n%s", bodyStr)
	}
}

func TestFleetPage_ImportScanAfterPerimeterKeepsPerimeterScore(t *testing.T) {
	// specs/web-ui/spec.md "Import after a perimeter scan": an import
	// scan landing after a scored perimeter scan must not blank out
	// the fleet row — habitat run 02 found this rendering "0/0" (no
	// oordeel) on prod because the import scan, not the perimeter
	// scan, was picked as the domain's "latest" scan.
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	_, startedAt := seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
	)
	time.Sleep(5 * time.Millisecond)
	imp, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("create import scan: %v", err)
	}
	if err := st.AppendFindings(context.Background(), imp.ID, []models.Finding{
		{ProbeID: "internetnl.dnssec", Subject: "voorbeeld.nl", Severity: models.SeverityInfo, SourceModus: models.SourceModusImport, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append import finding: %v", err)
	}
	if err := st.FinishScan(context.Background(), imp.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish import scan: %v", err)
	}

	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "nog geen oordeel") {
		t.Errorf("fleet page shows no verdict after an import scan landed — the import scan must not replace the perimeter scan as \"latest\"; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "1/1") {
		t.Errorf("fleet page missing the perimeter scan's score \"1/1\"; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, startedAt.UTC().Format(time.RFC3339)) {
		t.Errorf("fleet page's last-scan time is not the perimeter scan's timestamp %s; body:\n%s", startedAt.UTC().Format(time.RFC3339), bodyStr)
	}
}

func TestFleetPage_OnlyImportScansShowsAsNotYetScanned(t *testing.T) {
	// specs/web-ui/spec.md "Only imports, no perimeter scan yet": a
	// fleet domain whose only scans are import scans must be listed
	// as not yet scanned, not with a blank/zero score.
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	imp, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("create import scan: %v", err)
	}
	if err := st.AppendFindings(context.Background(), imp.ID, []models.Finding{
		{ProbeID: "internetnl.dnssec", Subject: "voorbeeld.nl", Severity: models.SeverityInfo, SourceModus: models.SourceModusImport, Attributes: map[string]any{}},
	}); err != nil {
		t.Fatalf("append import finding: %v", err)
	}
	if err := st.FinishScan(context.Background(), imp.ID, models.ScanStatusComplete, ""); err != nil {
		t.Fatalf("finish import scan: %v", err)
	}

	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "voorbeeld.nl") {
		t.Fatalf("fleet page missing the domain; body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "nog niet gescand") {
		t.Errorf("fleet page must list an import-only domain as \"nog niet gescand\"; body:\n%s", bodyStr)
	}
}

func TestFleetPage_ShowsUnansweredCount(t *testing.T) {
	// spec.md scenario "Twee domeinen naast elkaar": an onbekend
	// question is reported separately, never folded into n.
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
		models.Rationale{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "probe failed", Score: models.ScoreOnbekend},
	)
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "1/1") || !strings.Contains(bodyStr, "1 onbekend") {
		t.Errorf("fleet page missing \"1/1 · 1 onbekend\"; body:\n%s", bodyStr)
	}
}

func TestFleetPage_ShowsDeltaAndFlippedFlow(t *testing.T) {
	// run 03 task 3.2: the change since the previous scan, and which
	// flow flipped.
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "mx in NL", Score: models.ScoreSoeverein},
	)
	time.Sleep(5 * time.Millisecond)
	seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.mx_vendor_jurisdiction", Verdict: "mx hosts in US (outside EEA)", Score: models.ScoreAfhankelijk},
	)
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{"x -1", "omgeslagen: Mail"} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("fleet page missing %q; body:\n%s", want, bodyStr)
		}
	}
}

func TestFleetPage_FirstScanHasNoDelta(t *testing.T) {
	srv, st := newServer(t, "")
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	seedFleetScan(
		t, st, tgt.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
	)
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "eerste scan") {
		t.Errorf("fleet page missing \"eerste scan\" for a domain with only one scan; body:\n%s", string(body))
	}
}

func TestFleetPage_SortByScorePutsWorstFirst(t *testing.T) {
	srv, st := newServer(t, "")
	good, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "goed.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	bad, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "slecht.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	seedFleetScan(
		t, st, good.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in NL", Score: models.ScoreSoeverein},
	)
	seedFleetScan(
		t, st, bad.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in US", Score: models.ScoreAfhankelijk},
	)
	resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet?sort=score")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	iBad := strings.Index(bodyStr, "slecht.nl")
	iGood := strings.Index(bodyStr, "goed.nl")
	if iBad == -1 || iGood == -1 {
		t.Fatalf("both domains must render; body:\n%s", bodyStr)
	}
	if iBad > iGood {
		t.Errorf("sort=score must put the worst-scoring domain first: slecht.nl at %d, goed.nl at %d", iBad, iGood)
	}
	if !strings.Contains(bodyStr, "<strong>Score</strong>") {
		t.Errorf("active sort link must stay visible; body:\n%s", bodyStr)
	}
}

func TestFleetPage_UnscannedDomainStaysLastRegardlessOfSort(t *testing.T) {
	srv, st := newServer(t, "")
	scanned, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "gescand.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "nooit.nl"); err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	// A poor score for the scanned domain: sorting naively by a 0-valued
	// score for "nooit.nl" could otherwise put it ahead of a genuinely
	// bad, but scanned, domain.
	seedFleetScan(
		t, st, scanned.ID,
		models.Rationale{CriteriumID: "wand.juridisch.apex_ip_eea", Verdict: "apex in US", Score: models.ScoreAfhankelijk},
	)
	for _, sortKey := range []string{"", "score", "change", "last_scan"} {
		resp, err := http.Get(srv.URL + "/ui/orgs/default/fleet?sort=" + sortKey)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)
		iScanned := strings.Index(bodyStr, "gescand.nl")
		iUnscanned := strings.Index(bodyStr, "nooit.nl")
		if iScanned == -1 || iUnscanned == -1 {
			t.Fatalf("sort=%q: both domains must render; body:\n%s", sortKey, bodyStr)
		}
		if iUnscanned < iScanned {
			t.Errorf("sort=%q: unscanned domain must stay last, got nooit.nl before gescand.nl", sortKey)
		}
	}
}

// TestDutchPages_DeclareDutchLangAttribute covers run 05 task 5.3: a
// page whose content is Dutch must declare <html lang="nl">, not the
// "en" a screen reader would otherwise mispronounce it as.
func TestDutchPages_DeclareDutchLangAttribute(t *testing.T) {
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand")
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}

	for _, path := range []string{
		"/ui/",
		"/ui/scans/" + scanID + "/answer",
		"/ui/scans/" + scanID + "/assessment",
		"/ui/orgs/default/fleet",
	} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("%s: get: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)
		if !strings.Contains(bodyStr, `<html lang="nl">`) {
			t.Errorf("%s: expected <html lang=\"nl\"> for Dutch content; body:\n%s", path, bodyStr)
		}
	}
}

// stubSchedules is a fixed schedule list for tests that need
// ui.Options.Schedules without spinning up a real *scheduler.Scheduler
// (which requires a store + scanner).
type stubSchedules []scheduler.Schedule

func (s stubSchedules) Schedules() []scheduler.Schedule { return s }
