package ui

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestBuildFleetScore(t *testing.T) {
	tests := []struct {
		name       string
		rationales []models.Rationale
		want       FleetScore
	}{
		{
			name: "all soeverein",
			rationales: []models.Rationale{
				rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
				rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
				rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in NL", models.ScoreSoeverein),
				rationale("wand.juridisch.cert_issuer_eea", "cert in NL", models.ScoreSoeverein),
				rationale("wand.transit.eu_path", "transit in EER", models.ScoreSoeverein),
				rationale("wand.technologie.no_us_hyperscaler", "no hyperscaler", models.ScoreSoeverein),
				rationale("wand.technologie.third_parties_eea", "third parties in EER", models.ScoreSoeverein),
			},
			want: FleetScore{X: 7, N: 7, Unanswered: 0},
		},
		{
			name: "one afhankelijk",
			rationales: []models.Rationale{
				rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
				rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
				rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in NL", models.ScoreSoeverein),
				rationale("wand.juridisch.cert_issuer_eea", "cert in NL", models.ScoreSoeverein),
				rationale("wand.transit.eu_path", "transit in EER", models.ScoreSoeverein),
				rationale("wand.technologie.no_us_hyperscaler", "no hyperscaler", models.ScoreSoeverein),
				rationale("wand.technologie.third_parties_eea", "third parties in EER", models.ScoreSoeverein),
			},
			want: FleetScore{
				X: 6, N: 7, Unanswered: 0,
				WorstFlow:    "Mail",
				WorstVerdict: "mx hosts in US (outside EEA)",
			},
		},
		{
			name: "everything onbekend",
			rationales: []models.Rationale{
				rationale("wand.juridisch.apex_ip_eea", "probe failed", models.ScoreOnbekend),
				rationale("wand.juridisch.mx_vendor_jurisdiction", "probe failed", models.ScoreOnbekend),
			},
			want: FleetScore{X: 0, N: 0, Unanswered: 2},
		},
		{
			name:       "no flows at all",
			rationales: nil,
			want:       FleetScore{X: 0, N: 0, Unanswered: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFleetScore(assessmentWith(tt.rationales...))
			if got != tt.want {
				t.Fatalf("BuildFleetScore() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestBuildFleetScore_SameXNDifferentUnansweredAreNotEqual guards the
// case the proposal calls out explicitly: two domains with the same
// x/n but a different unanswered count must not compare equal, or the
// fleet screen would silently drop the "2 onbekend" distinction
// (spec.md "Twee domeinen naast elkaar").
func TestBuildFleetScore_SameXNDifferentUnansweredAreNotEqual(t *testing.T) {
	fiveOfSeven := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.cert_issuer_eea", "cert in NL", models.ScoreSoeverein),
		rationale("wand.transit.eu_path", "transit in EER", models.ScoreSoeverein),
		rationale("wand.technologie.no_us_hyperscaler", "geoip unavailable", models.ScoreOnbekend),
		rationale("wand.technologie.third_parties_eea", "geoip unavailable", models.ScoreOnbekend),
	)
	fiveOfSevenOneUnknown := assessmentWith(
		rationale("wand.juridisch.apex_ip_eea", "apex in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.mx_vendor_jurisdiction", "mx in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in NL", models.ScoreSoeverein),
		rationale("wand.juridisch.cert_issuer_eea", "cert in NL", models.ScoreSoeverein),
		rationale("wand.transit.eu_path", "transit in EER", models.ScoreSoeverein),
		rationale("wand.technologie.no_us_hyperscaler", "geoip unavailable", models.ScoreOnbekend),
	)

	a := BuildFleetScore(fiveOfSeven)
	b := BuildFleetScore(fiveOfSevenOneUnknown)

	if a.X != b.X || a.N != b.N {
		t.Fatalf("test setup broken: want equal x/n, got %+v and %+v", a, b)
	}
	if a.Unanswered == b.Unanswered {
		t.Fatalf("unanswered counts both %d, want different", a.Unanswered)
	}
	if a == b {
		t.Fatalf("FleetScore values compare equal despite different Unanswered: %+v == %+v", a, b)
	}
}
