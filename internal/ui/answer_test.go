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
	v := BuildAnswerVerdict(assessments)
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

func TestBuildAnswerVerdict_AfhankelijkIsNeeNamingTheFlow(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
	)
	v := BuildAnswerVerdict(assessments)
	if v.Verdict != "nee" {
		t.Fatalf("verdict = %q, want nee", v.Verdict)
	}
	if v.DecidingFlow != "Mail" {
		t.Errorf("deciding flow = %q, want Mail", v.DecidingFlow)
	}
	if v.DecidingVerdict != "mx hosts in US (outside EEA)" {
		t.Errorf("deciding verdict = %q", v.DecidingVerdict)
	}
	if !strings.Contains(v.Headline, "Mail") || !strings.Contains(v.Headline, "mx hosts in US") {
		t.Errorf("headline %q does not name the deciding flow", v.Headline)
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
	v := BuildAnswerVerdict(assessments)
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
	v := BuildAnswerVerdict(assessments)
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
	v := BuildAnswerVerdict(assessments)
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
	v := BuildAnswerVerdict(nil)
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
	v := BuildAnswerVerdict(assessments)
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
