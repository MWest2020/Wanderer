package wand

import (
	"regexp"
	"testing"
)

// accountabilityRuleParams declares, per rule ID, the named parameters
// that rule is allowed to supply to its accountability_nl.yaml copy.
// The completeness test below fails when an entry names a parameter
// outside this set — task 7.3's "an entry names a parameter the rule
// does not supply".
var accountabilityRuleParams = map[string][]string{
	"wand.accountability.registrant_identifiable": {"registrant", "proxy", "registry", "tld"},
	"wand.accountability.no_reseller":              {"reseller"},
	"wand.accountability.soa_rname":                {"mailbox"},
	"wand.accountability.securitytxt":              {"date"},
	"wand.accountability.ns_holder_transparent":    {"count", "total", "domains"},
	"wand.operationeel.domain_expiry":              {"date"},
	"wand.operationeel.variant_convergence":        {"count", "total", "path"},
}

// flowVerdictRuleIDs are the seven sovereignty-flow rules (run 04 task
// 4.1): unlike accountabilityRuleParams' rules, they carry no Question
// or Remediation in this table (the question lives in
// internal/ui/flows.go's flowRules, the remediation in this file's
// handelingen map) — just a fixed, parameter-free Dutch verdict per
// score, checked by TestAccountabilityNL_FlowVerdictsComplete.
var flowVerdictRuleIDs = []string{
	"wand.juridisch.apex_ip_eea",
	"wand.juridisch.mx_vendor_jurisdiction",
	"wand.juridisch.ns_vendor_jurisdiction",
	"wand.juridisch.cert_issuer_eea",
	"wand.transit.eu_path",
	"wand.technologie.no_us_hyperscaler",
	"wand.technologie.third_parties_eea",
}

// TestAccountabilityNL_FlowVerdictsComplete pins that every flow rule
// has a non-empty, parameter-free verdict for all four base outcomes —
// dutchFlowVerdict (internal/ui/flows.go) looks these up by rule ID +
// score alone, so a missing outcome would silently render a blank
// verdict line instead of failing loudly.
func TestAccountabilityNL_FlowVerdictsComplete(t *testing.T) {
	for _, ruleID := range flowVerdictRuleIDs {
		ruleID := ruleID
		t.Run(ruleID, func(t *testing.T) {
			entry, ok := AccountabilityCopyFor(ruleID)
			if !ok {
				t.Fatalf("no accountability_nl.yaml entry for %s", ruleID)
			}
			for _, outcome := range baseOutcomes {
				text, ok := entry.Verdicts[outcome]
				if !ok || text == "" {
					t.Errorf("%s: missing verdict text for outcome %q", ruleID, outcome)
					continue
				}
				checkParams(t, ruleID, "verdicts["+outcome+"]", text, nil)
			}
		})
	}
}

// baseOutcomes are the four models.Score values every rule entry must
// carry verdict text for, regardless of whether every branch of the
// rule's own Go logic currently reaches that outcome — the copy table
// is a complete answer sheet, not a mirror of today's code paths.
var baseOutcomes = []string{"soeverein", "voldoende", "afhankelijk", "onbekend"}

// failingOutcomes are the outcomes that need a remediation line. n.v.t.
// ("structural") is deliberately excluded: it is not a failure to fix.
var failingOutcomes = []string{"afhankelijk", "onbekend"}

var namedParamRe = regexp.MustCompile(`\{([a-zA-Z_]+)\}`)

// TestAccountabilityNL_Completeness is the load-time test task 7.3
// requires: it fails when a rule has no accountability_nl.yaml entry,
// an outcome is missing (question, a base verdict, or a remediation
// line for a failing outcome), or an entry names a parameter the rule
// does not supply.
func TestAccountabilityNL_Completeness(t *testing.T) {
	for ruleID, params := range accountabilityRuleParams {
		ruleID := ruleID
		allowed := map[string]bool{}
		for _, p := range params {
			allowed[p] = true
		}

		t.Run(ruleID, func(t *testing.T) {
			entry, ok := AccountabilityCopyFor(ruleID)
			if !ok {
				t.Fatalf("no accountability_nl.yaml entry for %s", ruleID)
			}
			if entry.Question == "" {
				t.Errorf("%s: empty question", ruleID)
			}
			checkParams(t, ruleID, "question", entry.Question, allowed)

			for _, outcome := range baseOutcomes {
				text, ok := entry.Verdicts[outcome]
				if !ok || text == "" {
					t.Errorf("%s: missing verdict text for outcome %q", ruleID, outcome)
					continue
				}
				checkParams(t, ruleID, "verdicts["+outcome+"]", text, allowed)
			}
			if text, ok := entry.Verdicts["structural"]; ok {
				checkParams(t, ruleID, `verdicts["structural"]`, text, allowed)
			}

			for _, outcome := range failingOutcomes {
				text, ok := entry.Remediation[outcome]
				if !ok || text == "" {
					t.Errorf("%s: missing remediation line for failing outcome %q", ruleID, outcome)
					continue
				}
				checkParams(t, ruleID, "remediation["+outcome+"]", text, allowed)
			}
		})
	}
}

// TestAccountabilityNL_NoOrphanEntries pins that every entry in the
// YAML file corresponds to a rule this test suite knows about — an
// entry for a rule ID that no longer exists (or was renamed) would
// otherwise go silently unchecked.
func TestAccountabilityNL_NoOrphanEntries(t *testing.T) {
	m, err := loadAccountabilityNL()
	if err != nil {
		t.Fatalf("load accountability_nl.yaml: %v", err)
	}
	flowIDs := map[string]bool{}
	for _, id := range flowVerdictRuleIDs {
		flowIDs[id] = true
	}
	for ruleID := range m {
		if _, ok := accountabilityRuleParams[ruleID]; ok {
			continue
		}
		if flowIDs[ruleID] {
			continue
		}
		t.Errorf("accountability_nl.yaml has an entry for unknown rule %q", ruleID)
	}
}

func checkParams(t *testing.T, ruleID, where, text string, allowed map[string]bool) {
	t.Helper()
	for _, m := range namedParamRe.FindAllStringSubmatch(text, -1) {
		if !allowed[m[1]] {
			t.Errorf("%s %s: names undeclared parameter {%s}: %q", ruleID, where, m[1], text)
		}
	}
}
