package ui

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func assessmentWith(rs ...models.Rationale) []models.Assessment {
	return []models.Assessment{{
		Framework:  "wand",
		Dimensions: []models.DimensionScore{{Rationale: rs}},
	}}
}

func TestBuildAnswerVerdict_AllSoevereinIsPlainJa(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in NL", models.ScoreSoeverein),
	)
	v := BuildAnswerVerdict(assessments, nil)
	if v.Verdict != "ja" {
		t.Fatalf("verdict = %q, want ja", v.Verdict)
	}
	if v.Unanswered != 0 {
		t.Errorf("unanswered = %d, want 0", v.Unanswered)
	}
	if v.DecidingFlow != "" {
		t.Errorf("deciding flow = %q, want empty", v.DecidingFlow)
	}
	want := "Ja — dit domein staat onder Nederlands of Europees recht."
	if v.Headline != want {
		t.Errorf("headline = %q, want %q", v.Headline, want)
	}
}

// TestBuildAnswerVerdict_AfhankelijkIsNeeNamingTheFlow also guards the
// run 04b regression: the "nee" headline must stay Dutch (never paste
// the rule's raw English Verdict) while still naming the observed fact
// (here, the mail flow's country) — losing the fact was exactly what
// run 04 broke.
func TestBuildAnswerVerdict_AfhankelijkIsNeeNamingTheFlow(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		models.Rationale{
			CriteriumID: "wand.juridisch.mx_vendor_jurisdiction",
			Verdict:     "mx hosts in US (outside EEA)",
			Score:       models.ScoreAfhankelijk,
			Evidence:    []string{"f1"},
		},
	)
	findings := map[string]models.Finding{
		"f1": {ID: "f1", ProbeID: "ip.asn", Subject: "mail.example.nl", Attributes: map[string]any{"country": "US"}},
	}
	v := BuildAnswerVerdict(assessments, findings)
	if v.Verdict != "nee" {
		t.Fatalf("verdict = %q, want nee", v.Verdict)
	}
	if v.DecidingFlow != "Mail" {
		t.Errorf("deciding flow = %q, want Mail", v.DecidingFlow)
	}
	if v.DecidingVerdict != "De mail wordt buiten de EER gerouteerd — mailservers in US." {
		t.Errorf("deciding verdict = %q, want the Dutch sentence naming US", v.DecidingVerdict)
	}
	if strings.Contains(v.Headline, "mx hosts") || strings.Contains(v.Headline, "EEA") {
		t.Errorf("headline %q leaks the rule's raw English verdict", v.Headline)
	}
	if !strings.Contains(v.Headline, "mail") || !strings.Contains(v.Headline, "US") {
		t.Errorf("headline %q does not name the deciding flow and its observed fact", v.Headline)
	}
	// The old headline pasted the flow label in front of a verdict that
	// already names it ("Nee — Mail: De mail wordt buiten..."), reading
	// as the stroomnaam twice (run 05 task 4.6). The label lives on
	// DecidingFlow for callers that want it on its own.
	if strings.Contains(v.Headline, "Mail:") {
		t.Errorf("headline %q still prefixes the flow label in front of a verdict that names it again", v.Headline)
	}
	if !strings.HasPrefix(v.Headline, "Nee") {
		t.Errorf("headline %q does not read Nee", v.Headline)
	}
}

func TestBuildAnswerVerdict_TwoAfhankelijkPicksTheFixedOrderFirst(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in US", models.ScoreAfhankelijk),
		rationale("wand.juridisch.apex_ip_eea", "apex in US", models.ScoreAfhankelijk),
	)
	v := BuildAnswerVerdict(assessments, nil)
	if v.Verdict != "nee" {
		t.Fatalf("verdict = %q, want nee", v.Verdict)
	}
	// SovereigntyFlows' fixed order puts Hosting before Mail regardless
	// of the Rationale slice's order above.
	if v.DecidingFlow != "Hosting" {
		t.Errorf("deciding flow = %q, want Hosting (fixed-order first)", v.DecidingFlow)
	}
}

func TestBuildAnswerVerdict_HostingSoevereinDNSOnbekendIsNotAPlainJa(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.ns_vendor_jurisdiction", "geoip unavailable", models.ScoreOnbekend),
	)
	v := BuildAnswerVerdict(assessments, nil)
	if v.Verdict != "ja" {
		t.Fatalf("verdict = %q, want ja (no afhankelijk present)", v.Verdict)
	}
	if v.Unanswered != 1 {
		t.Fatalf("unanswered = %d, want 1", v.Unanswered)
	}
	plainJa := "Ja — dit domein staat onder Nederlands of Europees recht."
	if v.Headline == plainJa {
		t.Errorf("headline reads a plain Ja, hiding the unanswered question: %q", v.Headline)
	}
	if !strings.Contains(v.Headline, "1 vraag") {
		t.Errorf("headline %q does not say one question is unanswered", v.Headline)
	}
}

func TestBuildAnswerVerdict_EverythingOnbekendIsOnbekend(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "probe failed", models.ScoreOnbekend),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "probe failed", models.ScoreOnbekend),
	)
	v := BuildAnswerVerdict(assessments, nil)
	if v.Verdict != "onbekend" {
		t.Fatalf("verdict = %q, want onbekend", v.Verdict)
	}
	if v.Unanswered != 2 {
		t.Errorf("unanswered = %d, want 2", v.Unanswered)
	}
	if v.DecidingFlow != "" {
		t.Errorf("deciding flow = %q, want empty — nothing decided a nee", v.DecidingFlow)
	}
	if !strings.Contains(v.Headline, "2 vragen") {
		t.Errorf("headline %q does not say how many questions are unanswered", v.Headline)
	}
}

func TestBuildAnswerVerdict_NoFlowsAtAllIsOnbekend(t *testing.T) {
	v := BuildAnswerVerdict(nil, nil)
	if v.Verdict != "onbekend" {
		t.Fatalf("verdict = %q, want onbekend", v.Verdict)
	}
	if v.Unanswered != 0 {
		t.Errorf("unanswered = %d, want 0 (there were no questions to count)", v.Unanswered)
	}
}

func TestBuildAnswerVerdict_AssessmentPredatingADimensionDoesNotCountAsNee(t *testing.T) {
	// Only Hosting fired — Mail/DNS/etc. never ran because this scan
	// predates those flow rules. SovereigntyFlows simply omits them;
	// they must not read as "nee" or as unanswered questions.
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
	)
	v := BuildAnswerVerdict(assessments, nil)
	if v.Verdict != "ja" {
		t.Fatalf("verdict = %q, want ja", v.Verdict)
	}
	if v.Unanswered != 0 {
		t.Errorf("unanswered = %d, want 0", v.Unanswered)
	}
	if v.DecidingFlow != "" {
		t.Errorf("deciding flow = %q, want empty", v.DecidingFlow)
	}
}
