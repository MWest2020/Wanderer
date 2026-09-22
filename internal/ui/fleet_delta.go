package ui

import "github.com/MWest2020/wanderer/pkg/models"

// FleetDelta is one domain's change since its previous scan (run 03
// task 3.2: "hoeveel vragen erbij of eraf, en welke stroom omsloeg").
// XDelta / NDelta are the change in FleetScore's x and n — together
// they read as questions coming online ("erbij") or dropping off
// ("eraf") between the two scans. FlippedFlows names the flows whose
// score crossed into or out of afhankelijk, in SovereigntyFlows' fixed
// order — "welke stroom omsloeg". A flow going from onbekend to
// soeverein (or back) changes N/X but is not a flip: it did not cross
// the sovereignty threshold. HasPrevious is false when there is no
// earlier scan to compare against; the other fields are then zero and
// MUST NOT be rendered as "no change" — the caller distinguishes the
// two by checking HasPrevious first.
type FleetDelta struct {
	HasPrevious bool
	XDelta      int
	NDelta      int

	FlippedFlows []string
}

// BuildFleetDelta compares two scans' assessments the same way
// BuildFleetScore scores a single one: by reusing
// SovereigntyFlows/classifyFlows for both sides instead of deriving a
// second notion of "changed" from raw Findings.
//
// internal/drift.Diff already compares two scans, but one layer down:
// it reads raw Findings (tls.issuer, dns.mx, ip.asn, ...) and emits
// operational drift Findings ("the MX set changed", "the TLS issuer
// changed") meant for alerting. The fleet screen's question is scored
// one layer up — did a *flow's verdict* (soeverein/afhankelijk/
// onbekend) change — which the drift rules do not compute and are not
// shaped to answer (they know nothing about wand's rule IDs or
// Rationale). The only reusable part is "find the previous scan",
// which already lives in store.PreviousScanForTarget and is used by
// this function's caller (fleetHandler) rather than duplicated here.
func BuildFleetDelta(hasPrevious bool, prev, curr []models.Assessment) FleetDelta {
	if !hasPrevious {
		return FleetDelta{}
	}
	prevScore := BuildFleetScore(prev, nil)
	currScore := BuildFleetScore(curr, nil)

	prevByLabel := make(map[string]models.Score, len(prev))
	for _, f := range SovereigntyFlows(prev, nil) {
		prevByLabel[f.Label] = models.Score(f.Score)
	}
	var flipped []string
	for _, f := range SovereigntyFlows(curr, nil) {
		was, ok := prevByLabel[f.Label]
		now := models.Score(f.Score)
		if !ok || was == now {
			continue
		}
		if (was == models.ScoreAfhankelijk) != (now == models.ScoreAfhankelijk) {
			flipped = append(flipped, f.Label)
		}
	}

	return FleetDelta{
		HasPrevious:  true,
		XDelta:       currScore.X - prevScore.X,
		NDelta:       currScore.N - prevScore.N,
		FlippedFlows: flipped,
	}
}
