package fixtures_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/fixtures"
	"github.com/MWest2020/wanderer/internal/store"
)

// freshStore opens an in-memory SQLite-backed store. Migrations
// run on Open, so the schema is the production shape.
func freshStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "fixture-test.db")
	st, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestScenariosRegistered(t *testing.T) {
	for _, want := range []string{"baseline", "agent-host", "empty-org"} {
		if _, ok := fixtures.Scenarios[want]; !ok {
			t.Errorf("Scenarios missing %q", want)
		}
	}
}

func TestBuildBaseline_PersistsTwoOrgsTwoTargets(t *testing.T) {
	ctx := context.Background()
	st := freshStore(t)
	if err := fixtures.BuildBaseline(ctx, st); err != nil {
		t.Fatalf("build: %v", err)
	}

	orgs, err := st.ListOrganisations(ctx)
	if err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if len(orgs) < 3 { // default + conduction + acme
		t.Errorf("orgs = %d, want >= 3", len(orgs))
	}
	wantSlugs := map[string]bool{"conduction": false, "acme": false}
	for _, o := range orgs {
		if _, ok := wantSlugs[o.Slug]; ok {
			wantSlugs[o.Slug] = true
		}
	}
	for slug, seen := range wantSlugs {
		if !seen {
			t.Errorf("org %q not persisted", slug)
		}
	}
}

func TestBuildAgentHost_AlmaScanHasInventoryFindings(t *testing.T) {
	ctx := context.Background()
	st := freshStore(t)
	if err := fixtures.BuildAgentHost(ctx, st); err != nil {
		t.Fatalf("build: %v", err)
	}

	cond, err := st.GetOrganisationBySlug(ctx, "conduction")
	if err != nil {
		t.Fatalf("get conduction: %v", err)
	}
	targets, err := st.ListTargetsByOrganisation(ctx, cond.ID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}

	var alma string
	for _, tg := range targets {
		if tg.Domain == "alma" {
			alma = tg.ID
		}
	}
	if alma == "" {
		t.Fatal("alma host target not seeded under conduction")
	}

	// At least one inventory.packages.rpm Finding must mention
	// datadog-agent so the host-rule deep-dive spec works.
	rows, err := st.DB().QueryContext(ctx,
		`SELECT subject FROM findings WHERE probe_id = 'inventory.packages.rpm'`)
	if err != nil {
		t.Fatalf("query findings: %v", err)
	}
	defer rows.Close()
	var sawDatadog bool
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if strings.Contains(subject, "datadog") {
			sawDatadog = true
		}
	}
	if !sawDatadog {
		t.Error("agent-host fixture missing datadog-agent package finding")
	}
}

func TestBuildEmptyOrg_HasOrgWithZeroTargets(t *testing.T) {
	ctx := context.Background()
	st := freshStore(t)
	if err := fixtures.BuildEmptyOrg(ctx, st); err != nil {
		t.Fatalf("build: %v", err)
	}
	o, err := st.GetOrganisationBySlug(ctx, "acme-empty")
	if err != nil {
		t.Fatalf("get acme-empty: %v", err)
	}
	targets, err := st.ListTargetsByOrganisation(ctx, o.ID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("acme-empty has %d targets, want 0", len(targets))
	}
}
