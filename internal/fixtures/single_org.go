package fixtures

import (
	"context"
	"fmt"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// BuildSingleOrg seeds an instance carrying exactly one organisation
// — the migration's seeded `default` — with one scanned domain. It
// is the vloot-beheren-bereikbaar Playwright scenario: /ui/ SHALL
// link straight to the fleet manager when the door shows one
// organisation's fleet, a case none of the other scenarios exercise
// (baseline and its derivatives always carry voorbeeld + acme
// alongside the migration's default org).
func BuildSingleOrg(ctx context.Context, st *store.Store) error {
	o, err := st.GetOrganisationBySlug(ctx, "default")
	if err != nil {
		return fmt.Errorf("single-org: lookup default org: %w", err)
	}
	target, err := upsertTarget(ctx, st, "solo.nl", models.TargetKindDomain, o.ID)
	if err != nil {
		return err
	}
	if _, err := addCompletedScan(ctx, st, target, baseTime, baselineSovereignFindings("solo.nl")); err != nil {
		return fmt.Errorf("single-org: scan: %w", err)
	}
	return nil
}
