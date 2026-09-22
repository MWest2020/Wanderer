// standards_rules.go implements the six wand-native `standards`
// rules (habitat run 03, tasks 3.1/3.2): Internet.nl's authoritative
// verdicts, imported via `wanderer import internetnl`
// (internal/scanner/netnlimport.go), mapped onto the wand four-value
// scale. See design.md "Design gate outcome" and
// runs/03-assessor.md for the measured category/test mapping this
// file encodes — it is checked against two real batches
// (api.westerweel.work, westerweel.work, 2026-09-22), not against
// the original contract text, which turned out to disagree with the
// API on five points.
package wand

import (
	"fmt"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// StandardsMaxAgeDays is task 3.2's configurable staleness boundary,
// default 30 days (design.md "Verdict mapping"): internetnl.*
// findings older than this score onbekend ("measurement stale")
// instead of being presented as current. Kept as a named constant —
// like domainExpirySafeDays and certValidityExpiringSoonDays — and
// surfaced on every standards rule's Thresholds so a reader sees the
// boundary on the rule page, not just in code.
const StandardsMaxAgeDays = 30

// StandardsMaxAge is StandardsMaxAgeDays as a time.Duration, the form
// scoreStandardsCategory actually compares against.
const StandardsMaxAge = StandardsMaxAgeDays * 24 * time.Hour

// standardsMaxAgeThreshold documents StandardsMaxAge on a rule page.
// A function (not a package var) so every rule gets its own copy —
// assessor.Rule.Thresholds is a slice and rules must not alias one
// another's backing array.
func standardsMaxAgeThreshold() assessor.Threshold {
	return assessor.Threshold{
		Name:  "standards_max_age_days",
		Value: StandardsMaxAgeDays,
		Unit:  "dagen",
		Explanation: fmt.Sprintf(
			"internetnl-metingen ouder dan %d dagen worden als verouderd beschouwd en scoren onbekend",
			StandardsMaxAgeDays,
		),
	}
}

// standardsCategoryRule is one row of the Internet.nl API category →
// wand.standards rule mapping (runs/03-assessor.md "De mapping van
// API-categorie naar regel"), measured against two real batches. The
// API already resolves web_ns_rpki_*/mail_ns_rpki_*/mail_mx_ns_rpki_*
// onto the web_rpki/mail_rpki category itself — netnl reads
// GET /metadata/report's hierarchy to do it (design.md finding 7) —
// so this table only needs the category name, never a test-name
// prefix rule.
//
// web_appsecpriv is deliberately absent: its tests (security.txt,
// security headers) stay first-party under the accountability
// dimension (spec.md "never double-score first-party ground") — see
// TestStandardsRules_NeverScoreAppsecpriv.
type standardsCategoryRule struct {
	category string
	ruleID   string
}

var standardsCategoryRules = []standardsCategoryRule{
	{"web_dnssec", "wand.standards.dnssec"},
	{"mail_dnssec", "wand.standards.dnssec"},
	{"mail_auth", "wand.standards.mail_auth"},
	// The rule ID is starttls_dane (spec.md); the API's own category
	// for it is mail_starttls (design.md finding 3) — there is no
	// "starttls_dane" category anywhere in a real response. Mapped
	// explicitly here rather than derived, so the next reader does
	// not go looking for a category that does not exist.
	{"mail_starttls", "wand.standards.starttls_dane"},
	{"web_ipv6", "wand.standards.ipv6"},
	{"mail_ipv6", "wand.standards.ipv6"},
	{"web_rpki", "wand.standards.rpki"},
	{"mail_rpki", "wand.standards.rpki"},
	{"web_https", "wand.standards.tls_config"},
}

// standardsCategoriesFor returns every API category that feeds
// ruleID, derived from standardsCategoryRules so the rule's Match
// (this file) and any test asserting the mapping (standards_rules_test.go)
// read the same single source of truth.
func standardsCategoriesFor(ruleID string) []string {
	var out []string
	for _, cr := range standardsCategoryRules {
		if cr.ruleID == ruleID {
			out = append(out, cr.category)
		}
	}
	return out
}

// standardsRules returns the six standards-dimension rules, wired
// into DefaultRules().
func standardsRules() []assessor.Rule {
	return []assessor.Rule{
		standardsDNSSEC(),
		standardsMailAuth(),
		standardsStarttlsDane(),
		standardsIPv6(),
		standardsRPKI(),
		standardsTLSConfig(),
	}
}

func standardsDNSSEC() assessor.Rule {
	id := "wand.standards.dnssec"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "DNSSEC is signed and valid for the web and mail zones (Internet.nl).",
		Observation: "Internet.nl's DNSSEC tests (`internetnl.web.web_dnssec_*`, `internetnl.mail.mail_dnssec_*`), imported via `wanderer import internetnl`.",
		Rationale: "DNSSEC lets a resolver verify that a DNS answer has not been tampered " +
			"with in transit — without it, a network-level attacker (a rogue Wi-Fi " +
			"hotspot, a compromised ISP resolver, a BGP hijack) can substitute forged " +
			"answers for the domain's own zone or its mail zone with no visible sign " +
			"to the visitor. Internet.nl already runs the authoritative validation " +
			"Wanderer would otherwise have to reimplement; this rule only maps its " +
			"verdict, never recomputes it.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "DNSSEC", StandardsMaxAge, time.Now())
		},
	}
}

func standardsMailAuth() assessor.Rule {
	id := "wand.standards.mail_auth"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "SPF, DKIM and DMARC are published and correctly configured (Internet.nl).",
		Observation: "Internet.nl's mail authentication tests (`internetnl.mail.mail_auth_*`), imported via `wanderer import internetnl`.",
		Rationale: "SPF, DKIM and DMARC are the three standards that let a receiving mail " +
			"server tell a genuine message from this domain apart from a spoofed one. " +
			"Missing or misconfigured mail authentication makes phishing that " +
			"impersonates the organisation's own domain trivial to send and hard for " +
			"recipients to detect — the opposite of what a public-sector sender's " +
			"reputation depends on.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "SPF/DKIM/DMARC", StandardsMaxAge, time.Now())
		},
	}
}

func standardsStarttlsDane() assessor.Rule {
	id := "wand.standards.starttls_dane"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "Inbound mail transport is encrypted and authenticated with STARTTLS and DANE (Internet.nl).",
		Observation: "Internet.nl's mail transport-security tests, API category `mail_starttls` (`internetnl.mail.mail_starttls_*`), imported via `wanderer import internetnl`.",
		Rationale: "STARTTLS encrypts mail in transit between mail servers; DANE lets the " +
			"receiving server verify the certificate against a DNSSEC-signed DNS " +
			"record instead of trusting the public CA system alone. Without both, " +
			"mail addressed to the organisation can be silently downgraded to " +
			"plaintext or intercepted by an on-path party between the sending and " +
			"receiving mail servers — invisible to both sender and recipient.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "STARTTLS/DANE", StandardsMaxAge, time.Now())
		},
	}
}

func standardsIPv6() assessor.Rule {
	id := "wand.standards.ipv6"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "The website and mail servers are reachable over IPv6 (Internet.nl).",
		Observation: "Internet.nl's IPv6 reachability tests (`internetnl.web.web_ipv6_*`, `internetnl.mail.mail_ipv6_*`), imported via `wanderer import internetnl`.",
		Rationale: "IPv6 is the comply-or-explain baseline for Dutch public-sector " +
			"connectivity: IPv4 address space is exhausted and increasingly shared " +
			"through carrier-grade NAT, eroding the direct, auditable path between a " +
			"citizen and a government service. A domain unreachable over IPv6 " +
			"depends indefinitely on IPv4-only infrastructure and the intermediaries " +
			"that come with it.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "IPv6", StandardsMaxAge, time.Now())
		},
	}
}

func standardsRPKI() assessor.Rule {
	id := "wand.standards.rpki"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "The domain's web, mail and nameserver routes are covered by valid RPKI route-origin authorisations (Internet.nl).",
		Observation: "Internet.nl's RPKI tests, API categories `web_rpki` and `mail_rpki` — which include the nameserver-RPKI subtests the API groups under the same category (`internetnl.web.web_ns_rpki_*`, `internetnl.mail.mail_ns_rpki_*`, `internetnl.mail.mail_mx_ns_rpki_*`) — imported via `wanderer import internetnl`.",
		Rationale: "RPKI lets a network cryptographically prove which Autonomous System " +
			"is authorised to originate a route, closing the BGP hijacking window " +
			"that has taken down public services before. A domain's own route " +
			"matters, but so does its nameservers' — a hijacked nameserver route can " +
			"redirect every lookup for the zone, web and mail alike, without " +
			"touching the origin server at all.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "RPKI", StandardsMaxAge, time.Now())
		},
	}
}

func standardsTLSConfig() assessor.Rule {
	id := "wand.standards.tls_config"
	return assessor.Rule{
		ID:          id,
		Dimension:   models.DimensionStandards,
		Description: "The website's TLS configuration (certificate chain, protocol versions, cipher suites, HSTS) meets current best practice (Internet.nl).",
		Observation: "Internet.nl's web HTTPS/TLS tests, API category `web_https` (`internetnl.web.web_https_*`), imported via `wanderer import internetnl`.",
		Rationale: "TLS configuration quality determines how much of the cryptographic " +
			"promise a valid certificate actually delivers: a correct chain with a " +
			"weak cipher suite or an outdated protocol version is still vulnerable " +
			"to a capable on-path attacker. Internet.nl's TLS test suite already " +
			"tracks the current best-practice baseline as it evolves; this rule " +
			"maps its verdict rather than pinning Wanderer to a fixed cipher list " +
			"that would go stale.",
		Thresholds: []assessor.Threshold{standardsMaxAgeThreshold()},
		Match: func(findings []models.Finding) assessor.RuleResult {
			return scoreStandardsCategory(findings, standardsCategoriesFor(id), "TLS configuration", StandardsMaxAge, time.Now())
		},
	}
}

// standardsTally accumulates one rule's relevant internetnl.* test
// results (design.md "Verdict mapping"). "stale" tests (older than
// maxAge) and "not_tested"/"error" tests are excluded from the
// pass/fail/warn tally: not_tested is Internet.nl's normal "this was
// not measured" state (runs/03-assessor.md: 18 of 38 real mail
// tests), and error means the measurement itself broke — never
// evidence of a failing target (design.md finding 2, "error is geen
// failed").
type standardsTally struct {
	total, passed, failed, warned, notTested, errored, stale int

	allEvidence      []string // every relevant finding's ID, whatever its status
	measuredEvidence []string // finding IDs behind passed/failed/warned only
	failedTests      []string
	warnedTests      []string
	reportURLs       []string // deduped, first-seen order
	newestStaleAt    time.Time
}

func (t *standardsTally) addReportURL(url string, seen map[string]bool) {
	if url == "" || seen[url] {
		return
	}
	seen[url] = true
	t.reportURLs = append(t.reportURLs, url)
}

// scoreStandardsCategory is the shared verdict-mapping engine behind
// all six standards rules (design.md "Verdict mapping"):
//
//   - no relevant internetnl.* finding at all → onbekend, "not measured"
//   - every relevant finding older than maxAge → onbekend, "measurement stale"
//   - relevant findings present but none actually tested (all
//     not_tested/error, Internet.nl's normal state for a domain
//     without the corresponding service) → onbekend, "not measured"
//   - every tested finding passed → soeverein
//   - at least one tested finding passed and at least one
//     failed/warned → voldoende (a genuine mix)
//   - tested findings present and none passed → afhankelijk
//     ("substantively failed" — design.md)
//
// It never recomputes Internet.nl's own percentage score (the
// clever valkuil design.md warns against) — only the per-test
// status/verdict fields feed the mapping.
func scoreStandardsCategory(findings []models.Finding, categories []string, label string, maxAge time.Duration, now time.Time) assessor.RuleResult {
	want := map[string]bool{}
	for _, c := range categories {
		want[c] = true
	}

	var t standardsTally
	urlSeen := map[string]bool{}
	for _, fnd := range findings {
		if !strings.HasPrefix(fnd.ProbeID, "internetnl.") {
			continue
		}
		category := stringFromAttr(fnd.Attributes, "category")
		if !want[category] {
			continue
		}
		t.total++
		t.allEvidence = append(t.allEvidence, fnd.ID)
		t.addReportURL(stringFromAttr(fnd.Attributes, "report_url"), urlSeen)

		measuredAt, parseErr := time.Parse(time.RFC3339, stringFromAttr(fnd.Attributes, "measured_at"))
		if parseErr == nil && now.Sub(measuredAt) > maxAge {
			t.stale++
			if measuredAt.After(t.newestStaleAt) {
				t.newestStaleAt = measuredAt
			}
			continue
		}

		test := stringFromAttr(fnd.Attributes, "test")
		switch stringFromAttr(fnd.Attributes, "status") {
		case "passed":
			t.passed++
			t.measuredEvidence = append(t.measuredEvidence, fnd.ID)
		case "failed":
			t.failed++
			t.measuredEvidence = append(t.measuredEvidence, fnd.ID)
			t.failedTests = append(t.failedTests, test)
		case "warning", "info":
			t.warned++
			t.measuredEvidence = append(t.measuredEvidence, fnd.ID)
			t.warnedTests = append(t.warnedTests, test)
		case "not_tested":
			t.notTested++
		case "error":
			t.errored++
		default:
			// An unrecognised status is treated like not_tested —
			// conservative, counts neither as evidence nor as
			// "measured" (spec.md: "verdict mapping only").
			t.notTested++
		}
	}

	if t.total == 0 {
		return assessor.RuleResult{
			Score:   models.ScoreOnbekend,
			Verdict: fmt.Sprintf("no internetnl.* findings imported for %s — not measured", label),
			Reason:  assessor.ReasonNotMeasured,
		}
	}

	urls := strings.Join(t.reportURLs, ", ")
	urlNote := ""
	if urls != "" {
		urlNote = " — see " + urls
	}

	measured := t.passed + t.failed + t.warned
	if measured == 0 {
		if t.stale > 0 && t.stale == t.total {
			return assessor.RuleResult{
				Score: models.ScoreOnbekend,
				Verdict: fmt.Sprintf(
					"%s measurement stale (measured %s)%s",
					label, t.newestStaleAt.UTC().Format("2006-01-02"), urlNote,
				),
				Evidence: t.allEvidence,
				Reason:   assessor.ReasonMeasurementStale,
			}
		}
		detail := fmt.Sprintf("%s not measured: %d of %d test(s) not_tested", label, t.notTested, t.total)
		if t.errored > 0 {
			detail = fmt.Sprintf("%s, %d measurement(s) failed", detail, t.errored)
		}
		return assessor.RuleResult{
			Score:    models.ScoreOnbekend,
			Verdict:  detail + urlNote,
			Evidence: t.allEvidence,
			Reason:   assessor.ReasonNotMeasured,
		}
	}

	staleNote := ""
	if t.stale > 0 {
		staleNote = fmt.Sprintf(" (%d older test(s) excluded as stale)", t.stale)
	}

	switch {
	case t.passed == measured:
		return assessor.RuleResult{
			Score:    models.ScoreSoeverein,
			Verdict:  fmt.Sprintf("all %d tested %s test(s) passed%s%s", measured, label, staleNote, urlNote),
			Evidence: t.measuredEvidence,
		}
	case t.failed > 0 && t.passed == 0:
		return assessor.RuleResult{
			Score: models.ScoreAfhankelijk,
			Verdict: fmt.Sprintf(
				"%s substantively failed: %s%s%s",
				label, strings.Join(dedupe(t.failedTests), ", "), staleNote, urlNote,
			),
			Evidence: t.measuredEvidence,
		}
	default:
		var mixed []string
		if len(t.failedTests) > 0 {
			mixed = append(mixed, fmt.Sprintf("failed: %s", strings.Join(dedupe(t.failedTests), ", ")))
		}
		if len(t.warnedTests) > 0 {
			mixed = append(mixed, fmt.Sprintf("warning: %s", strings.Join(dedupe(t.warnedTests), ", ")))
		}
		return assessor.RuleResult{
			Score: models.ScoreVoldoende,
			Verdict: fmt.Sprintf(
				"%d of %d tested %s test(s) passed; %s%s%s",
				t.passed, measured, label, strings.Join(mixed, "; "), staleNote, urlNote,
			),
			Evidence: t.measuredEvidence,
		}
	}
}
