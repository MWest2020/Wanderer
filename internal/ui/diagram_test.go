package ui

import "testing"

func TestSovereigntyDiagram_LaysOutNodes(t *testing.T) {
	flows := []Flow{
		{Label: "Hosting", Score: "soeverein"},
		{Label: "Mail", Score: "afhankelijk"},
		{Label: "DNS", Score: "onbekend"},
	}
	d := SovereigntyDiagram("example.nl", flows)
	if d.Subject != "example.nl" {
		t.Fatalf("subject = %q", d.Subject)
	}
	if len(d.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(d.Nodes))
	}
	for i, n := range d.Nodes {
		if n.Label != flows[i].Label || n.Score != flows[i].Score {
			t.Errorf("node[%d] = %+v, want %+v", i, n, flows[i])
		}
		if n.X < 0 || n.X > d.Width || n.Y < 0 || n.Y > d.Height {
			t.Errorf("node[%d] off-canvas: (%d,%d) in %dx%d", i, n.X, n.Y, d.Width, d.Height)
		}
	}
	// First node sits at the top (−90°): same X as centre, above it.
	if d.Nodes[0].X != d.CenterX || d.Nodes[0].Y >= d.CenterY {
		t.Errorf("first node should be top-centre, got (%d,%d) centre (%d,%d)", d.Nodes[0].X, d.Nodes[0].Y, d.CenterX, d.CenterY)
	}
}

func TestSovereigntyDiagram_Empty(t *testing.T) {
	if d := SovereigntyDiagram("x", nil); len(d.Nodes) != 0 {
		t.Fatalf("empty flows → %d nodes, want 0", len(d.Nodes))
	}
}
