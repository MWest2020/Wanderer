package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func newOrgTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(memory)&"+t.Name())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestOrganisation_DefaultSeededByMigration(t *testing.T) {
	st := newOrgTestStore(t)
	got, err := st.GetOrganisationBySlug(context.Background(), models.DefaultOrganisationSlug)
	if err != nil {
		t.Fatalf("GetOrganisationBySlug(default): %v", err)
	}
	if got.ID != models.DefaultOrganisationID {
		t.Errorf("default org ID = %q, want %q", got.ID, models.DefaultOrganisationID)
	}
	if got.Name != "Default organisation" {
		t.Errorf("default org name = %q", got.Name)
	}
}

func TestOrganisation_UpsertNewAssignsID(t *testing.T) {
	st := newOrgTestStore(t)
	o := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if o.ID == "" {
		t.Errorf("ID not assigned")
	}
	if o.CreatedAt.IsZero() {
		t.Errorf("CreatedAt not set")
	}
}

func TestOrganisation_UpsertIdempotent(t *testing.T) {
	st := newOrgTestStore(t)
	o1 := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := st.UpsertOrganisation(context.Background(), o1); err != nil {
		t.Fatal(err)
	}
	o2 := &models.Organisation{Slug: "acme", Name: "ACME Updated"}
	if err := st.UpsertOrganisation(context.Background(), o2); err != nil {
		t.Fatal(err)
	}
	if o1.ID != o2.ID {
		t.Errorf("ID changed across upserts: %s vs %s", o1.ID, o2.ID)
	}
	got, _ := st.GetOrganisationBySlug(context.Background(), "acme")
	if got.Name != "ACME Updated" {
		t.Errorf("name not updated, got %q", got.Name)
	}
}

func TestOrganisation_GetUnknownSlugReturnsErrNotFound(t *testing.T) {
	st := newOrgTestStore(t)
	_, err := st.GetOrganisationBySlug(context.Background(), "nope")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestOrganisation_List_OrderedBySlug(t *testing.T) {
	st := newOrgTestStore(t)
	for _, s := range []string{"zeta", "alpha", "default-already-seeded", "mu"} {
		// Skip the seed; default already exists.
		if s == "default-already-seeded" {
			continue
		}
		o := &models.Organisation{Slug: s, Name: s}
		if err := st.UpsertOrganisation(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}
	list, err := st.ListOrganisations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 4 { // alpha, default, mu, zeta
		t.Fatalf("expected ≥ 4, got %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Slug > list[i].Slug {
			t.Errorf("not sorted: %s before %s", list[i-1].Slug, list[i].Slug)
		}
	}
}

func TestOrganisation_Rename_HappyPath(t *testing.T) {
	st := newOrgTestStore(t)
	if err := st.RenameOrganisation(context.Background(), "default", "conduction", "Conduction B.V."); err != nil {
		t.Fatalf("rename: %v", err)
	}
	got, err := st.GetOrganisationBySlug(context.Background(), "conduction")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Conduction B.V." {
		t.Errorf("name = %q", got.Name)
	}
	// Old slug must be gone.
	_, err = st.GetOrganisationBySlug(context.Background(), "default")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("old slug still resolves: %v", err)
	}
}

func TestOrganisation_Rename_UnknownReturnsErrNotFound(t *testing.T) {
	st := newOrgTestStore(t)
	err := st.RenameOrganisation(context.Background(), "nope", "newslug", "New")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestOrganisation_Rename_RejectsInvalidNewSlug(t *testing.T) {
	st := newOrgTestStore(t)
	err := st.RenameOrganisation(context.Background(), "default", "BadSlug", "Default")
	if err == nil {
		t.Fatal("invalid new slug must be rejected before SQL runs")
	}
}

func TestUpsertTarget_DefaultsToDefaultOrganisation(t *testing.T) {
	st := newOrgTestStore(t)
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if tgt.OrganisationID != models.DefaultOrganisationID {
		t.Errorf("OrganisationID = %q, want default", tgt.OrganisationID)
	}
}

func TestUpsertTarget_HonoursExplicitOrganisation(t *testing.T) {
	st := newOrgTestStore(t)
	o := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	tgt := &models.Target{Domain: "example.nl", OrganisationID: o.ID}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetTarget(context.Background(), tgt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrganisationID != o.ID {
		t.Errorf("OrganisationID = %q, want %q", got.OrganisationID, o.ID)
	}
}

func TestListTargetsByOrganisation_FiltersByOrgID(t *testing.T) {
	st := newOrgTestStore(t)
	acme := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), acme); err != nil {
		t.Fatal(err)
	}
	for _, dom := range []string{"a.example", "b.example"} {
		t1 := &models.Target{Domain: dom, OrganisationID: acme.ID}
		if err := st.UpsertTarget(context.Background(), t1); err != nil {
			t.Fatal(err)
		}
	}
	// Default-org target — must NOT show up under acme.
	if err := st.UpsertTarget(context.Background(), &models.Target{Domain: "elsewhere.example"}); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListTargetsByOrganisation(context.Background(), acme.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 targets, got %d", len(got))
	}
	for _, tg := range got {
		if tg.OrganisationID != acme.ID {
			t.Errorf("filter leaked: target %q has org %q", tg.Domain, tg.OrganisationID)
		}
	}
}
