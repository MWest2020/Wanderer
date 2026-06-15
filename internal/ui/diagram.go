package ui

import "math"

// Diagram is a server-rendered, no-JS hub-and-spoke layout of the
// sovereignty flows — the target at the centre, each flow a spoke to a
// node coloured by its score. It is computed from the same Flow model
// the overview uses; the template emits it as inline SVG (the no-JS
// baseline). A progressive-enhancement JS layer can later make it
// interactive without changing this server-side structure.
type Diagram struct {
	Width, Height int
	CenterX       int
	CenterY       int
	Subject       string
	Nodes         []DiagramNode
}

// DiagramNode is one flow rendered as a spoke endpoint. X/Y are the
// node centre; LabelX/LabelY/Anchor place the text clear of the hub.
type DiagramNode struct {
	X, Y   int
	LabelX int
	LabelY int
	Anchor string // SVG text-anchor: start | middle | end
	Label  string
	Score  string
}

const (
	diagramSize   = 440
	diagramRadius = 150
)

// SovereigntyDiagram lays the flows out evenly around a circle. With
// the fixed flow set (≤6) the spokes never overlap. An empty flow list
// yields a zero-node diagram (the template omits it).
func SovereigntyDiagram(subject string, flows []Flow) Diagram {
	d := Diagram{
		Width:   diagramSize,
		Height:  diagramSize,
		CenterX: diagramSize / 2,
		CenterY: diagramSize / 2,
		Subject: subject,
	}
	n := len(flows)
	for i, f := range flows {
		// Start at the top (−90°) and go clockwise.
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
		cos, sin := math.Cos(angle), math.Sin(angle)
		x := d.CenterX + int(math.Round(float64(diagramRadius)*cos))
		y := d.CenterY + int(math.Round(float64(diagramRadius)*sin))
		// Push the label a little further out than the node, and anchor
		// it left/right depending on which half it sits in so text never
		// runs back over the hub.
		labelX := d.CenterX + int(math.Round(float64(diagramRadius+22)*cos))
		labelY := d.CenterY + int(math.Round(float64(diagramRadius+22)*sin))
		anchor := "middle"
		switch {
		case cos > 0.3:
			anchor = "start"
		case cos < -0.3:
			anchor = "end"
		}
		d.Nodes = append(d.Nodes, DiagramNode{
			X: x, Y: y, LabelX: labelX, LabelY: labelY, Anchor: anchor,
			Label: f.Label, Score: f.Score,
		})
	}
	return d
}
