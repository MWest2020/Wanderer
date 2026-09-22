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

// allSovereignRationales returns the seven flow rules all scoring
// soeverein, with overrides for the CriteriumIDs in `overrides`
// applied on top — the shared fixture for BuildFleetSummary's tests
// below.
func allSovereignRationales(overrides map[string]models.Rationale) []models.Rationale {
	ids := []string{
		"wand.juridisch.apex_ip_eea",
		"wand.juridisch.mx_vendor_jurisdiction",
		"wand.juridisch.ns_vendor_jurisdiction",
		"wand.juridisch.cert_issuer_eea",
		"wand.transit.eu_path",
		"wand.technologie.no_us_hyperscaler",
		"wand.technologie.third_parties_eea",
	}
	out := make([]models.Rationale, 0, len(ids))
	for _, id := range ids {
		if r, ok := overrides[id]; ok {
			out = append(out, r)
			continue
		}
		out = append(out, rationale(id, "soeverein", models.ScoreSoeverein))
	}
	return out
}

// fleetSnap builds one domain's TargetSnapshot for the fleet-summary
// tests: like the package's `snap` helper, but with LastStatus set so
// BuildFleetSummary's voltooide-scan filter can be exercised.
func fleetSnap(targetID, domain, status string, rationales ...models.Rationale) TargetSnapshot {
	s := snap(targetID, domain, "wand", rationales...)
	s.LastStatus = status
	return s
}

func TestBuildFleetSummary(t *testing.T) {
	tests := []struct {
		name  string
		snaps []TargetSnapshot
		check func(t *testing.T, got FleetSummary)
	}{
		{
			// The proposal's valkuil: nine good domains and one bad one
			// must not hide the bad one behind a high aggregate score.
			name: "nine sovereign one afhankelijk score is high but the bad domain still counts",
			snaps: func() []TargetSnapshot {
				snaps := make([]TargetSnapshot, 0, 10)
				for i := 0; i < 9; i++ {
					id := string(rune('a' + i))
					snaps = append(snaps, fleetSnap(id, id+".example", "complete", allSovereignRationales(nil)...))
				}
				bad := allSovereignRationales(map[string]models.Rationale{
					"wand.juridisch.mx_vendor_jurisdiction": rationale("wand.juridisch.mx_vendor_jurisdiction", "mx hosts in US (outside EEA)", models.ScoreAfhankelijk),
				})
				snaps = append(snaps, fleetSnap("bad", "bad.example", "complete", bad...))
				return snaps
			}(),
			check: func(t *testing.T, got FleetSummary) {
				if got.N != 70 {
					t.Fatalf("N = %d, want 70 (10 domains * 7 flows)", got.N)
				}
				if got.X != 69 {
					t.Fatalf("X = %d, want 69 (one flow afhankelijk)", got.X)
				}
				if float64(got.X)/float64(got.N) < 0.9 {
					t.Fatalf("score = %d/%d, want a high aggregate score despite the bad domain", got.X, got.N)
				}
				if got.NotSovereign != 1 {
					t.Fatalf("NotSovereign = %d, want 1 — the bad domain must not be hidden", got.NotSovereign)
				}
			},
		},
		{
			name: "domain without a voltooide scan is excluded from x and n, counted apart",
			snaps: []TargetSnapshot{
				fleetSnap("t1", "a.example", "complete", allSovereignRationales(nil)...),
				fleetSnap("t2", "b.example", "running", allSovereignRationales(nil)...),
			},
			check: func(t *testing.T, got FleetSummary) {
				if got.X != 7 || got.N != 7 {
					t.Fatalf("X/N = %d/%d, want 7/7 — the running domain must not count", got.X, got.N)
				}
				if got.DomainsWithoutScan != 1 {
					t.Fatalf("DomainsWithoutScan = %d, want 1", got.DomainsWithoutScan)
				}
			},
		},
		{
			name: "everything onbekend is 0/0 without dividing by zero",
			snaps: []TargetSnapshot{
				fleetSnap(
					"t1", "a.example", "complete",
					rationale("wand.juridisch.apex_ip_eea", "probe failed", models.ScoreOnbekend),
					rationale("wand.juridisch.mx_vendor_jurisdiction", "probe failed", models.ScoreOnbekend),
				),
			},
			check: func(t *testing.T, got FleetSummary) {
				if got.X != 0 || got.N != 0 {
					t.Fatalf("X/N = %d/%d, want 0/0", got.X, got.N)
				}
				if got.Unanswered != 2 {
					t.Fatalf("Unanswered = %d, want 2", got.Unanswered)
				}
				if got.NotSovereign != 0 {
					t.Fatalf("NotSovereign = %d, want 0", got.NotSovereign)
				}
			},
		},
		{
			name:  "empty organisation gives an empty result",
			snaps: nil,
			check: func(t *testing.T, got FleetSummary) {
				want := FleetSummary{}
				if got.X != want.X || got.N != want.N || got.Unanswered != want.Unanswered ||
					got.DomainsWithoutScan != want.DomainsWithoutScan || got.NotSovereign != want.NotSovereign {
					t.Fatalf("got = %+v, want all-zero", got)
				}
				if len(got.TopRules) != 0 || len(got.Flows) != 0 {
					t.Fatalf("got = %+v, want no rules and no flows", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFleetSummary(tt.snaps, nil)
			tt.check(t, got)
		})
	}
}

// TestBuildFleetSummary_TopRulesReusesTopConcerns pins that TopRules is
// TopConcerns' own afhankelijk/distinct-domain counting, not a
// reimplementation: the same rule firing afhankelijk on multiple
// domains must be the top row, capped at three total.
func TestBuildFleetSummary_TopRulesReusesTopConcerns(t *testing.T) {
	worst := "wand.juridisch.mx_vendor_jurisdiction"
	snaps := []TargetSnapshot{
		fleetSnap("t1", "a.example", "complete", rationale(worst, "mx in US", models.ScoreAfhankelijk)),
		fleetSnap("t2", "b.example", "complete", rationale(worst, "mx in US", models.ScoreAfhankelijk)),
		fleetSnap("t3", "c.example", "complete", rationale("wand.juridisch.ns_vendor_jurisdiction", "ns in US", models.ScoreAfhankelijk)),
	}
	got := BuildFleetSummary(snaps, nil)
	if len(got.TopRules) != 2 {
		t.Fatalf("TopRules = %+v, want 2 rows", got.TopRules)
	}
	if got.TopRules[0].CriteriumID != worst || got.TopRules[0].TargetCount != 2 {
		t.Fatalf("TopRules[0] = %+v, want %s with count 2", got.TopRules[0], worst)
	}
}
