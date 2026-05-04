package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"go.yaml.in/yaml/v2"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runOrg dispatches `wanderer org <verb> [args]`. The four verbs
// cover the full v1 surface: add, list, show, rename. `add` accepts
// either explicit flags for a single org or `--from-yaml <path>`
// for bulk seeding.
func runOrg(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "wanderer org: usage: wanderer org <add|list|show|rename> [flags]")
		return 2
	}
	verb := args[0]
	rest := args[1:]
	switch verb {
	case "add":
		return runOrgAdd(rest)
	case "list":
		return runOrgList(rest)
	case "show":
		return runOrgShow(rest)
	case "rename":
		return runOrgRename(rest)
	default:
		fmt.Fprintf(os.Stderr, "wanderer org: unknown verb %q\n", verb)
		return 2
	}
}

func runOrgAdd(args []string) int {
	fs := flag.NewFlagSet("org add", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	slug := fs.String("slug", "", "Organisation slug (lowercase, 2..40 chars, hyphens allowed)")
	name := fs.String("name", "", "Display name")
	desc := fs.String("description", "", "Optional free-text description")
	fromYAML := fs.String("from-yaml", "", "Bulk-seed organisations from a YAML file (idempotent)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := openOrgStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	ctx := context.Background()

	if *fromYAML != "" {
		if *slug != "" || *name != "" || *desc != "" {
			fmt.Fprintln(os.Stderr, "wanderer org add: --from-yaml is mutually exclusive with --slug/--name/--description")
			return 2
		}
		return runOrgAddFromYAML(ctx, st, *fromYAML)
	}
	if *slug == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "wanderer org add: --slug and --name are required (or use --from-yaml)")
		return 2
	}
	o := &models.Organisation{Slug: *slug, Name: *name, Description: *desc}
	if err := st.UpsertOrganisation(ctx, o); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer org add: %v\n", err)
		return 1
	}
	fmt.Printf("organisation %s (%s) saved\n", o.Slug, o.ID)
	return 0
}

// orgsYAMLFile is the shape `wanderer org add --from-yaml` reads.
// Mirrors the simplest possible operator-facing structure: one
// top-level key with a list of organisation entries.
type orgsYAMLFile struct {
	Organisations []orgYAMLEntry `yaml:"organisations"`
}

type orgYAMLEntry struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

func runOrgAddFromYAML(ctx context.Context, st *store.Store, path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer org add: read %s: %v\n", path, err)
		return 1
	}
	var f orgsYAMLFile
	if err := yaml.UnmarshalStrict(data, &f); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer org add: parse %s: %v\n", path, err)
		return 1
	}
	if len(f.Organisations) == 0 {
		fmt.Fprintf(os.Stderr, "wanderer org add: %s contains no organisations\n", path)
		return 1
	}
	for i, e := range f.Organisations {
		o := &models.Organisation{Slug: e.Slug, Name: e.Name, Description: e.Description}
		if err := st.UpsertOrganisation(ctx, o); err != nil {
			fmt.Fprintf(os.Stderr, "wanderer org add: entry %d (%s): %v\n", i, e.Slug, err)
			return 1
		}
		fmt.Printf("organisation %s (%s) saved\n", o.Slug, o.ID)
	}
	return 0
}

func runOrgList(args []string) int {
	fs := flag.NewFlagSet("org list", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := openOrgStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	ctx := context.Background()
	orgs, err := st.ListOrganisations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer org list: %v\n", err)
		return 1
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tNAME\tTARGETS\tCREATED")
	for _, o := range orgs {
		targets, err := st.ListTargetsByOrganisation(ctx, o.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer org list: %v\n", err)
			return 1
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", o.Slug, o.Name, len(targets), o.CreatedAt.UTC().Format("2006-01-02"))
	}
	tw.Flush()
	return 0
}

func runOrgShow(args []string) int {
	fs := flag.NewFlagSet("org show", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "wanderer org show: usage: wanderer org show <slug>")
		return 2
	}
	slug := fs.Arg(0)
	st, err := openOrgStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	ctx := context.Background()
	o, err := st.GetOrganisationBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "wanderer org show: no organisation with slug %q\n", slug)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer org show: %v\n", err)
		return 1
	}
	fmt.Printf("Slug:        %s\n", o.Slug)
	fmt.Printf("Name:        %s\n", o.Name)
	fmt.Printf("ID:          %s\n", o.ID)
	if o.Description != "" {
		fmt.Printf("Description: %s\n", o.Description)
	}
	fmt.Printf("Created:     %s\n", o.CreatedAt.UTC().Format("2006-01-02 15:04 UTC"))
	targets, err := st.ListTargetsByOrganisation(ctx, o.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer org show: list targets: %v\n", err)
		return 1
	}
	fmt.Printf("Targets:     %d\n", len(targets))
	for _, t := range targets {
		fmt.Printf("  - %s (%s)\n", t.Domain, t.Kind)
	}
	return 0
}

func runOrgRename(args []string) int {
	fs := flag.NewFlagSet("org rename", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	oldSlug := fs.String("slug", "", "Current slug (e.g. 'default')")
	newSlug := fs.String("new-slug", "", "Replacement slug")
	newName := fs.String("name", "", "Replacement display name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *oldSlug == "" || *newSlug == "" || *newName == "" {
		fmt.Fprintln(os.Stderr, "wanderer org rename: --slug, --new-slug, and --name are all required")
		return 2
	}
	st, err := openOrgStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.RenameOrganisation(ctx, *oldSlug, *newSlug, *newName); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "wanderer org rename: no organisation with slug %q\n", *oldSlug)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer org rename: %v\n", err)
		return 1
	}
	fmt.Printf("organisation %s renamed to %s (%q)\n", *oldSlug, *newSlug, *newName)
	return 0
}

func openOrgStore(path string) (*store.Store, error) {
	st, err := store.Open(context.Background(), "file:"+filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("wanderer org: open store: %w", err)
	}
	return st, nil
}
