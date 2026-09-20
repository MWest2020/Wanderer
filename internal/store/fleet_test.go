package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestAddFleetDomain_CreatesUnscannedTarget(t *testing.T) {
	st := newOrgTestStore(t)
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("AddFleetDomain: %v", err)
	}
	if tgt.ID == "" {
		t.Errorf("ID not assigned")
	}
	if tgt.OrganisationID != models.DefaultOrganisationID {
		t.Errorf("OrganisationID = %q, want default", tgt.OrganisationID)
	}
	scans, err := st.ListScans(context.Background(), store.Selectors{})
	if err != nil {
		t.Fatal(err)
	}
	if len(scans) != 0 {
		t.Errorf("expected zero scans for a fleet-only add, got %d", len(scans))
	}
}

func TestAddFleetDomain_RejectsDifferentOrganisation(t *testing.T) {
	st := newOrgTestStore(t)
	acme := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), acme); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFleetDomain(context.Background(), acme.ID, "voorbeeld.nl"); err == nil {
		t.Fatal("expected an error adding a domain already owned by another organisation")
	}
}

func TestRemoveFleetDomain_HidesFromListButKeepsHistory(t *testing.T) {
	st := newOrgTestStore(t)
	tgt, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RemoveFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatalf("RemoveFleetDomain: %v", err)
	}
	list, err := st.ListFleetDomains(context.Background(), models.DefaultOrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tg := range list {
		if tg.Domain == "voorbeeld.nl" {
			t.Errorf("removed domain still listed in fleet")
		}
	}
	// The scan is still reachable directly.
	got, err := st.GetScan(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("scan history lost after removal: %v", err)
	}
	if got.TargetID != tgt.ID {
		t.Errorf("scan target mismatch after removal")
	}
}

func TestRemoveFleetDomain_UnknownReturnsErrNotFound(t *testing.T) {
	st := newOrgTestStore(t)
	err := st.RemoveFleetDomain(context.Background(), models.DefaultOrganisationID, "nope.nl")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestAddFleetDomain_ReAddUndoesRemoval(t *testing.T) {
	st := newOrgTestStore(t)
	first, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RemoveFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl"); err != nil {
		t.Fatal(err)
	}
	again, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "voorbeeld.nl")
	if err != nil {
		t.Fatalf("re-add: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("re-add created a new Target (ID %q), want the same row (ID %q)", again.ID, first.ID)
	}
	list, err := st.ListFleetDomains(context.Background(), models.DefaultOrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tg := range list {
		if tg.Domain == "voorbeeld.nl" {
			found = true
		}
	}
	if !found {
		t.Errorf("re-added domain not back in the fleet listing")
	}
}

func TestListFleetDomains_IncludesUnscannedAndFiltersByOrg(t *testing.T) {
	st := newOrgTestStore(t)
	acme := &models.Organisation{Slug: "acme", Name: "ACME"}
	if err := st.UpsertOrganisation(context.Background(), acme); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFleetDomain(context.Background(), acme.ID, "a.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddFleetDomain(context.Background(), models.DefaultOrganisationID, "elsewhere.example"); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListFleetDomains(context.Background(), acme.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Domain != "a.example" {
		t.Errorf("ListFleetDomains(acme) = %v, want [a.example]", got)
	}
}
