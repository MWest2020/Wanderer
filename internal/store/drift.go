package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// PreviousScanForTarget returns the most recent scan for targetID
// that finished before before. ErrNotFound is returned when there is
// no such scan — the caller treats that as "this is a baseline run".
func (s *Store) PreviousScanForTarget(ctx context.Context, targetID string, before time.Time) (*models.Scan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_id, started_at, ended_at, status, COALESCE(error,'')
		 FROM scans
		 WHERE target_id = ? AND started_at < ?
		 ORDER BY started_at DESC, id DESC
		 LIMIT 1`,
		targetID, before.UTC())
	sc := &models.Scan{}
	var endedAt sql.NullTime
	var status string
	if err := row.Scan(&sc.ID, &sc.TargetID, &sc.StartedAt, &endedAt, &status, &sc.Error); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: previous scan: %w", err)
	}
	sc.Status = models.ScanStatus(status)
	if endedAt.Valid {
		t := endedAt.Time
		sc.EndedAt = &t
	}
	full, err := s.GetScan(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	return full, nil
}

// ListDriftForTarget returns drift Findings (those whose ProbeID
// starts with "drift.") associated with targetID's scans, ordered
// by created_at ascending. The since argument acts as a lower bound;
// pass time.Time{} to skip the filter.
func (s *Store) ListDriftForTarget(ctx context.Context, targetID string, since time.Time) ([]models.Finding, error) {
	q := `SELECT f.id, f.scan_id, f.probe_id, COALESCE(f.dimension_hint,''), COALESCE(f.criterium_hint,''),
	             f.subject, f.severity, f.attributes, f.evidence, f.created_at
	      FROM findings f
	      JOIN scans s ON s.id = f.scan_id
	      WHERE s.target_id = ?
	        AND f.probe_id LIKE 'drift.%'`
	args := []any{targetID}
	if !since.IsZero() {
		q += ` AND f.created_at >= ?`
		args = append(args, since.UTC())
	}
	q += ` ORDER BY f.created_at, f.id`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list drift: %w", err)
	}
	defer rows.Close()
	var out []models.Finding
	for rows.Next() {
		var f models.Finding
		var dim, crit, sev, attrs string
		if err := rows.Scan(&f.ID, &f.ScanID, &f.ProbeID, &dim, &crit, &f.Subject, &sev, &attrs, &f.Evidence, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan drift: %w", err)
		}
		f.DimensionHint = models.DimensionHint(dim)
		f.CriteriumHint = crit
		f.Severity = models.Severity(sev)
		if err := unmarshalAttrs(attrs, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// unmarshalAttrs decodes the JSON attributes blob into f.Attributes.
// Lives in this file because the existing scan-loading code path in
// sqlite.go inlines the equivalent call; keeping it private here
// avoids exporting a helper that would otherwise belong nowhere.
func unmarshalAttrs(raw string, f *models.Finding) error {
	if raw == "" {
		f.Attributes = map[string]any{}
		return nil
	}
	return jsonUnmarshalString(raw, &f.Attributes)
}
