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

func TestDashboard_VerdictPill_PerFramework(t *testing.T) {
	// After the 2026-05-10 layer restructure, Dashboard renders
	// per-framework verdict pills (worst-score) — no posture
	// distribution blocks, no Top concerns table, no Recent
	// activity table.
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
	resp, err := http.Get(srv.URL + "/ui/")
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
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("dashboard missing %q", want)
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
		"Assessment for scan",
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
	for _, banned := range []string{"r.Post(", "r.Put(", "r.Patch(", "r.Delete("} {
		if strings.Contains(string(src), banned) {
			t.Errorf("ui package contains mutating handler registration: %s", banned)
		}
	}
}

func TestReporting_Catalogue_ListsRulesWithDescriptions(t *testing.T) {
	// /ui/reporting is the rule catalogue after the 2026-05-10 layer
	// restructure. No scoring data here; just rule descriptions.
	srv, _ := newServer(t, "")
	resp, err := http.Get(srv.URL + "/ui/reporting")
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
		"Reporting · rule catalogue",
		"wand.juridisch.cert_issuer_eea",
		"eucsf.sov2.cert_issuer_eu",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("reporting catalogue missing %q", want)
		}
	}
	// Catalogue must NOT carry scoring data — that lives on /ui/analysis.
	if strings.Contains(bodyStr, ">soeverein<") {
		t.Errorf("reporting catalogue must not include score columns")
	}
}

func TestAnalysis_RulesMatrix(t *testing.T) {
	// The rule × score matrix moved to /ui/analysis on 2026-05-10.
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	seedAssessment(t, st, scanID, "wand", "eucsf")
	resp, err := http.Get(srv.URL + "/ui/analysis")
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
		"Analysis · steering matrix",
		"wand.juridisch.cert_issuer_eea",
		"eucsf.sov2.cert_issuer_eu",
		"soeverein",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("analysis matrix missing %q", want)
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
		"ACME B.V.",      // headline rebadged
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
		// Analysis tab threads the org filter; Reporting catalogue
		// does not carry org filter (it's reference data, not
		// scope-bound).
		`href="/ui/analysis?org=acme"`,
		`href="/ui/reporting"`,
		`href="/ui/orgs/acme"`,
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("per-org page nav missing %q", want)
		}
	}
}

func TestNav_AnalysisPagesIncludeReportingTab(t *testing.T) {
	// Regression: scan/assessment/drift/targets pages used to pass
	// HasReporting=false to nav.tmpl, omitting the Reporting link.
	srv, st := newServer(t, "")
	_, scanID := seed(t, st)
	for _, path := range []string{"/ui/targets", "/ui/scans/" + scanID, "/ui/scans/" + scanID + "/assessment"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// Accept either `href="/ui/reporting"` or `href="/ui/reporting?org=...`
		// — the latter happens whenever the page resolves a scope through
		// the seeded `default` org. The point is the link is *present*.
		if !strings.Contains(string(body), `href="/ui/reporting`) {
			t.Errorf("%s: Reporting nav link missing", path)
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

func TestDashboard_Global_ListsOrganisationsWhenMultiple(t *testing.T) {
	srv, st := newServer(t, "")
	for _, slug := range []string{"acme", "beta"} {
		o := &models.Organisation{Slug: slug, Name: slug}
		if err := st.UpsertOrganisation(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}
	// Seed at least one scan so HasData=true and the org list block renders.
	tgt := &models.Target{Domain: "a.example"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateScan(context.Background(), tgt.ID); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/ui/")
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
			t.Errorf("global dashboard missing %q", want)
		}
	}
}
