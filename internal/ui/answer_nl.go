package ui

import (
	_ "embed"
	"fmt"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed answer_nl.yaml
var answerNLYAML []byte

type answerNLFile struct {
	Outcomes map[string]string `yaml:"outcomes"`
}

var (
	answerNLOnce sync.Once
	answerNLMap  map[string]string
	answerNLErr  error
)

func loadAnswerNL() (map[string]string, error) {
	answerNLOnce.Do(func() {
		var f answerNLFile
		if err := yaml.UnmarshalStrict(answerNLYAML, &f); err != nil {
			answerNLErr = fmt.Errorf("ui: parse answer_nl.yaml: %w", err)
			return
		}
		answerNLMap = f.Outcomes
	})
	return answerNLMap, answerNLErr
}

// renderAnswerCopy looks up the Dutch template for outcome key and
// fills its {param} placeholders via fillParams (accountability.go).
// A missing key or a load error yields an empty string rather than
// panicking — answer_nl_test.go's completeness test is what catches a
// broken copy table, not a runtime panic on the answer page.
func renderAnswerCopy(key string, params map[string]string) string {
	m, err := loadAnswerNL()
	if err != nil {
		return ""
	}
	tmpl, ok := m[key]
	if !ok {
		return ""
	}
	return fillParams(tmpl, params)
}
