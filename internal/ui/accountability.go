// accountability.go builds the accountability dimension's "answer
// sheet" (design.md "UI direction", run 08 task 8.1): one
// plain-language question per wand.accountability.* rule, filled from
// internal/assessor/wand/accountability_nl.yaml. The dimension's two
// operationeel-scored siblings (domain_expiry, variant_convergence)
// share that same copy table but keep the pre-existing generic
// dimension-card rendering — this run's scope is the accountability
// dimension only (run 08 task-ref, "Accountability section").
package ui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/pkg/models"
)

// accountabilityRuleOrder is the answer sheet's fixed display order —
// the five wand.accountability.* rules, independent of Rationale
// slice ordering (which sorts by full rule ID and would interleave
// fine here anyway, but an explicit order keeps the sheet stable if a
// rule is ever renamed).
var accountabilityRuleOrder = []string{
	"wand.accountability.registrant_identifiable",
	"wand.accountability.no_reseller",
	"wand.accountability.soa_rname",
	"wand.accountability.securitytxt",
	"wand.accountability.ns_holder_transparent",
}

// AccountabilityAnswer is one rendered answer-sheet row.
type AccountabilityAnswer struct {
	RuleID          string
	Question        string
	AnswerLabel     string // "Ja" | "Nee" | "Onbekend" | "n.v.t."
	AnswerClass     string // "ja" | "nee" | "onbekend" | "nvt" — CSS + test hook
	Verdict         string
	Remediation     string // set only when AnswerClass == "nee"
	RuleDescription string
	Evidence        []AccountabilityEvidence
	NoEvidenceNote  string // set when Evidence is empty
}

// AccountabilityEvidence is one finding backing an answer, expanded
// in the report's <details> block — technical keys stay here, never
// in the headline (spec.md "Rule IDs, RDAP field names, and RFC
// numbers SHALL appear only inside the expandable evidence block").
type AccountabilityEvidence struct {
	ProbeID    string
	Subject    string
	Attributes []AccountabilityAttr
}

// AccountabilityAttr is one labelled attribute inside an evidence
// block: the raw attribute key doubles as its own technical label.
type AccountabilityAttr struct {
	Key   string
	Value string
}

// BuildAccountabilityAnswers renders the accountability dimension's
// five rules into answer-sheet rows. findingsByID resolves each
// Rationale's Evidence IDs back to the Finding that produced it;
// domain is the scanned domain, needed only to resolve the TLD for
// the registrant_identifiable n.v.t. case (registry redaction can
// fire before any RDAP evidence exists).
func BuildAccountabilityAnswers(dim models.DimensionScore, findingsByID map[string]models.Finding, domain string) []AccountabilityAnswer {
	byRule := map[string]models.Rationale{}
	for _, r := range dim.Rationale {
		byRule[r.CriteriumID] = r
	}
	var out []AccountabilityAnswer
	for _, ruleID := range accountabilityRuleOrder {
		rat, ok := byRule[ruleID]
		if !ok {
			continue
		}
		copyEntry, ok := wand.AccountabilityCopyFor(ruleID)
		if !ok {
			continue
		}
		outcome, label, class := accountabilityOutcome(rat)
		params := accountabilityParams(ruleID, rat, findingsByID, domain)
		row := AccountabilityAnswer{
			RuleID:      ruleID,
			Question:    copyEntry.Question,
			AnswerLabel: label,
			AnswerClass: class,
			Verdict:     fillParams(copyEntry.Verdicts[outcome], params),
		}
		if rule, ok := lookupRule("wand", ruleID); ok {
			row.RuleDescription = rule.Description
		}
		if class == "nee" {
			row.Remediation = fillParams(copyEntry.Remediation["afhankelijk"], params)
		}
		for _, id := range rat.Evidence {
			f, ok := findingsByID[id]
			if !ok {
				continue
			}
			row.Evidence = append(row.Evidence, accountabilityEvidenceFor(f))
		}
		if len(row.Evidence) == 0 {
			row.NoEvidenceNote = rat.Verdict
		}
		out = append(out, row)
	}
	return out
}

// accountabilityOutcome maps a Rationale onto the four-value answer
// scale the report SHALL use (spec.md "ja / nee / onbekend / n.v.t."),
// distinct from the six-value models.Score/reason-class combination
// underneath it. A structural reason (registry redaction) is n.v.t.;
// any other reason, or a rule that never got evidence, reads as
// onbekend — never dressed up as a defect. soeverein and voldoende
// both read "Ja": neither carries a remediation line in
// accountability_nl.yaml (only afhankelijk and onbekend do), so
// voldoende's nuance lives in the verdict sentence, not the badge.
func accountabilityOutcome(rat models.Rationale) (outcomeKey, label, class string) {
	if rat.Reason != "" {
		if class, _ := assessor.ReasonInfo(rat.Reason); class == assessor.ReasonStructural {
			return "structural", "n.v.t.", "nvt"
		}
		return "onbekend", "Onbekend", "onbekend"
	}
	switch rat.Score {
	case models.ScoreSoeverein, models.ScoreVoldoende:
		return string(rat.Score), "Ja", "ja"
	case models.ScoreAfhankelijk:
		return string(rat.Score), "Nee", "nee"
	default:
		return "onbekend", "Onbekend", "onbekend"
	}
}

// accountabilityParamRe matches the {name} placeholders
// accountability_nl.yaml uses for named parameters.
var accountabilityParamRe = regexp.MustCompile(`\{([a-zA-Z_]+)\}`)

// fillParams substitutes {key} placeholders with params[key], leaving
// an unmatched placeholder as-is rather than blanking it — a missing
// param should be loud (visible in the rendered page), not silently
// swallowed.
func fillParams(tmpl string, params map[string]string) string {
	if tmpl == "" {
		return ""
	}
	return accountabilityParamRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := m[1 : len(m)-1]
		if v, ok := params[key]; ok {
			return v
		}
		return m
	})
}

// accountabilityParams re-derives the named parameters
// accountability_nl.yaml's copy for ruleID needs, reading them off
// the same Finding attributes the rule itself matched on (the
// Rationale only carries the rule's own English Verdict sentence and
// the Finding IDs, not structured params — this run makes no assessor
// changes, so the UI re-extracts rather than the rule exporting them).
func accountabilityParams(ruleID string, rat models.Rationale, findingsByID map[string]models.Finding, domain string) map[string]string {
	params := map[string]string{}
	firstEvidence := func() (models.Finding, bool) {
		for _, id := range rat.Evidence {
			if f, ok := findingsByID[id]; ok {
				return f, true
			}
		}
		return models.Finding{}, false
	}

	switch ruleID {
	case "wand.accountability.registrant_identifiable":
		if rat.Reason == assessor.ReasonRegistryRedacted {
			tld := accountabilityTLD(domain)
			params["tld"] = tld
			if registry, ok := wand.RegistryRedactionFor(tld); ok {
				params["registry"] = registry
			}
			return params
		}
		f, ok := firstEvidence()
		if !ok {
			return params
		}
		name := strings.TrimSpace(accountabilityStringAttr(f.Attributes, "name"))
		params["registrant"] = name
		if proxy, ok := wand.MatchPrivacyProxy(name); ok {
			params["proxy"] = proxy
		} else if rat.Score == models.ScoreAfhankelijk {
			// RDAP published no registrant entity at all — afhankelijk
			// but not a named proxy. {proxy} still needs a value; the
			// copy table has one afhankelijk sentence for both cases.
			params["proxy"] = "een niet-gepubliceerde partij"
		}
	case "wand.accountability.no_reseller":
		if f, ok := firstEvidence(); ok {
			params["reseller"] = accountabilityStringAttr(f.Attributes, "name")
		}
	case "wand.accountability.soa_rname":
		if f, ok := firstEvidence(); ok {
			params["mailbox"] = accountabilityStringAttr(f.Attributes, "mailbox_domain")
		}
	case "wand.accountability.securitytxt":
		if f, ok := firstEvidence(); ok {
			raw := accountabilityStringAttr(f.Attributes, "expires")
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				params["date"] = t.UTC().Format("2006-01-02")
			} else {
				params["date"] = "onbekende datum"
			}
		}
	case "wand.accountability.ns_holder_transparent":
		total, identifiable := 0, 0
		var opaque []string
		for _, id := range rat.Evidence {
			f, ok := findingsByID[id]
			if !ok {
				continue
			}
			total++
			if accountabilityStringAttr(f.Attributes, "registrant") == "present" {
				identifiable++
			} else {
				opaque = append(opaque, f.Subject)
			}
		}
		params["total"] = fmt.Sprintf("%d", total)
		params["count"] = fmt.Sprintf("%d", identifiable)
		params["domains"] = strings.Join(accountabilityDedupe(opaque), ", ")
	}
	return params
}

func accountabilityEvidenceFor(f models.Finding) AccountabilityEvidence {
	ev := AccountabilityEvidence{ProbeID: f.ProbeID, Subject: f.Subject}
	keys := make([]string, 0, len(f.Attributes))
	for k := range f.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ev.Attributes = append(ev.Attributes, AccountabilityAttr{Key: k, Value: fmt.Sprintf("%v", f.Attributes[k])})
	}
	return ev
}

// accountabilityStringAttr reads a string attribute, tolerant of a
// missing map or key — mirrors wand.stringFromAttr, duplicated here
// (unexported in wand) rather than exported for a single UI caller.
func accountabilityStringAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	s, _ := attrs[key].(string)
	return s
}

func accountabilityDedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// accountabilityTLD returns the last label of domain, lower-cased, or
// "" when domain has no dot — mirrors wand.tldOf (unexported).
func accountabilityTLD(domain string) string {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	i := strings.LastIndex(domain, ".")
	if i < 0 || i == len(domain)-1 {
		return ""
	}
	return domain[i+1:]
}
