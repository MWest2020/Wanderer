package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/drift"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runDiff executes `wanderer diff <scan-a> <scan-b>` and prints a
// markdown report of the drift Findings without persisting them.
func runDiff(args []string) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	positional, err := parseFlagsInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(positional) != 2 {
		fmt.Fprintln(os.Stderr, "usage: wanderer diff <scan-a> <scan-b>")
		return 2
	}
	scanA, scanB := positional[0], positional[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.Open(ctx, "file:"+filepath.Clean(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	a, err := st.GetScan(ctx, scanA)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "wanderer: scan not found: %s\n", scanA)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer: get %s: %v\n", scanA, err)
		return 1
	}
	b, err := st.GetScan(ctx, scanB)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "wanderer: scan not found: %s\n", scanB)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer: get %s: %v\n", scanB, err)
		return 1
	}

	findings := drift.Diff(a, b)
	renderDiffMarkdown(os.Stdout, a, b, findings)
	return 0
}

func renderDiffMarkdown(w *os.File, a, b *models.Scan, findings []models.Finding) {
	fmt.Fprintf(w, "# Wanderer drift — %s → %s\n\n", a.ID, b.ID)
	fmt.Fprintf(w, "Previous: %s (started %s)\n", a.ID, a.StartedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(w, "Current:  %s (started %s)\n\n", b.ID, b.StartedAt.UTC().Format(time.RFC3339))

	if len(findings) == 0 {
		fmt.Fprintln(w, "_No drift Findings produced._")
		return
	}
	// Stable rendering: order by ProbeID then Subject.
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].ProbeID != findings[j].ProbeID {
			return findings[i].ProbeID < findings[j].ProbeID
		}
		return findings[i].Subject < findings[j].Subject
	})
	for _, f := range findings {
		fmt.Fprintf(w, "## %s — %s\n", f.ProbeID, f.Severity)
		if f.Subject != "" {
			fmt.Fprintf(w, "Subject: %s\n", f.Subject)
		}
		if f.DimensionHint != "" {
			fmt.Fprintf(w, "Dimension: %s\n", f.DimensionHint)
		}
		// Stable attribute key order.
		keys := make([]string, 0, len(f.Attributes))
		for k := range f.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "- %s: %v\n", k, prettyValue(f.Attributes[k]))
		}
		fmt.Fprintln(w)
	}
}

func prettyValue(v any) string {
	switch x := v.(type) {
	case []string:
		return strings.Join(x, ", ")
	case []any:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, fmt.Sprint(p))
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}
