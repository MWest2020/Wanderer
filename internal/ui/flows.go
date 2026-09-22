package ui

import (
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Flow is one row of the Sovereignty overview — a single "what goes
// where" statement synthesised from an already-scored rule. Label is
// the flow category (Hosting, Mail, …); Verdict is the rule's observed
// statement (e.g. "mx hosts in US (outside EEA)"); Score drives the
// colour pill.
type Flow struct {
	Label   string
	Verdict string
	Score   string
}

// flowRule maps a wand rule ID to its flow label + display order, plus
// the plain-language Dutch question the reasoning page asks for it
// (spec.md "The reasoning is one click from the answer": each flow "a
// plain-language question with ja / nee / onbekend / n.v.t."). The
// overview is pure presentation: it re-groups signals the rule pack
// already produced into the org/host-as-the-spil-in-the-web picture
// ("your service lives here, its mail goes there, its DNS is run by …").
var flowRules = []struct {
	id, label, question string
}{
	{"wand.juridisch.apex_ip_eea", "Hosting", "Waar staat de hosting?"},
	{"wand.juridisch.mx_vendor_jurisdiction", "Mail", "Waar loopt de mail?"},
	{"wand.juridisch.ns_vendor_jurisdiction", "DNS", "Wie beheert de DNS?"},
	{"wand.juridisch.cert_issuer_eea", "Certificate", "Waar is het certificaat uitgegeven?"},
	{"wand.transit.eu_path", "Transit path", "Blijft het netwerkverkeer binnen de EER?"},
	{"wand.technologie.no_us_hyperscaler", "CDN / hyperscaler", "Zit er een Amerikaanse hyperscaler tussen?"},
	{"wand.technologie.third_parties_eea", "Third parties", "Zijn de derde partijen in de EER gevestigd?"},
}

// isSovereigntyFlowRule reports whether ruleID is one of the seven
// flow rules above. The reasoning page renders these exclusively
// through BuildFlowAnswers; the generic per-dimension rationale table
// (assessmentHandler) skips them so the same rule ID does not appear
// twice on the page — once collapsed in the flow's evidence, once
// again in the open (spec.md "Rule IDs ... SHALL appear only inside
// the evidence").
func isSovereigntyFlowRule(ruleID string) bool {
	for _, fr := range flowRules {
		if fr.id == ruleID {
			return true
		}
	}
	return false
}

// SovereigntyFlows synthesises the overview from a scan's assessments.
// It reads the per-rule rationales (which already carry the observed
// verdict + score) and emits one Flow per known flow rule, in a fixed
// order. Rules that did not fire (no rationale) are omitted. No EEA
// logic lives here — the flows mirror what the rule pack already
// scored; only the verdict's *language* is translated here, via
// dutchFlowVerdict, from the Dutch answer-sheet table (spec.md "Eén
// taal per laag"). findingsByID resolves a rule's Evidence back to the
// Finding that produced it so the Dutch verdict can name the observed
// fact (dutchFlowVerdict's {feit}); it may be nil where no findings are
// available (e.g. cross-target fleet aggregation), in which case the
// verdict is still Dutch but names no specific fact.
func SovereigntyFlows(assessments []models.Assessment, findingsByID map[string]models.Finding) []Flow {
	// Index the latest rationale per rule ID across all frameworks.
	byRule := map[string]models.Rationale{}
	for _, a := range assessments {
		for _, d := range a.Dimensions {
			for _, r := range d.Rationale {
				byRule[r.CriteriumID] = r
			}
		}
	}
	var flows []Flow
	for _, fr := range flowRules {
		rat, ok := byRule[fr.id]
		if !ok {
			continue
		}
		flows = append(flows, Flow{
			Label:   fr.label,
			Verdict: dutchFlowVerdict(fr.id, rat.Score, flowFeit(fr.id, rat, findingsByID)),
			Score:   string(rat.Score),
		})
	}
	return flows
}

// dutchFlowVerdict renders a sovereignty-flow rule's outcome as the
// Dutch answer-sheet sentence for ruleID + score, from
// accountability_nl.yaml's rules table — never a raw fallback to the
// rule's own English Verdict (spec.md "Eén taal per laag": a shown
// verdict SHALL come from the table). feit is the pre-formatted
// observed-fact clause from flowFeit (either "" or " — <fact>"),
// spliced into the template's {feit} placeholder. Returns "" when
// ruleID or the outcome has no table entry, rather than fabricating
// text — task 4.3 guards this with
// TestDutchFlowVerdict_UnknownRuleNeverFallsBackToRawEnglish.
func dutchFlowVerdict(ruleID string, score models.Score, feit string) string {
	verdicts, ok := wand.FlowVerdictFor(ruleID)
	if !ok {
		return ""
	}
	outcome := string(score)
	if score == "" || score == models.ScoreOnbekend {
		outcome = "onbekend"
	}
	tmpl, ok := verdicts[outcome]
	if !ok {
		return ""
	}
	return fillParams(tmpl, map[string]string{"feit": feit})
}

// flowFeitLead is the Dutch noun phrase dutchFlowVerdict's {feit}
// leads with, per flow rule — "apex-adressen in NL" rather than a bare
// "NL" — so the fact reads as a complete clause on its own.
var flowFeitLead = map[string]string{
	"wand.juridisch.apex_ip_eea":            "apex-adressen in",
	"wand.juridisch.mx_vendor_jurisdiction": "mailservers in",
	"wand.juridisch.ns_vendor_jurisdiction": "nameservers in",
	"wand.juridisch.cert_issuer_eea":        "certificaat uitgegeven in",
	"wand.transit.eu_path":                  "bestemming in",
	"wand.technologie.no_us_hyperscaler":    "hyperscaler",
	"wand.technologie.third_parties_eea":    "derde partijen in",
}

// flowFeit reads the observed country/party behind a flow rule's
// rationale off the Finding(s) its Evidence points at — the same
// structured attributes the rule itself matched on (`country`,
// `organisation`, or `issuer_country` for the certificate rule) —
// rather than parsing them out of the rule's English Verdict sentence.
// Parsing the English sentence would either paste it whole (half-
// English, the bug run 04 introduced by dropping it entirely) or
// require regexing free-form prose, which this package's "boring and
// auditable" convention rules out. Returns "" when findingsByID is nil
// or none of the Evidence IDs resolve, so dutchFlowVerdict's {feit}
// degrades to an empty clause rather than a dangling " — ".
func flowFeit(ruleID string, rat models.Rationale, findingsByID map[string]models.Finding) string {
	var countries, parties []string
	seenC, seenP := map[string]bool{}, map[string]bool{}
	for _, id := range rat.Evidence {
		f, ok := findingsByID[id]
		if !ok {
			continue
		}
		if ruleID == "wand.juridisch.cert_issuer_eea" {
			for _, c := range flowStringsAttr(f.Attributes, "issuer_country") {
				c = strings.ToUpper(c)
				if c != "" && !seenC[c] {
					seenC[c] = true
					countries = append(countries, c)
				}
			}
			continue
		}
		if c, _ := f.Attributes["country"].(string); c != "" && !seenC[c] {
			seenC[c] = true
			countries = append(countries, c)
		}
		if org, _ := f.Attributes["organisation"].(string); org != "" && !seenP[org] {
			seenP[org] = true
			parties = append(parties, org)
		}
	}
	var fact string
	switch {
	case len(countries) > 0 && len(parties) > 0:
		fact = strings.Join(countries, ", ") + " (" + strings.Join(parties, ", ") + ")"
	case len(countries) > 0:
		fact = strings.Join(countries, ", ")
	case len(parties) > 0:
		fact = strings.Join(parties, ", ")
	default:
		return ""
	}
	if lead := flowFeitLead[ruleID]; lead != "" {
		return " — " + lead + " " + fact
	}
	return " — " + fact
}

// flowStringsAttr reads a list-shaped attribute, tolerant of the
// in-memory ([]string) and JSON-reloaded ([]any) shapes a Finding's
// Attributes can carry — mirrors wand.stringsFromAttr (unexported,
// duplicated here for this single UI caller).
func flowStringsAttr(attrs map[string]any, key string) []string {
	switch v := attrs[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{v}
	}
	return nil
}

// classifyFlows buckets a scan's sovereignty flows the same way
// BuildAnswerVerdict and BuildFleetScore both need to: which flows
// scored afhankelijk (in SovereigntyFlows' fixed order, so the first
// entry is always the same "heaviest" one), how many fired but scored
// onbekend, and how many landed soeverein/voldoende. Shared here so
// the two callers can't drift into counting "answered" differently.
func classifyFlows(flows []Flow) (afhankelijk []Flow, unanswered, answered int) {
	for _, f := range flows {
		switch models.Score(f.Score) {
		case models.ScoreAfhankelijk:
			afhankelijk = append(afhankelijk, f)
		case models.ScoreOnbekend:
			unanswered++
		default: // soeverein, voldoende
			answered++
		}
	}
	return afhankelijk, unanswered, answered
}

// FlowState is one flow's progress on the progressive answer page
// (spec.md "The answer fills in while the scan runs"): a flow whose
// rule has evidence reads as "beantwoord"; one without evidence reads
// as "bezig" while the scan can still produce it, or "niet_gemeten"
// once the scan is done and it plainly never fired.
type FlowState struct {
	Label   string
	State   string // "bezig" | "beantwoord" | "niet_gemeten"
	Verdict string
	Score   string
	// Remediation is the handeling for this flow, set only when Score
	// is "afhankelijk" — read straight from wand.HandelingFor with its
	// {domein} placeholder filled in, the same table BuildFlowAnswers
	// (flow_answers.go) and BuildAccountabilityAnswers use, so the
	// answer page names the same action the onderbouwing page does.
	Remediation string
}

// BuildFlowStates renders per-flow progress from a scan's assessments
// (typically computed on the fly from the findings persisted so far —
// see answerHandler). done is the scan's completion state: false while
// it is still running, true once no more findings will land. domain
// fills the {domein} placeholder in a niet-soeverein flow's handeling.
//
// A rule that has already run against the current findings always
// produces a Rationale entry, evidenced or not (assessor.Assess emits
// one per registered rule). So "the rule fired but had nothing to go
// on" and "the rule has not been reached yet" look identical in the
// Rationale — the only way to tell them apart is whether the scan
// could still produce more findings.
func BuildFlowStates(assessments []models.Assessment, done bool, domain string, findingsByID map[string]models.Finding) []FlowState {
	byRule := map[string]models.Rationale{}
	for _, a := range assessments {
		for _, d := range a.Dimensions {
			for _, r := range d.Rationale {
				byRule[r.CriteriumID] = r
			}
		}
	}
	out := make([]FlowState, 0, len(flowRules))
	for _, fr := range flowRules {
		r, ok := byRule[fr.id]
		evidenced := ok && len(r.Evidence) > 0
		switch {
		case evidenced:
			fs := FlowState{Label: fr.label, State: "beantwoord", Verdict: dutchFlowVerdict(fr.id, r.Score, flowFeit(fr.id, r, findingsByID)), Score: string(r.Score)}
			if r.Score == models.ScoreAfhankelijk {
				if h, ok := wand.HandelingFor(fr.id); ok {
					fs.Remediation = fillParams(h, map[string]string{"domein": domain})
				}
			}
			out = append(out, fs)
		case done:
			out = append(out, FlowState{Label: fr.label, State: "niet_gemeten"})
		default:
			out = append(out, FlowState{Label: fr.label, State: "bezig"})
		}
	}
	return out
}

// FlowRollup is one flow category aggregated across an organisation's
// targets: how many were assessed for it and how many landed
// afhankelijk (the actionable count), with the worst score reached for
// the pill colour. It turns the per-scan overview into the org-as-the-
// spider-in-the-web posture ("across your services, mail is the weak
// spot").
type FlowRollup struct {
	Label       string
	Total       int
	Afhankelijk int
	Worst       string
}

// SovereigntyFlowRollup aggregates the per-target flows across a set of
// snapshots (an organisation's latest scans) into one row per flow
// category. Categories no target was assessed for are omitted.
func SovereigntyFlowRollup(snaps []TargetSnapshot) []FlowRollup {
	type acc struct {
		total, afh int
		worst      models.Score
	}
	byLabel := map[string]*acc{}
	for _, s := range snaps {
		assessments := make([]models.Assessment, 0, len(s.Assessments))
		for _, a := range s.Assessments {
			assessments = append(assessments, a)
		}
		for _, f := range SovereigntyFlows(assessments, nil) {
			a := byLabel[f.Label]
			if a == nil {
				a = &acc{}
				byLabel[f.Label] = a
			}
			a.total++
			score := models.Score(f.Score)
			if score == models.ScoreAfhankelijk {
				a.afh++
			}
			if worseScore(score, a.worst) {
				a.worst = score
			}
		}
	}
	var out []FlowRollup
	for _, fr := range flowRules { // fixed order
		a := byLabel[fr.label]
		if a == nil {
			continue
		}
		out = append(out, FlowRollup{
			Label:       fr.label,
			Total:       a.total,
			Afhankelijk: a.afh,
			Worst:       string(a.worst),
		})
	}
	return out
}

// worseScore reports whether candidate is a worse (less sovereign)
// score than current, treating onbekend as the mildest so a genuine
// afhankelijk/voldoende always wins the pill. Empty current loses.
func worseScore(candidate, current models.Score) bool {
	return scoreSeverity(candidate) > scoreSeverity(current)
}

func scoreSeverity(s models.Score) int {
	switch s {
	case models.ScoreAfhankelijk:
		return 3
	case models.ScoreVoldoende:
		return 2
	case models.ScoreSoeverein:
		return 1
	default: // onbekend / empty
		return 0
	}
}
