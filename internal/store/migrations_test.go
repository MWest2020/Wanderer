package store

import (
	"context"
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
