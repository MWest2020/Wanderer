package ui

import (
	"fmt"
	"math"
)

// The ring's geometry constants (design.md "sizes computed in Go, not in
// the template"): an SVG viewBox fleetRingSize square, a stroked circle of
// fleetRingRadius centred at fleetRingCenter, thick enough
// (fleetRingStrokeWidth) to read as a ring rather than a thin line. All
// four are exposed on FleetRing so dashboard.tmpl never repeats them.
const (
	fleetRingSize        = 140.0
	fleetRingCenter      = fleetRingSize / 2
	fleetRingStrokeWidth = 16.0
	fleetRingRadius      = fleetRingCenter - fleetRingStrokeWidth/2
)

var fleetRingCircumference = 2 * math.Pi * fleetRingRadius

// RingSegment is one arc of the fleet-score ring — a `<circle>` sharing
// the ring's centre/radius, distinguished only by its
// stroke-dasharray/stroke-dashoffset (which draw/rotate the arc) and its
// score-* class (which colours it, from the existing verdict palette).
type RingSegment struct {
	Class      string
	Dasharray  string
	Dashoffset string
}

// RingLegendItem is one line of the ring's legend (run 02, nakijken op
// prod-data task 1b.2): a colour bullet matching a segment's class, its
// Dutch label, and the same Count the segment's arc was sized from — so
// the onbeantwoord count, which used to live only inside the ring's
// aria-label, is visible as plain text too.
type RingLegendItem struct {
	Class string
	Label string
	Count int
}

// FleetRing is the vlootscore ring (proposal.md "de vlootscore wordt een
// ring": soeverein / niet-soeverein / onbeantwoord as segments, x/n in the
// middle). AriaLabel carries the same facts as text (spec.md scenario
// "Elk beeld heeft zijn woorden"): "{x} van {n} vragen soeverein, {u}
// onbeantwoord". Legend carries those same three counts as a visible
// list next to the ring, always in soeverein/niet-soeverein/onbeantwoord
// order regardless of which segments actually drew an arc.
type FleetRing struct {
	AriaLabel   string
	CenterLabel string

	Size, Center, Radius, StrokeWidth float64

	Segments []RingSegment
	Legend   []RingLegendItem
}

// BuildFleetRing turns a fleet's x/n/onbeantwoord counts into ring
// geometry. x is how many sovereignty questions scored soeverein or
// voldoende (FleetSummary.X), n how many could be scored at all
// (FleetSummary.N) so n-x is the niet-soeverein segment, and unanswered
// the onbekend segment kept apart from n (FleetSummary.Unanswered). A
// fleet with nothing to show (n+unanswered == 0) renders the bare grey
// track — no segment, no division by zero — but Legend still lists its
// three zero counts, since "nothing to show" is itself a fact worth
// reading (run 02: the onbeantwoord count used to disappear entirely
// when it wasn't part of an arc).
func BuildFleetRing(x, n, unanswered int) FleetRing {
	parts := []struct {
		class string
		label string
		count int
	}{
		{"score-soeverein", "soeverein", x},
		{"score-afhankelijk", "niet soeverein", n - x},
		{"score-onbekend", "onbeantwoord", unanswered},
	}
	ring := FleetRing{
		AriaLabel:   fmt.Sprintf("%d van %d vragen soeverein, %d onbeantwoord", x, n, unanswered),
		CenterLabel: fmt.Sprintf("%d/%d", x, n),
		Size:        fleetRingSize,
		Center:      fleetRingCenter,
		Radius:      fleetRingRadius,
		StrokeWidth: fleetRingStrokeWidth,
	}
	for _, p := range parts {
		ring.Legend = append(ring.Legend, RingLegendItem{Class: p.class, Label: p.label, Count: p.count})
	}
	total := n + unanswered
	if total <= 0 {
		return ring
	}
	var cumulative float64
	for _, p := range parts {
		if p.count <= 0 {
			continue
		}
		frac := float64(p.count) / float64(total)
		arc := frac * fleetRingCircumference
		ring.Segments = append(ring.Segments, RingSegment{
			Class:      p.class,
			Dasharray:  fmt.Sprintf("%.3f %.3f", arc, fleetRingCircumference-arc),
			Dashoffset: fmt.Sprintf("%.3f", -cumulative*fleetRingCircumference),
		})
		cumulative += frac
	}
	return ring
}
