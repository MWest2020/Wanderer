package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// AddFleetDomain adds domain to organisation orgID's fleet without
// scanning it (spec.md "Domeinen zijn bij te houden als vloot"). A
// domain new to the store is created as a bare Target — no Scan, no
// Finding — so it shows as "nog niet gescand" until the first scan
// lands. Re-adding a domain previously removed from this same
// organisation's fleet un-removes the existing row in place, so its
// scan history keeps lining up with the same Target ID. A domain
// that already belongs to a DIFFERENT organisation is rejected:
// targets.domain is globally unique, so silently reassigning it
// would pull that org's scan history out from under it.
func (s *Store) AddFleetDomain(ctx context.Context, orgID, domain string) (*models.Target, error) {
	d, err := models.NormaliseDomain(domain)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, organisation_id FROM targets WHERE domain = ?`, d)
	var id, existingOrg string
	switch err := row.Scan(&id, &existingOrg); {
	case errors.Is(err, sql.ErrNoRows):
		t := &models.Target{Domain: d, OrganisationID: orgID}
		if err := s.UpsertTarget(ctx, t); err != nil {
			return nil, err
		}
		return t, nil
	case err != nil:
		return nil, fmt.Errorf("store: lookup target: %w", err)
	}
	if existingOrg != orgID {
		return nil, fmt.Errorf("store: domain %q already belongs to a different organisation", d)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE targets SET removed_at = NULL WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("store: un-remove target: %w", err)
	}
	return s.GetTarget(ctx, id)
}

// RemoveFleetDomain takes domain out of organisation orgID's fleet.
// It only stamps the Target row's removed_at — scans and findings
// that reference it by target_id are untouched and stay reachable by
// anyone who already has the scan/report URL (spec.md "Een domein
// verwijderen SHALL de eerdere scans en oordelen niet weggooien").
// Returns ErrNotFound when no active (not already removed) target
// matches domain under this organisation.
func (s *Store) RemoveFleetDomain(ctx context.Context, orgID, domain string) error {
	d, err := models.NormaliseDomain(domain)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE targets SET removed_at = ? WHERE domain = ? AND organisation_id = ? AND removed_at IS NULL`,
		time.Now().UTC(), d, orgID)
	if err != nil {
		return fmt.Errorf("store: remove target: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListFleetDomains returns every domain currently in organisation
// orgID's fleet — i.e. not soft-removed — ordered by domain. This is
// deliberately a separate query from ListTargetsByOrganisation:
// existing callers of that method (MCP org.targets, the Trends target
// count) expect every target an org has ever scanned, removed or
// not, so filtering it here would change behaviour nobody asked for.
func (s *Store) ListFleetDomains(ctx context.Context, orgID string) ([]models.Target, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, domain, COALESCE(kind,'domain'), organisation_id, created_at
		   FROM targets
		  WHERE organisation_id = ? AND removed_at IS NULL
		  ORDER BY domain`, orgID)
	if err != nil {
		return nil, fmt.Errorf("store: list fleet domains: %w", err)
	}
	defer rows.Close()
	var out []models.Target
	for rows.Next() {
		var t models.Target
		var kind string
		if err := rows.Scan(&t.ID, &t.Domain, &kind, &t.OrganisationID, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Kind = models.TargetKind(kind)
		out = append(out, t)
	}
	return out, rows.Err()
}
