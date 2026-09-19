package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
)

func TestRunOrgAdd_ExpectedRegistrantFlagIsRepeatable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")

	rc := runOrgAdd([]string{
		"--db", dbPath,
		"--slug", "acme",
		"--name", "ACME B.V.",
		"--expected-registrant", "ACME B.V.",
		"--expected-registrant", "Stichting ACME",
	})
	if rc != 0 {
		t.Fatalf("runOrgAdd exit = %d, want 0", rc)
	}

	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	got, err := st.GetOrganisationBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetOrganisationBySlug: %v", err)
	}
	want := []string{"ACME B.V.", "Stichting ACME"}
	if len(got.ExpectedRegistrant) != len(want) {
		t.Fatalf("ExpectedRegistrant = %v, want %v", got.ExpectedRegistrant, want)
	}
	for i, name := range want {
		if got.ExpectedRegistrant[i] != name {
			t.Errorf("ExpectedRegistrant[%d] = %q, want %q", i, got.ExpectedRegistrant[i], name)
		}
	}
}

// TestRunOrgAdd_WithoutFlagDoesNotWipeExisting exercises the CLI path
// for the "upsert without names must not wipe stored names" rule: a
// second `org add` that updates only --name (no --expected-registrant)
// must leave the previously declared names intact.
func TestRunOrgAdd_WithoutFlagDoesNotWipeExisting(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")

	if rc := runOrgAdd([]string{
		"--db", dbPath,
		"--slug", "acme",
		"--name", "ACME B.V.",
		"--expected-registrant", "ACME B.V.",
	}); rc != 0 {
		t.Fatalf("first runOrgAdd exit = %d, want 0", rc)
	}
	if rc := runOrgAdd([]string{
		"--db", dbPath,
		"--slug", "acme",
		"--name", "ACME Updated B.V.",
	}); rc != 0 {
		t.Fatalf("second runOrgAdd exit = %d, want 0", rc)
	}

	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	got, err := st.GetOrganisationBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetOrganisationBySlug: %v", err)
	}
	if got.Name != "ACME Updated B.V." {
		t.Errorf("Name = %q, want updated name", got.Name)
	}
	if len(got.ExpectedRegistrant) != 1 || got.ExpectedRegistrant[0] != "ACME B.V." {
		t.Errorf("ExpectedRegistrant = %v, want [ACME B.V.] (not wiped)", got.ExpectedRegistrant)
	}
}
