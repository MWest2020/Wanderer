// action_report_test.go exercises action-report.jq with the real jq
// binary (present in the build image) against a fixture Assessment,
// pinning task 3.1/3.2: a failing (afhankelijk) rationale that carries
// a `handeling` field surfaces it verbatim in the filter's `failing`
// array, and one with no `handeling` field surfaces a null the caller
// (action-report.sh) then falls back on.
package scripts

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func runJQFilter(t *testing.T, assessment map[string]any) map[string]any {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed")
	}
	input, err := json.Marshal(assessment)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	cmd := exec.Command("jq", "-f", filepath.Join(".", "action-report.jq"))
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("jq -f action-report.jq: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal jq output: %v (output: %s)", err, out)
	}
	return got
}

func TestActionReportJQ_FailingRuleShowsRealHandeling(t *testing.T) {
	got := runJQFilter(t, map[string]any{
		"dimensions": []map[string]any{
			{
				"dimension":    "juridisch",
				"score":        "afhankelijk",
				"completeness": "complete",
				"rationale": []map[string]any{
					{
						"criterium_id": "wand.juridisch.registrar_jurisdiction",
						"score":        "afhankelijk",
						"verdict":      "registrant outside EEA",
						"evidence":     []string{"f1"},
						"handeling":    "Verhuis de domeinregistratie van {domein} naar een registrar die in de EER is gevestigd.",
					},
				},
			},
		},
	})
	failing, ok := got["failing"].([]any)
	if !ok || len(failing) != 1 {
		t.Fatalf("want 1 failing entry, got %#v", got["failing"])
	}
	row := failing[0].(map[string]any)
	handeling, _ := row["handeling"].(string)
	if handeling == "" {
		t.Fatalf("want the real handeling text, got empty/null: %#v", row)
	}
	if handeling != "Verhuis de domeinregistratie van {domein} naar een registrar die in de EER is gevestigd." {
		t.Errorf("want the verbatim handeling text passed through, got %q", handeling)
	}
}

func TestActionReportJQ_FailingRuleWithoutHandelingIsNull(t *testing.T) {
	got := runJQFilter(t, map[string]any{
		"dimensions": []map[string]any{
			{
				"dimension":    "juridisch",
				"score":        "afhankelijk",
				"completeness": "complete",
				"rationale": []map[string]any{
					{
						"criterium_id": "wand.juridisch.some_rule",
						"score":        "afhankelijk",
						"verdict":      "no handeling on this one",
						"evidence":     []string{"f1"},
					},
				},
			},
		},
	})
	failing, ok := got["failing"].([]any)
	if !ok || len(failing) != 1 {
		t.Fatalf("want 1 failing entry, got %#v", got["failing"])
	}
	row := failing[0].(map[string]any)
	if row["handeling"] != nil {
		t.Errorf("want handeling null when the field is absent, got %#v", row["handeling"])
	}
}
