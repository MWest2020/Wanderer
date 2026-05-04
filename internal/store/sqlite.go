// Package store persists targets, scans, and findings to SQLite.
//
// modernc.org/sqlite is pure Go (no CGo), so the binary builds
// everywhere and the database file is trivially auditable on disk. A
// future migration to PostgreSQL is a pg_dump-equivalent away — this
// package keeps its surface small to make that move cheap.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"crypto/rand"
	"math/big"

	"github.com/MWest2020/wanderer/pkg/models"

	_ "modernc.org/sqlite"
)

// Store wraps a *sql.DB with Wanderer-specific operations.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite database at dsn and runs migrations.
// A typical dsn for on-disk is "file:wanderer.db?cache=shared", and for
// tests "file::memory:?cache=shared".
func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	// SQLite can only handle one writer at a time; serialise writes.
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying handle for callers that need custom queries
// (tests, operator tools). Application code should prefer the typed
// methods on Store.
func (s *Store) DB() *sql.DB { return s.db }

// migrate dispatches to the numbered migration runner. The schema
// itself lives in migrations.go.
func (s *Store) migrate(ctx context.Context) error {
	return s.runMigrations(ctx)
}

// newID returns a time-sortable identifier. We use a simplified scheme
// (timestamp + random suffix, base36) to avoid pulling in a ULID
// dependency for a value that only needs to be unique per row.
func newID(prefix string) string {
	ts := time.Now().UTC().UnixMilli()
	n, _ := rand.Int(rand.Reader, big.NewInt(1<<40))
	return fmt.Sprintf("%s_%013d_%010s", prefix, ts, n.Text(36))
}

// UpsertTarget inserts a new target or returns the existing one by
// domain. The domain is normalised by Target.Validate first; the
// Kind column round-trips so an existing host-mode Target keeps its
// kind on subsequent upserts. An empty OrganisationID falls back
// to models.DefaultOrganisationID so callers that pre-date the
// organisation pivot keep working unchanged.
func (s *Store) UpsertTarget(ctx context.Context, t *models.Target) error {
	if err := t.Validate(); err != nil {
		return err
	}
	if t.OrganisationID == "" {
		t.OrganisationID = models.DefaultOrganisationID
	}
	rel, err := json.Marshal(t.Related)
	if err != nil {
		return fmt.Errorf("store: marshal related: %w", err)
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, kind, organisation_id FROM targets WHERE domain = ?`, t.Domain)
	var id string
	var createdAt time.Time
	var kind, orgID string
	switch err := row.Scan(&id, &createdAt, &kind, &orgID); {
	case errors.Is(err, sql.ErrNoRows):
		t.ID = newID("t")
		t.CreatedAt = time.Now().UTC()
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO targets (id, domain, related, kind, organisation_id, created_at) VALUES (?,?,?,?,?,?)`,
			t.ID, t.Domain, string(rel), string(t.Kind), t.OrganisationID, t.CreatedAt)
		if err != nil {
			return fmt.Errorf("store: insert target: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("store: lookup target: %w", err)
	}
	t.ID = id
	t.CreatedAt = createdAt
	t.Kind = models.TargetKind(kind)
	t.OrganisationID = orgID
	return nil
}

// GetTarget returns a Target by its ID. Useful for callers that want
// the full row, including Kind, without re-deriving it from a domain
// lookup.
func (s *Store) GetTarget(ctx context.Context, id string) (*models.Target, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, domain, related, COALESCE(kind,'domain'), COALESCE(organisation_id,'o_default'), created_at FROM targets WHERE id = ?`, id)
	t := &models.Target{}
	var rel, kind, orgID string
	if err := row.Scan(&t.ID, &t.Domain, &rel, &kind, &orgID, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select target: %w", err)
	}
	if rel != "" {
		if err := json.Unmarshal([]byte(rel), &t.Related); err != nil {
			return nil, fmt.Errorf("store: unmarshal related: %w", err)
		}
	}
	t.Kind = models.TargetKind(kind)
	t.OrganisationID = orgID
	return t, nil
}

// CreateScan records a new scan in the "running" state.
func (s *Store) CreateScan(ctx context.Context, targetID string) (*models.Scan, error) {
	now := time.Now().UTC()
	sc := &models.Scan{
		ID:        newID("s"),
		TargetID:  targetID,
		StartedAt: now,
		Status:    models.ScanStatusRunning,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO scans (id, target_id, started_at, status) VALUES (?,?,?,?)`,
		sc.ID, sc.TargetID, sc.StartedAt, string(sc.Status))
	if err != nil {
		return nil, fmt.Errorf("store: insert scan: %w", err)
	}
	return sc, nil
}

// FinishScan updates a scan's terminal status and optional error.
func (s *Store) FinishScan(ctx context.Context, scanID string, status models.ScanStatus, scanErr string) error {
	if !status.Valid() {
		return fmt.Errorf("store: invalid scan status %q", status)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE scans SET ended_at = ?, status = ?, error = ? WHERE id = ?`,
		time.Now().UTC(), string(status), scanErr, scanID)
	if err != nil {
		return fmt.Errorf("store: finish scan: %w", err)
	}
	return nil
}

// AppendFindings persists a batch of findings for a scan. It assigns
// IDs and timestamps in place. Batches run in a single transaction so a
// replay on failure never leaves duplicates for the same scan unless
// explicitly re-invoked.
func (s *Store) AppendFindings(ctx context.Context, scanID string, findings []models.Finding) error {
	if len(findings) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO findings (id, scan_id, probe_id, source_modus, dimension_hint, criterium_hint, subject, severity, attributes, evidence, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("store: prepare: %w", err)
	}
	defer stmt.Close()
	now := time.Now().UTC()
	for i := range findings {
		f := &findings[i]
		f.ScanID = scanID
		if err := f.Validate(); err != nil {
			return fmt.Errorf("store: finding[%d]: %w", i, err)
		}
		if f.ID == "" {
			f.ID = newID("f")
		}
		f.CreatedAt = now
		attrs, err := json.Marshal(f.Attributes)
		if err != nil {
			return fmt.Errorf("store: marshal attributes: %w", err)
		}
		modus := f.SourceModus
		if modus == "" {
			modus = models.SourceModusPerimeter
		}
		_, err = stmt.ExecContext(ctx,
			f.ID, f.ScanID, f.ProbeID, string(modus),
			string(f.DimensionHint), f.CriteriumHint,
			f.Subject, string(f.Severity),
			string(attrs), f.Evidence, f.CreatedAt)
		if err != nil {
			return fmt.Errorf("store: insert finding: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// GetScan returns a scan and all its findings by scan ID.
func (s *Store) GetScan(ctx context.Context, scanID string) (*models.Scan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_id, started_at, ended_at, status, COALESCE(error,'') FROM scans WHERE id = ?`, scanID)
	sc := &models.Scan{}
	var endedAt sql.NullTime
	var status string
	if err := row.Scan(&sc.ID, &sc.TargetID, &sc.StartedAt, &endedAt, &status, &sc.Error); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select scan: %w", err)
	}
	sc.Status = models.ScanStatus(status)
	if endedAt.Valid {
		t := endedAt.Time
		sc.EndedAt = &t
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, probe_id, COALESCE(source_modus,'perimeter'),
		        COALESCE(dimension_hint,''), COALESCE(criterium_hint,''),
		        subject, severity, attributes, evidence, created_at
		 FROM findings WHERE scan_id = ? ORDER BY rowid`, scanID)
	if err != nil {
		return nil, fmt.Errorf("store: select findings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f models.Finding
		var modus, dim, crit, sev, attrs string
		if err := rows.Scan(&f.ID, &f.ProbeID, &modus, &dim, &crit, &f.Subject, &sev, &attrs, &f.Evidence, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan finding: %w", err)
		}
		f.ScanID = scanID
		f.SourceModus = models.SourceModus(modus)
		f.DimensionHint = models.DimensionHint(dim)
		f.CriteriumHint = crit
		f.Severity = models.Severity(sev)
		if err := json.Unmarshal([]byte(attrs), &f.Attributes); err != nil {
			return nil, fmt.Errorf("store: unmarshal attributes: %w", err)
		}
		sc.Findings = append(sc.Findings, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows: %w", err)
	}
	return sc, nil
}

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("store: not found")

// jsonUnmarshalString decodes raw into v. Wraps json.Unmarshal so
// callers do not have to allocate a []byte() for short-lived strings.
func jsonUnmarshalString(raw string, v any) error {
	return json.Unmarshal([]byte(raw), v)
}
