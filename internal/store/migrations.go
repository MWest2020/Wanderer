package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// migration captures one numbered, up-only schema change. Migrations
// are applied in numeric order; once applied, a (version, applied_at)
// row is recorded in schema_migrations so the next Open knows which
// migrations have run.
type migration struct {
	Version int
	Name    string
	Up      string
}

// schemaInitial is the consolidated schema as it stood before the
// schema_migrations machinery existed: every CREATE TABLE the prior
// `const schema` block carried, including the source_modus column
// that used to be added by an idempotent ALTER TABLE.
const schemaInitial = `
CREATE TABLE IF NOT EXISTS targets (
  id         TEXT PRIMARY KEY,
  domain     TEXT NOT NULL UNIQUE,
  related    TEXT NOT NULL DEFAULT '[]',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scans (
  id         TEXT PRIMARY KEY,
  target_id  TEXT NOT NULL REFERENCES targets(id),
  started_at DATETIME NOT NULL,
  ended_at   DATETIME,
  status     TEXT NOT NULL,
  error      TEXT
);

CREATE TABLE IF NOT EXISTS findings (
  id             TEXT PRIMARY KEY,
  scan_id        TEXT NOT NULL REFERENCES scans(id),
  probe_id       TEXT NOT NULL,
  source_modus   TEXT NOT NULL DEFAULT 'perimeter',
  dimension_hint TEXT,
  criterium_hint TEXT,
  subject        TEXT NOT NULL,
  severity       TEXT NOT NULL,
  attributes     TEXT NOT NULL,
  evidence       BLOB,
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_findings_scan  ON findings(scan_id);
CREATE INDEX IF NOT EXISTS idx_findings_probe ON findings(probe_id);
CREATE INDEX IF NOT EXISTS idx_findings_modus ON findings(source_modus);

CREATE TABLE IF NOT EXISTS assessments (
  id         TEXT PRIMARY KEY,
  scan_id    TEXT NOT NULL REFERENCES scans(id),
  framework  TEXT NOT NULL,
  dimensions TEXT NOT NULL,
  report     TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assessments_scan ON assessments(scan_id);
`

// migrations is the canonical list. Adding a schema change means
// appending an entry here with the next free Version number. Never
// edit a previous entry — production databases at version N expect
// the historical SQL to be exactly what they ran.
var migrations = []migration{
	{Version: 1, Name: "initial_schema", Up: schemaInitial},
	{
		Version: 2,
		Name:    "ensure_source_modus_for_pre_v1_databases",
		Up: `-- A database created before the schema_migrations table existed
-- via the old ad-hoc ALTER TABLE pattern may already have the
-- source_modus column. Migration 001 declares the column with a
-- DEFAULT, so on a fresh database this is a no-op. On a database
-- that pre-existed migration 001 we still want to be idempotent;
-- SQLite does not support IF NOT EXISTS on ADD COLUMN, so we treat
-- the duplicate-column error as success in the runner below.
ALTER TABLE findings ADD COLUMN source_modus TEXT NOT NULL DEFAULT 'perimeter';
`,
	},
}

// runMigrations applies every migration whose Version is not already
// recorded in schema_migrations, in numeric order, each inside its
// own transaction so a partial failure rolls back cleanly.
func (s *Store) runMigrations(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at DATETIME NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}
	applied, err := loadAppliedVersions(ctx, s.db)
	if err != nil {
		return err
	}
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := applyOneMigration(ctx, s.db, m); err != nil {
			return fmt.Errorf("store: migration %d (%s): %w", m.Version, m.Name, err)
		}
	}
	return nil
}

func loadAppliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("store: read schema_migrations: %w", err)
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func applyOneMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, m.Up); err != nil {
		// Migration 002 documents why this duplicate-column branch
		// exists. Apart from that one historical case, every other
		// migration is expected to run cleanly on every database.
		if !isDuplicateColumnError(err) || m.Version != 2 {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.Version, m.Name, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "duplicate column name")
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	}())
}
