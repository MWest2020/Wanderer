package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// marshalExpectedRegistrant renders names as the JSON array stored in
// organisations.expected_registrant. A nil slice marshals to "[]", not
// the JSON literal null — the column is NOT NULL and callers that
// unmarshal it (GetOrganisation*, ListOrganisations) expect valid JSON.
func marshalExpectedRegistrant(names []string) (string, error) {
	if names == nil {
		names = []string{}
	}
	b, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("store: marshal expected_registrant: %w", err)
	}
	return string(b), nil
}

// UpsertOrganisation inserts a new organisation or updates the
// name + description of an existing one (matched by slug). Idempotent
// — running the same input twice keeps the same ID.
//
// o.ExpectedRegistrant is nil-sensitive: a nil slice (the zero value,
// e.g. when a caller only sets Slug/Name/Description) leaves the
// stored list untouched and is filled in with the existing value on
// return. A non-nil slice — including an explicitly empty one —
// overwrites the stored list. This lets `wanderer org add` update
// name/description without a --expected-registrant flag wiping names
// set by a previous call.
func (s *Store) UpsertOrganisation(ctx context.Context, o *models.Organisation) error {
	if err := o.Validate(); err != nil {
		return err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, expected_registrant FROM organisations WHERE slug = ?`, o.Slug)
	var id, existingRegistrant string
	var createdAt time.Time
	switch err := row.Scan(&id, &createdAt, &existingRegistrant); {
	case errors.Is(err, sql.ErrNoRows):
		o.ID = newID("o")
		o.CreatedAt = time.Now().UTC()
		er, err := marshalExpectedRegistrant(o.ExpectedRegistrant)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO organisations (id, slug, name, description, expected_registrant, created_at) VALUES (?,?,?,?,?,?)`,
			o.ID, o.Slug, o.Name, o.Description, er, o.CreatedAt); err != nil {
			return fmt.Errorf("store: insert organisation: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("store: lookup organisation: %w", err)
	}
	o.ID = id
	o.CreatedAt = createdAt
	if o.ExpectedRegistrant == nil {
		if err := json.Unmarshal([]byte(existingRegistrant), &o.ExpectedRegistrant); err != nil {
			return fmt.Errorf("store: unmarshal expected_registrant: %w", err)
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE organisations SET name = ?, description = ? WHERE id = ?`,
			o.Name, o.Description, o.ID); err != nil {
			return fmt.Errorf("store: update organisation: %w", err)
		}
		return nil
	}
	er, err := marshalExpectedRegistrant(o.ExpectedRegistrant)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE organisations SET name = ?, description = ?, expected_registrant = ? WHERE id = ?`,
		o.Name, o.Description, er, o.ID); err != nil {
		return fmt.Errorf("store: update organisation: %w", err)
	}
	return nil
}

// GetOrganisationBySlug returns the row matching slug, or
// ErrNotFound if no row exists. Slug is the operator-facing handle;
// callers should resolve once and reuse the ID.
func (s *Store) GetOrganisationBySlug(ctx context.Context, slug string) (*models.Organisation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, expected_registrant, created_at FROM organisations WHERE slug = ?`, slug)
	o := &models.Organisation{}
	var er string
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.Description, &er, &o.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select organisation: %w", err)
	}
	if err := json.Unmarshal([]byte(er), &o.ExpectedRegistrant); err != nil {
		return nil, fmt.Errorf("store: unmarshal expected_registrant: %w", err)
	}
	return o, nil
}

// GetOrganisation returns the row by ID. Useful in code paths that
// have already resolved a slug and now need the full row for
// rendering.
func (s *Store) GetOrganisation(ctx context.Context, id string) (*models.Organisation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, expected_registrant, created_at FROM organisations WHERE id = ?`, id)
	o := &models.Organisation{}
	var er string
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.Description, &er, &o.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select organisation: %w", err)
	}
	if err := json.Unmarshal([]byte(er), &o.ExpectedRegistrant); err != nil {
		return nil, fmt.Errorf("store: unmarshal expected_registrant: %w", err)
	}
	return o, nil
}

// ListOrganisations returns every organisation, ordered by slug. Small
// result sets — operators rarely have more than a handful of orgs —
// so streaming an iterator would be over-engineering.
func (s *Store) ListOrganisations(ctx context.Context) ([]models.Organisation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, description, expected_registrant, created_at FROM organisations ORDER BY slug`)
	if err != nil {
		return nil, fmt.Errorf("store: list organisations: %w", err)
	}
	defer rows.Close()
	var out []models.Organisation
	for rows.Next() {
		var o models.Organisation
		var er string
		if err := rows.Scan(&o.ID, &o.Slug, &o.Name, &o.Description, &er, &o.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(er), &o.ExpectedRegistrant); err != nil {
			return nil, fmt.Errorf("store: unmarshal expected_registrant: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// RenameOrganisation updates the slug and/or name of an existing
// organisation. The escape hatch from the migration's seeded
// `default` slug — see Q1 in the add-organisation-pivot proposal.
// An unknown old-slug returns ErrNotFound; a duplicate new-slug
// returns the underlying SQLite UNIQUE error.
func (s *Store) RenameOrganisation(ctx context.Context, oldSlug, newSlug, newName string) error {
	o := &models.Organisation{Slug: newSlug, Name: newName}
	if err := o.Validate(); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE organisations SET slug = ?, name = ? WHERE slug = ?`,
		newSlug, newName, oldSlug)
	if err != nil {
		return fmt.Errorf("store: rename organisation: %w", err)
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

// ListTargetsByOrganisation returns every Target attached to one
// organisation, ordered by domain. Used by the per-org dashboard
// and the MCP `org.targets` method.
func (s *Store) ListTargetsByOrganisation(ctx context.Context, orgID string) ([]models.Target, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, domain, COALESCE(kind,'domain'), organisation_id, created_at
		   FROM targets
		  WHERE organisation_id = ?
		  ORDER BY domain`, orgID)
	if err != nil {
		return nil, fmt.Errorf("store: list targets by organisation: %w", err)
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
