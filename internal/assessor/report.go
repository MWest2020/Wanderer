package assessor

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Rules provides the description lookup the report renderers need to
// print a human-readable rule title next to each Rationale entry.
// Callers pass the same rule set they passed to Assess.
type Rules []Rule

func (r Rules) description(id string) string {
	for _, rule := range r {
		if rule.ID == id {
			return rule.Description
		}
	}
	return ""
}

// RenderMarkdown writes a Markdown report for a to w. It is deterministic
// for a given Assessment and rule set.
func RenderMarkdown(w io.Writer, a *models.Assessment, rules Rules, subject string) error {
	b := &strings.Builder{}
	if subject == "" {
		subject = a.ScanID
	}
	fmt.Fprintf(b, "# Wanderer Assessment — %s\n\n", subject)
	fmt.Fprintf(b, "Scan: %s\n", a.ScanID)
	fmt.Fprintf(b, "Generated: %s\n", a.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(b, "Framework: %s\n\n", a.Framework)

	b.WriteString("## Samenvatting\n\n")
	b.WriteString("| Dimensie | Score | Volledigheid |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, d := range a.Dimensions {
		fmt.Fprintf(b, "| %s | %s | %s |\n", d.Dimension, d.Score, completenessLabel(d))
	}
	b.WriteString("\n")

	for _, d := range a.Dimensions {
		fmt.Fprintf(b, "## %s — %s (%s)\n\n", d.Dimension, d.Score, completenessLabel(d))
		if len(d.Rationale) == 0 {
			b.WriteString("_Geen regels beschikbaar voor deze dimensie._\n\n")
			continue
		}
		for _, r := range d.Rationale {
			desc := rules.description(r.CriteriumID)
			if desc == "" {
				desc = r.CriteriumID
			}
			fmt.Fprintf(b, "### %s — %s\n", r.CriteriumID, r.Score)
			fmt.Fprintf(b, "_%s_\n\n", desc)
			if r.Verdict != "" {
				fmt.Fprintf(b, "Verdict: %s\n\n", r.Verdict)
			}
			if len(r.Evidence) == 0 {
				b.WriteString("Evidence: _(none — rule had no matching findings)_\n\n")
			} else {
				fmt.Fprintf(b, "Evidence: %s\n\n", strings.Join(r.Evidence, ", "))
			}
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// RenderText writes a compact plain-text summary for a terminal.
func RenderText(w io.Writer, a *models.Assessment, _ Rules, subject string) error {
	b := &strings.Builder{}
	if subject == "" {
		subject = a.ScanID
	}
	fmt.Fprintf(b, "Wanderer Assessment — %s\n", subject)
	fmt.Fprintf(b, "Scan: %s  Framework: %s\n", a.ScanID, a.Framework)
	fmt.Fprintf(b, "Generated: %s\n\n", a.CreatedAt.UTC().Format(time.RFC3339))
	for _, d := range a.Dimensions {
		fmt.Fprintf(b, "[%s] %-12s volledigheid=%s\n", d.Score, d.Dimension, completenessLabel(d))
		// Stable rationale order by criterium id (already enforced by engine).
		rs := append([]models.Rationale(nil), d.Rationale...)
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].CriteriumID < rs[j].CriteriumID })
		for _, r := range rs {
			fmt.Fprintf(b, "    %-10s %s  %s\n", r.Score, r.CriteriumID, r.Verdict)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderJSON writes the Assessment as indented JSON.
func RenderJSON(w io.Writer, a *models.Assessment) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(a)
}

// completenessLabel maps an incomplete dimension with zero rationale
// to "n/a" so reports do not claim "incomplete" for dimensions the
// rule set does not address at all. For every other state it passes
// the completeness value through unchanged.
func completenessLabel(d models.DimensionScore) string {
	if d.Completeness == models.CompletenessIncomplete && len(d.Rationale) == 0 {
		return "n/a"
	}
	return string(d.Completeness)
}
