// Package drift compares two scans of the same target and emits drift
// Findings for changes worth surfacing. Drift Findings carry
// source_modus=drift in their Attributes alongside the two scan IDs
// they were derived from. The package is a pure consumer of
// models.Scan; it has no I/O of its own.
package drift

import (
	"context"
	"errors"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// SourceModusDrift marks a Finding as produced by drift analysis
// rather than by a probe. Stored under Attributes["source_modus"].
const SourceModusDrift = "drift"

// Diff returns the Findings that describe how curr differs from prev.
// When prev is nil a single drift.baseline_established Finding is
// returned. When prev and curr are equivalent (no rule fires) a
// single drift.no_changes Finding is returned. Diff itself is pure;
// callers persist the Findings via the store.
func Diff(prev, curr *models.Scan) []models.Finding {
	if curr == nil {
		return nil
	}
	if prev == nil {
		return []models.Finding{baselineFinding(curr)}
	}
	var out []models.Finding
	for _, rule := range DefaultRules {
		out = append(out, rule(prev, curr)...)
	}
	if len(out) == 0 {
		return []models.Finding{noChangesFinding(prev, curr)}
	}
	return out
}

// Compute is the storage-aware entry point. It looks up the previous
// scan for curr's target before calling Diff, then returns the
// resulting Findings without persisting them. Persistence is the
// caller's responsibility (the scheduler appends them to the store;
// the diff CLI prints them to stdout).
func Compute(ctx context.Context, st *store.Store, curr *models.Scan) ([]models.Finding, error) {
	if curr == nil {
		return nil, errors.New("drift: curr is nil")
	}
	prev, err := st.PreviousScanForTarget(ctx, curr.TargetID, curr.StartedAt)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	return Diff(prev, curr), nil
}

func baselineFinding(curr *models.Scan) models.Finding {
	return models.Finding{
		ProbeID:  "drift.baseline_established",
		Subject:  scanSubject(curr),
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"source_modus":  SourceModusDrift,
			"curr_scan_id":  curr.ID,
		},
	}
}

func noChangesFinding(prev, curr *models.Scan) models.Finding {
	return models.Finding{
		ProbeID:  "drift.no_changes",
		Subject:  scanSubject(curr),
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"source_modus": SourceModusDrift,
			"prev_scan_id": prev.ID,
			"curr_scan_id": curr.ID,
		},
	}
}

// scanSubject picks the most representative subject for a drift
// Finding: the first non-empty Subject from the scan's Findings, or
// the empty string if the scan has none.
func scanSubject(scan *models.Scan) string {
	for _, f := range scan.Findings {
		if f.Subject != "" {
			return f.Subject
		}
	}
	return ""
}
