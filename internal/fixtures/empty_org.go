package fixtures

import (
	"context"

	"github.com/MWest2020/wanderer/internal/store"
)

// BuildEmptyOrg seeds an `acme-empty` org with zero targets so
// the UI's empty-state paths (per-org dashboard, reporting
// catalogue with no rationale rows, targets list with no rows)
// render and can be pinned by Playwright. The migration's
// `default` org plus this one is the entire universe of orgs.
func BuildEmptyOrg(ctx context.Context, st *store.Store) error {
	_, err := upsertOrg(ctx, st, "acme-empty", "ACME (empty)")
	return err
}
