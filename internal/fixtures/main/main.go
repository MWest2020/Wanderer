// internal/fixtures/main is the `go run` entry point for the
// Playwright fixture seeder. The Makefile invokes it per
// scenario; humans should never run it directly in production.
//
// Usage:
//
//	go run ./internal/fixtures/main \
//	  --scenario baseline \
//	  --out /tmp/playwright-baseline.db
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/fixtures"
	"github.com/MWest2020/wanderer/internal/store"
)

func main() {
	scenario := flag.String("scenario", "", "scenario name (one of: "+strings.Join(scenarioNames(), ", ")+")")
	out := flag.String("out", "", "path to the SQLite DB to (re)create")
	flag.Parse()

	if *scenario == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "wanderer-fixture: --scenario and --out are required")
		flag.Usage()
		os.Exit(2)
	}

	build, ok := fixtures.Scenarios[*scenario]
	if !ok {
		fmt.Fprintf(os.Stderr, "wanderer-fixture: unknown scenario %q. Known: %s\n",
			*scenario, strings.Join(scenarioNames(), ", "))
		os.Exit(2)
	}

	// Recreate the DB from scratch so re-runs are idempotent and
	// schema-migration ordering is exercised on every invocation.
	if err := os.Remove(*out); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "wanderer-fixture: remove existing DB: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	st, err := store.Open(ctx, *out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer-fixture: open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := build(ctx, st); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer-fixture: build %s: %v\n", *scenario, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wanderer-fixture: %s → %s\n", *scenario, *out)
}

func scenarioNames() []string {
	names := make([]string, 0, len(fixtures.Scenarios))
	for k := range fixtures.Scenarios {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
