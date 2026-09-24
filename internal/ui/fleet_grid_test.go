package ui

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

// TestGridCellsForDomain_TitleCarriesFlowAndVerdict pins spec.md's
// scenario "Elk beeld heeft zijn woorden": "elke rastercel draagt de
// stroom en het oordeel als titel".
func TestGridCellsForDomain_TitleCarriesFlowAndVerdict(t *testing.T) {
	a := models.Assessment{Framework: "wand", Dimensions: []models.DimensionScore{{
		Rationale: []models.Rationale{
			rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
			rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in US", models.ScoreAfhankelijk),
		},
	}}}
	cells := gridCellsForDomain([]models.Assessment{a})
	if len(cells) != len(flowRules) {
		t.Fatalf("cells = %d, want %d (one per flow, fixed order)", len(cells), len(flowRules))
	}
	byFlow := map[string]GridCell{}
	for _, c := range cells {
		byFlow[c.FlowLabel] = c
	}
	hosting := byFlow["Hosting"]
	if hosting.Class != "score-soeverein" || hosting.Title != "Hosting: soeverein" || hosting.Label != "S" {
		t.Errorf("Hosting cell = %+v", hosting)
	}
	mail := byFlow["Mail"]
	if mail.Class != "score-afhankelijk" || mail.Title != "Mail: niet soeverein" || mail.Label != "N" {
		t.Errorf("Mail cell = %+v", mail)
	}
}

// TestGridCellsForDomain_MissingFlowReadsAsNietGemeten covers a domain
// whose rule never fired (including one with no scan at all): the cell
// must read distinctly from a genuine "onbekend" verdict in its title,
// even though it shares the same grey class.
func TestGridCellsForDomain_MissingFlowReadsAsNietGemeten(t *testing.T) {
	cells := gridCellsForDomain(nil)
	for _, c := range cells {
		if c.Class != "score-onbekend" {
			t.Errorf("cell %+v: want class score-onbekend for a domain with no data", c)
		}
		if want := c.FlowLabel + ": niet gemeten"; c.Title != want {
			t.Errorf("cell title = %q, want %q", c.Title, want)
		}
	}
}

// TestFlowColumnLabels_MatchesFlowRulesOrder pins that the grid's header
// row and every domain row's cells share exactly one column order.
func TestFlowColumnLabels_MatchesFlowRulesOrder(t *testing.T) {
	cols := flowColumnLabels()
	if len(cols) != len(flowRules) {
		t.Fatalf("len = %d, want %d", len(cols), len(flowRules))
	}
	for i, fr := range flowRules {
		if cols[i] != fr.label {
			t.Errorf("cols[%d] = %q, want %q", i, cols[i], fr.label)
		}
	}
}
