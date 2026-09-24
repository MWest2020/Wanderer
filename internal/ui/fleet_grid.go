package ui

import "github.com/MWest2020/wanderer/pkg/models"

// GridCell is one coloured cell of the door's domain × stroom raster
// (proposal.md "Domains become a grid: one row per domain, one coloured
// cell per flow"). Label is a single letter/symbol so the cell reads on
// screen and in print without colour; Title carries the same fact as
// text (design.md "each grid cell a title with flow and verdict") —
// spec.md's scenario "Elk beeld heeft zijn woorden" asserts on Title,
// not on Class.
type GridCell struct {
	FlowLabel string
	Label     string
	Class     string
	Title     string
}

// gridScoreWord is the plain-language verdict a grid cell's Title
// carries — the flow's outcome, never its onderbouwing (design.md "The
// top 3 without the rationale" applies here too: this raster is the
// same vlootlaag, no English rationale, no long sentence).
func gridScoreWord(s models.Score) string {
	switch s {
	case models.ScoreSoeverein:
		return "soeverein"
	case models.ScoreVoldoende:
		return "voldoende"
	case models.ScoreAfhankelijk:
		return "niet soeverein"
	default:
		return "onbekend"
	}
}

// gridScoreLetter is the cell's screen/print-safe symbol — the fact the
// colour also carries, spelled out for a reader who cannot see colour.
func gridScoreLetter(s models.Score) string {
	switch s {
	case models.ScoreSoeverein:
		return "S"
	case models.ScoreVoldoende:
		return "V"
	case models.ScoreAfhankelijk:
		return "N"
	default:
		return "?"
	}
}

// flowColumnLabels is the domain grid's fixed header row — flowRules'
// labels in the same order gridCellsForDomain emits cells.
func flowColumnLabels() []string {
	out := make([]string, len(flowRules))
	for i, fr := range flowRules {
		out[i] = fr.label
	}
	return out
}

// gridCellsForDomain renders one domain's row of the raster: one cell
// per flow rule, in flowRules' fixed order, so every row (and the header
// above it) lines up column for column. A flow whose rule never fired
// for this domain (no rationale at all — including a domain with no
// scan yet) reads as "niet gemeten", visually identical to onbekend but
// named distinctly in its Title so the two are never confused in the
// text a screen reader hears.
func gridCellsForDomain(assessments []models.Assessment) []GridCell {
	byLabel := map[string]Flow{}
	for _, f := range SovereigntyFlows(assessments, nil) {
		byLabel[f.Label] = f
	}
	cells := make([]GridCell, 0, len(flowRules))
	for _, fr := range flowRules {
		f, ok := byLabel[fr.label]
		if !ok {
			cells = append(cells, GridCell{
				FlowLabel: fr.label,
				Label:     "–",
				Class:     "score-onbekend",
				Title:     fr.label + ": niet gemeten",
			})
			continue
		}
		score := models.Score(f.Score)
		cells = append(cells, GridCell{
			FlowLabel: fr.label,
			Label:     gridScoreLetter(score),
			Class:     "score-" + string(score),
			Title:     fr.label + ": " + gridScoreWord(score),
		})
	}
	return cells
}
