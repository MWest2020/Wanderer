package wand

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/pkg/models"
)

// standardsFinding builds one internetnl.* finding the way
// internal/scanner/netnlimport.go's NetnlFindings does, for tests
// that construct their own small finding sets rather than loading
// the golden fixtures.
func standardsFinding(id, typ, test, category, status string, measuredAt time.Time, reportURL string) models.Finding {
	return models.Finding{
		ID:      id,
		ProbeID: "internetnl." + typ + "." + test,
		Subject: "westerweel.work",
		Attributes: map[string]any{
			"domain":      "westerweel.work",
			"type":        typ,
			"test":        test,
			"category":    category,
			"status":      status,
			"measured_at": measuredAt.UTC().Format(time.RFC3339),
			"report_url":  reportURL,
		},
	}
}

// loadGoldenStandardsFindings loads both golden netnl fixtures (task
// 1.3's testdata copies, measured against api.westerweel.work on
// 2026-09-22 — design.md "Design gate outcome") and converts them
// with the same scanner.NetnlFindings the real importer uses, so
// these tests exercise the exact shape the assessor sees in
// production, not a hand-rolled approximation.
//
// The fixtures' measured_at is fixed at 2026-09-22; freshenMeasuredAt
// rewrites it to "now" so a test run any time after that date does
// not start seeing the golden data as stale. Staleness itself is
// tested separately with synthetic findings that pin their own
// reference time.
func loadGoldenStandardsFindings(t *testing.T) []models.Finding {
	t.Helper()
	var out []models.Finding
	for _, name := range []string{"findings-v1-web-20260922.json", "findings-v1-mail-20260922.json"} {
		path := filepath.Join("..", "..", "scanner", "testdata", name)
		f, err := scanner.LoadNetnlFindings(path, nil)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		for _, d := range f.Domains {
			out = append(out, scanner.NetnlFindings(d)...)
		}
	}
	return freshenMeasuredAt(out)
}

// freshenMeasuredAt rewrites every finding's measured_at attribute to
// "now" (see loadGoldenStandardsFindings) and assigns Finding.ID from
// ProbeID when empty — scanner.NetnlFindings never sets ID (real IDs
// are assigned at persistence time), but ProbeID is already unique
// per test across both fixtures ("internetnl.web.*" vs
// "internetnl.mail.*" never collide), so it doubles as a stable,
// distinct ID the identity-sensitive tests in this file can compare
// against.
func freshenMeasuredAt(findings []models.Finding) []models.Finding {
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]models.Finding, len(findings))
	for i, f := range findings {
		attrs := make(map[string]any, len(f.Attributes))
		for k, v := range f.Attributes {
			attrs[k] = v
		}
		if _, ok := attrs["measured_at"]; ok {
			attrs["measured_at"] = now
		}
		f.Attributes = attrs
		if f.ID == "" {
			f.ID = f.ProbeID
		}
		out[i] = f
	}
	return out
}

// standardsMatchedCount counts every internetnl.* finding whose
// category attribute is one of categories — "how many tests the rule
// sees" (runs/03-assessor.md), independent of what verdict those
// tests produce. Deliberately separate from RuleResult.Evidence,
// which for a scored rule only lists the tests that actually counted
// toward the verdict (not_tested/error/stale tests are matched but
// excluded from scoring) — this helper is the completeness check,
// Evidence is the scoring check.
func standardsMatchedCount(findings []models.Finding, categories []string) int {
	want := map[string]bool{}
	for _, c := range categories {
		want[c] = true
	}
	n := 0
	for _, f := range findings {
		if !strings.HasPrefix(f.ProbeID, "internetnl.") {
			continue
		}
		if want[stringFromAttr(f.Attributes, "category")] {
			n++
		}
	}
	return n
}

// TestStandardsRules_GoldenFixtureCounts pins runs/03-assessor.md
// "Hoeveel tests elke regel hoort te zien": the exact number of
// Internet.nl tests each standards rule maps to, counted from the
// real metadata-hierarchy-resolved category on every test in both
// golden fixtures combined (web + mail, westerweel.work,
// 2026-09-22). 76 of 76 measured tests are accounted for across the
// six rules plus the deliberately-unscored web_appsecpriv category —
// nothing is silently uncategorised. A rule that sees fewer than its
// pinned count is scoring on incomplete data while still sounding
// definite, most sharply for RPKI's nameserver subtests (see
// TestStandardsRules_RPKIIncludesNameserverTests).
func TestStandardsRules_GoldenFixtureCounts(t *testing.T) {
	findings := loadGoldenStandardsFindings(t)
	if len(findings) != 76 {
		t.Fatalf("test setup: golden fixtures carry %d findings, want 76 (38 web + 38 mail)", len(findings))
	}

	cases := []struct {
		ruleID    string
		wantCount int
		wantScore models.Score
	}{
		{"wand.standards.tls_config", 22, models.ScoreVoldoende},
		{"wand.standards.starttls_dane", 19, models.ScoreOnbekend}, // 18 not_tested + 1 error: nothing measured
		{"wand.standards.rpki", 10, models.ScoreSoeverein},
		{"wand.standards.ipv6", 9, models.ScoreVoldoende},
		{"wand.standards.dnssec", 6, models.ScoreSoeverein},
		{"wand.standards.mail_auth", 5, models.ScoreSoeverein},
	}

	total := 0
	for _, tc := range cases {
		total += tc.wantCount
		t.Run(tc.ruleID, func(t *testing.T) {
			if got := standardsMatchedCount(findings, standardsCategoriesFor(tc.ruleID)); got != tc.wantCount {
				t.Errorf("matched test count = %d, want %d", got, tc.wantCount)
			}
			r := ruleByID(t, tc.ruleID)
			res := r.Match(findings)
			if res.Score != tc.wantScore {
				t.Errorf("score = %s, want %s (verdict: %s)", res.Score, tc.wantScore, res.Verdict)
			}
		})
	}

	appsecpriv := standardsMatchedCount(findings, []string{"web_appsecpriv"})
	if appsecpriv != 5 {
		t.Errorf("web_appsecpriv matched count = %d, want 5 (unscored, but must still exist in the fixture)", appsecpriv)
	}
	if total+appsecpriv != len(findings) {
		t.Errorf("rule counts (%d) + unscored web_appsecpriv (%d) = %d, want %d (every fixture test accounted for)",
			total, appsecpriv, total+appsecpriv, len(findings))
	}
}

// TestStandardsRules_RPKIIncludesNameserverTests pins
// runs/03-assessor.md finding 7 directly: the RPKI rule must see the
// web_ns_rpki_*/mail_ns_rpki_*/mail_mx_ns_rpki_* subtests too, not
// just the four tests whose own name starts with the category name.
// A naive prefix-match implementation scores 4 of 10 here while still
// returning soeverein — the exact silently-green gap the run's task
// description warns against.
func TestStandardsRules_RPKIIncludesNameserverTests(t *testing.T) {
	findings := loadGoldenStandardsFindings(t)
	r := ruleByID(t, "wand.standards.rpki")
	res := r.Match(findings)

	wantTests := []string{
		"web_ns_rpki_exists", "web_ns_rpki_valid",
		"mail_ns_rpki_exists", "mail_ns_rpki_valid",
		"mail_mx_ns_rpki_exists", "mail_mx_ns_rpki_valid",
	}
	evidenceIDs := map[string]bool{}
	for _, id := range res.Evidence {
		evidenceIDs[id] = true
	}
	for _, want := range wantTests {
		found := false
		for _, f := range findings {
			if strings.HasSuffix(f.ProbeID, "."+want) && evidenceIDs[f.ID] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RPKI rule evidence does not include nameserver test %q", want)
		}
	}
	if res.Score != models.ScoreSoeverein {
		t.Errorf("score = %s, want soeverein (all 10 RPKI tests passed in the golden fixtures)", res.Score)
	}
}

// TestStandardsRules_NeverScoreAppsecpriv pins spec.md "Standards
// rules never double-score first-party ground": no standards rule's
// category set includes web_appsecpriv, and running the six rules
// against a fixture that carries web_appsecpriv_securitytxt never
// cites that finding as evidence. security.txt stays scored solely
// by wand.accountability.securitytxt.
func TestStandardsRules_NeverScoreAppsecpriv(t *testing.T) {
	for _, cr := range standardsCategoryRules {
		if cr.category == "web_appsecpriv" {
			t.Fatalf("standardsCategoryRules maps web_appsecpriv to %q — it must stay unscored (never double-score first-party ground)", cr.ruleID)
		}
	}

	findings := loadGoldenStandardsFindings(t)
	var securitytxtID string
	for _, f := range findings {
		if strings.HasSuffix(f.ProbeID, ".web_appsecpriv_securitytxt") {
			securitytxtID = f.ID
		}
	}
	if securitytxtID == "" {
		t.Fatal("test setup: golden web fixture should carry a web_appsecpriv_securitytxt finding")
	}

	for _, r := range standardsRules() {
		res := r.Match(findings)
		for _, id := range res.Evidence {
			if id == securitytxtID {
				t.Errorf("rule %s cites the web_appsecpriv_securitytxt finding as evidence — first-party ground must not be double-scored", r.ID)
			}
		}
	}
}

// TestStandardsRules_NotMeasured pins spec.md "No import present":
// with no internetnl.* findings at all, every standards rule scores
// onbekend with reason "not measured", regardless of what other
// (non-internetnl) findings are present.
func TestStandardsRules_NotMeasured(t *testing.T) {
	other := []models.Finding{
		{ID: "f1", ProbeID: "dns.mx", Subject: "example.nl", Attributes: map[string]any{"host": "mx.example.nl"}},
	}
	for _, r := range standardsRules() {
		res := r.Match(other)
		if res.Score != models.ScoreOnbekend {
			t.Errorf("rule %s: score = %s, want onbekend", r.ID, res.Score)
		}
		if res.Reason != assessor.ReasonNotMeasured {
			t.Errorf("rule %s: reason = %q, want %q", r.ID, res.Reason, assessor.ReasonNotMeasured)
		}
		if len(res.Evidence) != 0 {
			t.Errorf("rule %s: evidence = %v, want empty (no internetnl finding exists)", r.ID, res.Evidence)
		}
	}
}

// TestStandardsRules_NotTestedIsNotSoeverein pins runs/03-assessor.md
// "not_tested is de normale toestand, niet de uitzondering": a
// category whose every test is not_tested must score onbekend, never
// soeverein — a domain without a full mail setup must not read as
// mail-secure. Also confirms error is folded into the same onbekend
// bucket rather than dragging the category to afhankelijk
// (design.md finding 2, "error is geen failed").
func TestStandardsRules_NotTestedIsNotSoeverein(t *testing.T) {
	now := time.Now()
	findings := []models.Finding{
		standardsFinding("f1", "mail", "mail_starttls_cert_chain", "mail_starttls", "not_tested", now, "https://netnl.example/mail/x/1/"),
		standardsFinding("f2", "mail", "mail_starttls_dane_exist", "mail_starttls", "not_tested", now, "https://netnl.example/mail/x/1/"),
		standardsFinding("f3", "mail", "mail_starttls_tls_available", "mail_starttls", "error", now, "https://netnl.example/mail/x/1/"),
	}
	r := ruleByID(t, "wand.standards.starttls_dane")
	res := r.Match(findings)
	if res.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend (verdict: %s)", res.Score, res.Verdict)
	}
	if res.Reason != assessor.ReasonNotMeasured {
		t.Errorf("reason = %q, want %q", res.Reason, assessor.ReasonNotMeasured)
	}
	if !strings.Contains(res.Verdict, "not measured") {
		t.Errorf("verdict = %q, want it to say not measured", res.Verdict)
	}
}

// TestStandardsRules_ErrorAloneIsNotAfhankelijk is the single-test
// version of design.md's "error is geen failed" correction: a
// category with one erroring test and nothing else present must not
// score afhankelijk, which would charge the target for a measurement
// failure that was Internet.nl's own, not the target's.
func TestStandardsRules_ErrorAloneIsNotAfhankelijk(t *testing.T) {
	now := time.Now()
	findings := []models.Finding{
		standardsFinding("f1", "mail", "mail_starttls_tls_available", "mail_starttls", "error", now, "https://netnl.example/mail/x/1/"),
	}
	r := ruleByID(t, "wand.standards.starttls_dane")
	res := r.Match(findings)
	if res.Score == models.ScoreAfhankelijk {
		t.Fatalf("score = afhankelijk, want onbekend — a failed measurement is not evidence of a failing target (verdict: %s)", res.Verdict)
	}
	if res.Score != models.ScoreOnbekend {
		t.Errorf("score = %s, want onbekend", res.Score)
	}
}

// TestStandardsRules_MixedVerdict pins design.md's "any warning, or a
// mix of passed/failed within the category → voldoende": a category
// with both a passing and a failing test is neither soeverein
// (not all passed) nor afhankelijk (not substantively failed, since
// a passing test is also present) — it is voldoende.
func TestStandardsRules_MixedVerdict(t *testing.T) {
	now := time.Now()
	findings := []models.Finding{
		standardsFinding("f1", "mail", "mail_ipv6_mx_address", "mail_ipv6", "passed", now, "https://netnl.example/mail/x/1/"),
		standardsFinding("f2", "mail", "mail_ipv6_mx_reach", "mail_ipv6", "failed", now, "https://netnl.example/mail/x/1/"),
	}
	r := ruleByID(t, "wand.standards.ipv6")
	res := r.Match(findings)
	if res.Score != models.ScoreVoldoende {
		t.Fatalf("score = %s, want voldoende (verdict: %s)", res.Score, res.Verdict)
	}
	if len(res.Evidence) != 2 {
		t.Errorf("evidence = %v, want both findings cited", res.Evidence)
	}
}

// TestStandardsRules_SubstantiallyFailed pins spec.md's "Failed mail
// authentication" scenario: a category where the present tests are
// entirely failed (no passing counterweight) scores afhankelijk and
// names the failing standard(s).
func TestStandardsRules_SubstantiallyFailed(t *testing.T) {
	now := time.Now()
	findings := []models.Finding{
		standardsFinding("f1", "mail", "mail_auth_dmarc_exist", "mail_auth", "failed", now, "https://netnl.example/mail/x/1/"),
		standardsFinding("f2", "mail", "mail_auth_dmarc_policy", "mail_auth", "failed", now, "https://netnl.example/mail/x/1/"),
	}
	r := ruleByID(t, "wand.standards.mail_auth")
	res := r.Match(findings)
	if res.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk (verdict: %s)", res.Score, res.Verdict)
	}
	for _, want := range []string{"mail_auth_dmarc_exist", "mail_auth_dmarc_policy"} {
		if !strings.Contains(res.Verdict, want) {
			t.Errorf("verdict = %q, want it to name the failing standard %q", res.Verdict, want)
		}
	}
}

// TestStandardsRules_AllPassedIsSoeverein pins spec.md "Signed and
// valid DNSSEC": every relevant test passed → soeverein, and the
// Internet.nl report URL is linked in the verdict.
func TestStandardsRules_AllPassedIsSoeverein(t *testing.T) {
	now := time.Now()
	reportURL := "https://netnl.westerweel.work/site/westerweel.work/485/"
	findings := []models.Finding{
		standardsFinding("f1", "web", "web_dnssec_exist", "web_dnssec", "passed", now, reportURL),
		standardsFinding("f2", "web", "web_dnssec_valid", "web_dnssec", "passed", now, reportURL),
	}
	r := ruleByID(t, "wand.standards.dnssec")
	res := r.Match(findings)
	if res.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein (verdict: %s)", res.Score, res.Verdict)
	}
	if !strings.Contains(res.Verdict, reportURL) {
		t.Errorf("verdict = %q, want it to link the report URL %q", res.Verdict, reportURL)
	}
}

// TestStandardsRules_MeasurementPastMaxAge pins design.md's "Stale
// measurements are not presented as current": findings older than
// standards.max_age score onbekend with reason "measurement stale"
// and the verdict names the measurement date.
func TestStandardsRules_MeasurementPastMaxAge(t *testing.T) {
	staleAt := time.Now().Add(-45 * 24 * time.Hour)
	findings := []models.Finding{
		standardsFinding("f1", "web", "web_dnssec_exist", "web_dnssec", "passed", staleAt, "https://netnl.example/site/x/1/"),
		standardsFinding("f2", "web", "web_dnssec_valid", "web_dnssec", "passed", staleAt, "https://netnl.example/site/x/1/"),
	}
	r := ruleByID(t, "wand.standards.dnssec")
	res := r.Match(findings)
	if res.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend (verdict: %s)", res.Score, res.Verdict)
	}
	if res.Reason != assessor.ReasonMeasurementStale {
		t.Errorf("reason = %q, want %q", res.Reason, assessor.ReasonMeasurementStale)
	}
	if !strings.Contains(res.Verdict, "measurement stale") {
		t.Errorf("verdict = %q, want it to say measurement stale", res.Verdict)
	}
	if !strings.Contains(res.Verdict, staleAt.UTC().Format("2006-01-02")) {
		t.Errorf("verdict = %q, want it to name the measurement date", res.Verdict)
	}
}

// TestStandardsRules_FreshReimportRestoresScoring pins design.md's
// "Fresh re-import restores scoring": when a stale finding and a
// fresh finding for the same category coexist (a re-import that
// landed after the previous one went stale), the rule scores from
// the fresh evidence only and ignores the stale one.
func TestStandardsRules_FreshReimportRestoresScoring(t *testing.T) {
	staleAt := time.Now().Add(-45 * 24 * time.Hour)
	freshAt := time.Now().Add(-24 * time.Hour)
	findings := []models.Finding{
		standardsFinding("stale1", "web", "web_dnssec_exist", "web_dnssec", "failed", staleAt, "https://netnl.example/site/x/1/"),
		standardsFinding("fresh1", "web", "web_dnssec_exist", "web_dnssec", "passed", freshAt, "https://netnl.example/site/x/2/"),
		standardsFinding("fresh2", "web", "web_dnssec_valid", "web_dnssec", "passed", freshAt, "https://netnl.example/site/x/2/"),
	}
	r := ruleByID(t, "wand.standards.dnssec")
	res := r.Match(findings)
	if res.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein — the stale failed test must be ignored (verdict: %s)", res.Score, res.Verdict)
	}
	for _, id := range res.Evidence {
		if id == "stale1" {
			t.Errorf("evidence %v includes the stale finding; it must be excluded", res.Evidence)
		}
	}
}

// TestStandardsMaxAge_DefaultIsThirtyDays pins task 3.2.
func TestStandardsMaxAge_DefaultIsThirtyDays(t *testing.T) {
	if StandardsMaxAgeDays != 30 {
		t.Errorf("StandardsMaxAgeDays = %d, want 30", StandardsMaxAgeDays)
	}
	if StandardsMaxAge != 30*24*time.Hour {
		t.Errorf("StandardsMaxAge = %s, want 720h (30 days)", StandardsMaxAge)
	}
	for _, r := range standardsRules() {
		found := false
		for _, th := range r.Thresholds {
			if th.Name == "standards_max_age_days" {
				found = true
				if th.Value != 30 {
					t.Errorf("rule %s: standards_max_age_days Threshold value = %v, want 30", r.ID, th.Value)
				}
			}
		}
		if !found {
			t.Errorf("rule %s: no standards_max_age_days Threshold", r.ID)
		}
	}
}

// TestStandardsRules_RegisteredInDefaultRules pins that all six rules
// are wired into wand.DefaultRules() under the standards dimension.
func TestStandardsRules_RegisteredInDefaultRules(t *testing.T) {
	want := map[string]bool{
		"wand.standards.dnssec":        false,
		"wand.standards.mail_auth":     false,
		"wand.standards.starttls_dane": false,
		"wand.standards.ipv6":          false,
		"wand.standards.rpki":          false,
		"wand.standards.tls_config":    false,
	}
	for _, r := range DefaultRules() {
		if _, ok := want[r.ID]; !ok {
			continue
		}
		want[r.ID] = true
		if r.Dimension != models.DimensionStandards {
			t.Errorf("rule %s: dimension = %s, want standards", r.ID, r.Dimension)
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("standards rule %q not registered in DefaultRules", id)
		}
	}
}
