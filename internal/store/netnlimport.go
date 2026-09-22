package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// GetTargetByDomain returns the Target matching domain exactly, or
// ErrNotFound if no target is registered under that domain.
// targets.domain is globally unique (see AddFleetDomain's doc
// comment), so unlike UpsertTarget this never creates a row — it is
// the "match against existing targets, never invent one" lookup the
// netnl importer needs (spec.md "Unknown domains SHALL be logged at
// WARN and skipped").
func (s *Store) GetTargetByDomain(ctx context.Context, domain string) (*models.Target, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id FROM targets WHERE domain = ?`, domain)
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: lookup target by domain: %w", err)
	}
	return s.GetTarget(ctx, id)
}

// NetnlImportRecorded reports whether a netnl-findings file with this
// exact sha256 hash has already been imported, so the caller can skip
// re-persisting an identical file (spec.md "Re-importing an identical
// file SHALL be idempotent").
func (s *Store) NetnlImportRecorded(ctx context.Context, fileHash string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM netnl_imports WHERE file_hash = ?)`, fileHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: check netnl import: %w", err)
	}
	return exists, nil
}

// RecordNetnlImport marks fileHash as imported. Call this once the
// findings it describes have been persisted, so a later re-run of the
// same file is recognised as a no-op.
func (s *Store) RecordNetnlImport(ctx context.Context, fileHash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO netnl_imports (file_hash, imported_at) VALUES (?, ?)`,
		fileHash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("store: record netnl import: %w", err)
	}
	return nil
}

// LatestScanByModus returns the most recently started scan for
// targetID that carries at least one Finding with the given
// SourceModus, or ErrNotFound if none exists. An import scan and a
// perimeter scan for the same target coexist independently (neither
// overwrites the other — spec.md "Imports and perimeter scans do not
// erase each other"); this is how a consumer reads each kind's
// newest scan on its own timeline instead of the target's overall
// newest scan, which might be of the other kind.
func (s *Store) LatestScanByModus(ctx context.Context, targetID string, modus models.SourceModus) (*models.Scan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT s.id FROM scans s
		 JOIN findings f ON f.scan_id = s.id
		 WHERE s.target_id = ? AND f.source_modus = ?
		 ORDER BY s.started_at DESC, s.id DESC
		 LIMIT 1`,
		targetID, string(modus))
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: latest scan by modus: %w", err)
	}
	return s.GetScan(ctx, id)
}
