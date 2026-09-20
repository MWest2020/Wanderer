package wand

import (
	_ "embed"
	"fmt"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed accountability_nl.yaml
var accountabilityNLYAML []byte

// AccountabilityCopy is the Dutch answer-sheet copy for one
// accountability-change rule: the plain-language question, a verdict
// string per outcome (keyed by models.Score value, plus "structural"
// for the n.v.t. case), and a remediation line per failing outcome.
type AccountabilityCopy struct {
	Question    string            `yaml:"question"`
	Verdicts    map[string]string `yaml:"verdicts"`
	Remediation map[string]string `yaml:"remediation"`
}

type accountabilityNLFile struct {
	Rules map[string]AccountabilityCopy `yaml:"rules"`
	// Handelingen is the generic, {domein}-parametrised remediation
	// line every wand rule carries (run 05 task 5.2/5.3) — distinct
	// from Rules[...].Remediation, which only exists for the seven
	// answer-sheet rules and carries richer, evidence-derived params.
	Handelingen map[string]string `yaml:"handelingen"`
}

var (
	accountabilityNLOnce        sync.Once
	accountabilityNLMap         map[string]AccountabilityCopy
	accountabilityNLHandelingen map[string]string
	accountabilityNLErr         error
)

func loadAccountabilityNL() (map[string]AccountabilityCopy, error) {
	accountabilityNLOnce.Do(func() {
		var f accountabilityNLFile
		if err := yaml.UnmarshalStrict(accountabilityNLYAML, &f); err != nil {
			accountabilityNLErr = fmt.Errorf("wand: parse accountability_nl.yaml: %w", err)
			return
		}
		accountabilityNLMap = f.Rules
		accountabilityNLHandelingen = f.Handelingen
	})
	return accountabilityNLMap, accountabilityNLErr
}

// AccountabilityCopyFor returns the Dutch copy entry for ruleID (a
// full rule ID such as "wand.accountability.registrant_identifiable").
func AccountabilityCopyFor(ruleID string) (AccountabilityCopy, bool) {
	m, err := loadAccountabilityNL()
	if err != nil {
		return AccountabilityCopy{}, false
	}
	c, ok := m[ruleID]
	return c, ok
}

// HandelingFor returns the Dutch remediation sentence for ruleID — one
// concrete action for the case the rule scores afhankelijk, with a
// {domein} placeholder the caller fills in. ok is false when ruleID
// has no entry in accountability_nl.yaml's handelingen table.
func HandelingFor(ruleID string) (string, bool) {
	if _, err := loadAccountabilityNL(); err != nil {
		return "", false
	}
	h, ok := accountabilityNLHandelingen[ruleID]
	return h, ok
}
