package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runImport dispatches `wanderer import <source> <file>`. The only
// source today is "internetnl" (netnl-findings/v1); the verb form
// leaves room for other import sources later without a breaking CLI
// change.
func runImport(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: wanderer import internetnl [flags] <file>")
		return 2
	}
	switch args[0] {
	case "internetnl":
		return runImportInternetnl(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "wanderer import: unknown source %q (want internetnl)\n", args[0])
		return 2
	}
}

// runImportInternetnl executes `wanderer import internetnl <file>`
// and returns the intended process exit code. See specs/scanner/spec.md
// "Wanderer imports netnl findings files".
func runImportInternetnl(args []string) int {
	fs := flag.NewFlagSet("import internetnl", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	jsonLogs := fs.Bool("json-logs", false, "Emit logs as JSON (default text)")
	positional, err := parseFlagsInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "usage: wanderer import internetnl [flags] <file>")
		return 2
	}
	path := positional[0]

	logger := newLogger(*jsonLogs)
	slog.SetDefault(logger)

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: import: read %s: %v\n", path, err)
		return 1
	}
	sum := sha256.Sum256(data)
	fileHash := hex.EncodeToString(sum[:])

	file, err := scanner.ParseNetnlFindings(bytes.NewReader(data), logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: import: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := store.Open(ctx, "file:"+filepath.Clean(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	already, err := st.NetnlImportRecorded(ctx, fileHash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: import: %v\n", err)
		return 1
	}
	if already {
		fmt.Fprintf(os.Stdout, "wanderer: import: %s already imported (file unchanged), nothing to do\n", path)
		return 0
	}

	imported, skipped, err := importNetnlDomains(ctx, st, logger, file.Domains)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: import: %v\n", err)
		return 1
	}

	if err := st.RecordNetnlImport(ctx, fileHash); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: import: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "wanderer: import: %d target(s) imported, %d domain(s) skipped (unknown target)\n", imported, skipped)
	return 0
}

// importNetnlDomains matches each parsed domain entry to an existing
// target, groups entries by target (a batch file can carry both a web
// and a mail entry for the same domain), and persists one import-kind
// scan per matched target. Unknown domains are logged at WARN and
// skipped — spec.md "Unknown domains SHALL be logged at WARN and
// skipped".
func importNetnlDomains(ctx context.Context, st *store.Store, logger *slog.Logger, domains []scanner.NetnlDomain) (imported, skipped int, err error) {
	byDomain := map[string][]scanner.NetnlDomain{}
	var order []string
	for _, d := range domains {
		if _, ok := byDomain[d.Domain]; !ok {
			order = append(order, d.Domain)
		}
		byDomain[d.Domain] = append(byDomain[d.Domain], d)
	}

	for _, domain := range order {
		target, err := st.GetTargetByDomain(ctx, domain)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				logger.Warn("import.internetnl.unknown_domain", "domain", domain)
				skipped++
				continue
			}
			return imported, skipped, fmt.Errorf("lookup target %q: %w", domain, err)
		}

		scan, err := st.CreateScan(ctx, target.ID)
		if err != nil {
			return imported, skipped, fmt.Errorf("create scan for %q: %w", domain, err)
		}
		var findings []models.Finding
		for _, d := range byDomain[domain] {
			findings = append(findings, scanner.NetnlFindings(d)...)
		}
		if err := st.AppendFindings(ctx, scan.ID, findings); err != nil {
			return imported, skipped, fmt.Errorf("persist findings for %q: %w", domain, err)
		}
		if err := st.FinishScan(ctx, scan.ID, models.ScanStatusComplete, ""); err != nil {
			return imported, skipped, fmt.Errorf("finish scan for %q: %w", domain, err)
		}
		imported++
	}
	return imported, skipped, nil
}
