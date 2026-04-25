package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/MWest2020/wanderer/internal/export"
	"github.com/MWest2020/wanderer/internal/store"
)

// runExport executes `wanderer export <resource> [flags]`.
func runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	format := fs.String("format", "csv", "Output format: csv | jsonl")
	out := fs.String("o", "", "Output file (default stdout)")
	scanID := fs.String("scan", "", "Filter to one scan ID")
	probe := fs.String("probe", "", "Filter findings whose probe_id starts with this prefix")
	dimension := fs.String("dimension", "", "Filter findings/assessments by DICTU dimension")
	since := fs.String("since", "", "Lower bound on created_at (RFC 3339)")
	until := fs.String("until", "", "Upper bound on created_at (RFC 3339)")
	includeEvidence := fs.Bool("include-evidence", true, "Include evidence in JSONL output (base64)")
	positional, err := parseFlagsInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "usage: wanderer export <findings|scans|assessments> [flags]")
		return 2
	}
	resource := positional[0]

	sel := store.Selectors{
		ScanID:    *scanID,
		ProbePref: *probe,
		Dimension: *dimension,
	}
	if *since != "" {
		t, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: --since: %v\n", err)
			return 2
		}
		sel.Since = t
	}
	if *until != "" {
		t, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: --until: %v\n", err)
			return 2
		}
		sel.Until = t
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.Open(ctx, "file:"+filepath.Clean(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	w, closer, code := openOutput(*out)
	if code != 0 {
		return code
	}
	defer closer()

	switch *format {
	case "csv":
		switch resource {
		case "findings":
			err = export.WriteFindingsCSV(ctx, w, st, sel)
		case "scans":
			err = export.WriteScansCSV(ctx, w, st, sel)
		case "assessments":
			err = export.WriteAssessmentsCSV(ctx, w, st, sel)
		default:
			fmt.Fprintf(os.Stderr, "wanderer: unknown resource %q\n", resource)
			return 2
		}
	case "jsonl":
		switch resource {
		case "findings":
			err = export.WriteFindingsJSONL(ctx, w, st, sel, *includeEvidence)
		case "scans":
			err = export.WriteScansJSONL(ctx, w, st, sel)
		case "assessments":
			err = export.WriteAssessmentsJSONL(ctx, w, st, sel)
		default:
			fmt.Fprintf(os.Stderr, "wanderer: unknown resource %q\n", resource)
			return 2
		}
	default:
		fmt.Fprintf(os.Stderr, "wanderer: unknown --format %q (want csv|jsonl)\n", *format)
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: export: %v\n", err)
		return 1
	}
	return 0
}

// openOutput returns the writer the export should send its bytes to,
// a closer to call when done, and a non-zero exit code on failure.
// When path is empty stdout is used and the closer is a no-op.
func openOutput(path string) (io.Writer, func(), int) {
	if path == "" {
		return os.Stdout, func() {}, 0
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open output: %v\n", err)
		return nil, func() {}, 1
	}
	return f, func() { _ = f.Close() }, 0
}
