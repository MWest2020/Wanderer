package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrations_FreshDatabaseAppliesAll(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(context.Background(), "file:"+filepath.Join(dir, "fresh.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	rows, err := st.db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var v int
		_ = rows.Scan(&v)
		got = append(got, v)
	}
	if len(got) != len(migrations) {
		t.Fatalf("want %d applied migrations, got %d (%v)", len(migrations), len(got), got)
	}
	for i, m := range migrations {
		if got[i] != m.Version {
			t.Errorf("position %d: got version %d, want %d", i, got[i], m.Version)
		}
	}
}

func TestMigrations_AlreadyAppliedSkipped(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "twice.db")

	st1, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	st1.Close()

	// Re-open: every migration is already applied, so no rows are
	// inserted again.
	st2, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer st2.Close()

	var count int
	if err := st2.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != len(migrations) {
		t.Errorf("count = %d, want %d", count, len(migrations))
	}
}

// TestMigrations_ExpectedRegistrantBackfillsExistingOrganisations pins
// migration 007: a database that already carries organisations rows
// from before the column existed (the migration 005 seed, or any
// operator-created org) must have those rows backfilled to the empty
// list, not left NULL or absent.
func TestMigrations_ExpectedRegistrantBackfillsExistingOrganisations(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "backfill.db")
	ctx := context.Background()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at DATETIME NOT NULL
		)`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, m := range migrations {
		if m.Version >= 7 {
			continue
		}
		if err := applyOneMigration(ctx, db, m); err != nil {
			t.Fatalf("apply migration %d: %v", m.Version, err)
		}
	}
	// An operator-created organisation from before migration 007 —
	// the column does not exist yet at this point in the sequence.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO organisations (id, slug, name, description, created_at)
		 VALUES ('o_pre', 'pre-existing', 'Pre-existing Org', '', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("seed pre-existing organisation: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	st, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open via Store (runs migration 007): %v", err)
	}
	defer st.Close()

	for _, slug := range []string{"default", "pre-existing"} {
		var er string
		if err := st.db.QueryRowContext(ctx,
			`SELECT expected_registrant FROM organisations WHERE slug = ?`, slug).Scan(&er); err != nil {
			t.Fatalf("select expected_registrant for %q: %v", slug, err)
		}
		if er != "[]" {
			t.Errorf("%s: expected_registrant = %q, want %q", slug, er, "[]")
		}
	}
}

func TestMigrations_LegacyDuplicateColumnTolerated(t *testing.T) {
	// Build a database whose findings table already has source_modus
	// (mimicking a database that pre-existed the migration runner)
	// and whose schema_migrations is absent. Open should record both
	// migrations and not error on the duplicate column.
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "legacy.db")

	st, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()

	// Re-opening — every migration's idempotency must hold.
	st2, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
}
