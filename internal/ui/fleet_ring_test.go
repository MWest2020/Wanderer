package ui

import (
	"fmt"
	"math"
	"testing"
)

// TestBuildFleetRing_AriaLabelCarriesTheFacts pins spec.md's scenario
// "Elk beeld heeft zijn woorden": "24 van 46 vragen soeverein, 31
// onbeantwoord" for x=24, n=46, unanswered=31.
func TestBuildFleetRing_AriaLabelCarriesTheFacts(t *testing.T) {
	ring := BuildFleetRing(24, 46, 31)
	want := "24 van 46 vragen soeverein, 31 onbeantwoord"
	if ring.AriaLabel != want {
		t.Errorf("AriaLabel = %q, want %q", ring.AriaLabel, want)
	}
	if ring.CenterLabel != "24/46" {
		t.Errorf("CenterLabel = %q, want 24/46", ring.CenterLabel)
	}
}

// TestBuildFleetRing_SegmentsSumToFullCircle is the geometry test
// design.md asks for: "Geometry ... computed in Go (not in the
// template), so the arithmetic is unit-tested". Every segment's arc
// length (the first number of its Dasharray) must sum to the circle's
// full circumference, within floating-point tolerance.
func TestBuildFleetRing_SegmentsSumToFullCircle(t *testing.T) {
	ring := BuildFleetRing(24, 46, 31)
	if len(ring.Segments) != 3 {
		t.Fatalf("Segments = %d, want 3 (soeverein, niet-soeverein, onbekend)", len(ring.Segments))
	}
	var total float64
	for _, seg := range ring.Segments {
		var arc, gap float64
		if _, err := fmt.Sscanf(seg.Dasharray, "%f %f", &arc, &gap); err != nil {
			t.Fatalf("parse dasharray %q: %v", seg.Dasharray, err)
		}
		total += arc
	}
	if math.Abs(total-fleetRingCircumference) > 0.01 {
		t.Errorf("segment arcs sum to %.3f, want %.3f (the full circumference)", total, fleetRingCircumference)
	}
}

// TestBuildFleetRing_ClassesMatchExistingVerdictPalette pins design.md
// "Colours": the ring reuses the existing score-* palette rather than
// inventing a parallel one.
func TestBuildFleetRing_ClassesMatchExistingVerdictPalette(t *testing.T) {
	ring := BuildFleetRing(1, 2, 3)
	want := map[string]bool{"score-soeverein": false, "score-afhankelijk": false, "score-onbekend": false}
	for _, seg := range ring.Segments {
		if _, ok := want[seg.Class]; !ok {
			t.Errorf("unexpected class %q", seg.Class)
			continue
		}
		want[seg.Class] = true
	}
	for class, seen := range want {
		if !seen {
			t.Errorf("class %q never emitted for x=1,n=2,unanswered=3", class)
		}
	}
}

// TestBuildFleetRing_NoDataRendersBareTrackWithoutPanicking covers the
// n=0/unanswered=0 edge — an organisation with domains but no scored
// question yet must not divide by zero.
func TestBuildFleetRing_NoDataRendersBareTrackWithoutPanicking(t *testing.T) {
	ring := BuildFleetRing(0, 0, 0)
	if len(ring.Segments) != 0 {
		t.Errorf("Segments = %+v, want none when there is nothing to show", ring.Segments)
	}
}

// TestBuildFleetRing_LegendCountsMatchSegments pins run 02's nakijken
// finding (task 1b.2): the ring needs a visible legend next to it, and
// its counts must come from the same Go computation as the segments —
// not a second, independently-derived set of numbers that could drift.
func TestBuildFleetRing_LegendCountsMatchSegments(t *testing.T) {
	ring := BuildFleetRing(24, 46, 31)
	if len(ring.Legend) != 3 {
		t.Fatalf("Legend = %d items, want 3 (soeverein, niet soeverein, onbeantwoord)", len(ring.Legend))
	}
	want := map[string]struct {
		label string
		count int
	}{
		"score-soeverein":   {"soeverein", 24},
		"score-afhankelijk": {"niet soeverein", 22}, // n - x = 46 - 24
		"score-onbekend":    {"onbeantwoord", 31},
	}
	seen := map[string]bool{}
	for _, item := range ring.Legend {
		w, ok := want[item.Class]
		if !ok {
			t.Errorf("unexpected legend class %q", item.Class)
			continue
		}
		seen[item.Class] = true
		if item.Label != w.label {
			t.Errorf("%s: Label = %q, want %q", item.Class, item.Label, w.label)
		}
		if item.Count != w.count {
			t.Errorf("%s: Count = %d, want %d", item.Class, item.Count, w.count)
		}
	}
	for class := range want {
		if !seen[class] {
			t.Errorf("legend never carried class %q", class)
		}
	}

	// Cross-check against the segments the same call produced: the
	// legend must not just be plausible-looking, it must equal what
	// actually got drawn.
	segByClass := map[string]bool{}
	for _, seg := range ring.Segments {
		segByClass[seg.Class] = true
	}
	for class, w := range want {
		if w.count > 0 && !segByClass[class] {
			t.Errorf("legend lists %q with count %d but no matching segment was drawn", class, w.count)
		}
	}
}

// TestBuildFleetRing_LegendListsZeroCountsWhenThereIsNothingToShow
// covers the n=0/unanswered=0 edge the ring itself renders as a bare
// track for: the legend still lists all three lines, at 0, rather than
// disappearing along with the segments.
func TestBuildFleetRing_LegendListsZeroCountsWhenThereIsNothingToShow(t *testing.T) {
	ring := BuildFleetRing(0, 0, 0)
	if len(ring.Legend) != 3 {
		t.Fatalf("Legend = %d items, want 3 even with nothing to show", len(ring.Legend))
	}
	for _, item := range ring.Legend {
		if item.Count != 0 {
			t.Errorf("%s: Count = %d, want 0", item.Class, item.Count)
		}
	}
}

// TestBuildFleetRing_AllSovereignHasOneSegment covers a fleet with
// nothing niet-soeverein or onbeantwoord: only the soeverein segment
// renders, not three segments with two at zero width.
func TestBuildFleetRing_AllSovereignHasOneSegment(t *testing.T) {
	ring := BuildFleetRing(5, 5, 0)
	if len(ring.Segments) != 1 || ring.Segments[0].Class != "score-soeverein" {
		t.Errorf("Segments = %+v, want exactly one score-soeverein segment", ring.Segments)
	}
}
