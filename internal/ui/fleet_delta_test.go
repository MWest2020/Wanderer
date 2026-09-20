package ui

import (
	"reflect"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestBuildFleetDelta_NoPreviousScan(t *testing.T) {
	curr := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
	)
	got := BuildFleetDelta(false, nil, curr)
	want := FleetDelta{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFleetDelta() = %+v, want %+v", got, want)
	}
}

func TestBuildFleetDelta_NoChange(t *testing.T) {
	assessments := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
	)
	got := BuildFleetDelta(true, assessments, assessments)
	want := FleetDelta{HasPrevious: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFleetDelta() = %+v, want %+v", got, want)
	}
}

func TestBuildFleetDelta_FlowFlipsIntoAfhankelijk(t *testing.T) {
	prev := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
	)
	curr := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
	)
	got := BuildFleetDelta(true, prev, curr)
	want := FleetDelta{
		HasPrevious:  true,
		XDelta:       -1,
		NDelta:       0, // still counted in n, just on the other side
		FlippedFlows: []string{"Mail"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFleetDelta() = %+v, want %+v", got, want)
	}
}

func TestBuildFleetDelta_FlowFlipsOutOfAfhankelijk(t *testing.T) {
	prev := assessmentWith(
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
	)
	curr := assessmentWith(
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
	)
	got := BuildFleetDelta(true, prev, curr)
	want := FleetDelta{
		HasPrevious:  true,
		XDelta:       1,
		NDelta:       0,
		FlippedFlows: []string{"Mail"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFleetDelta() = %+v, want %+v", got, want)
	}
}

// TestBuildFleetDelta_OnbekendToSoevereinIsNotAFlip guards the
// distinction the doc comment calls out: a question that was
// unanswerable and becomes answerable changes x and n ("vragen erbij")
// but never crossed the afhankelijk threshold, so it must not be
// reported as a flipped flow.
func TestBuildFleetDelta_OnbekendToSoevereinIsNotAFlip(t *testing.T) {
	prev := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "probe failed", models.ScoreOnbekend),
	)
	curr := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
	)
	got := BuildFleetDelta(true, prev, curr)
	want := FleetDelta{
		HasPrevious: true,
		XDelta:      1,
		NDelta:      1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFleetDelta() = %+v, want %+v", got, want)
	}
}

func TestBuildFleetDelta_MultipleFlipsKeepFixedOrder(t *testing.T) {
	prev := assessmentWith(
		rationale("wand.technologie.third_parties_eea", "third parties in EER", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
	)
	curr := assessmentWith(
		rationale("wand.technologie.third_parties_eea", "third parties outside EER", models.ScoreAfhankelijk),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
	)
	got := BuildFleetDelta(true, prev, curr)
	// SovereigntyFlows' fixed order puts Mail before Third parties.
	want := []string{"Mail", "Third parties"}
	if !reflect.DeepEqual(got.FlippedFlows, want) {
		t.Fatalf("FlippedFlows = %v, want %v", got.FlippedFlows, want)
	}
}
