package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/dictu"
	"github.com/MWest2020/wanderer/internal/assessor/eucsf"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runAssess executes `wanderer assess <scan-id>` and returns the
// intended process exit code.
func runAssess(args []string) int {
	fs := flag.NewFlagSet("assess", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	format := fs.String("format", "text", "Output format: text | markdown | json")
	framework := fs.String("framework", "dictu", "Rule pack: dictu | eucsf | both")
	persist := fs.Bool("persist", true, "Persist the Assessment to the store")
	positional, err := parseFlagsInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "usage: wanderer assess [flags] <scan-id>")
		return 2
	}
	scanID := positional[0]

	frameworks, err := selectedFrameworks(*framework)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: %v\n", err)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := store.Open(ctx, "file:"+filepath.Clean(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	scan, err := st.GetScan(ctx, scanID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "wanderer: scan not found")
			return 1
		}
		fmt.Fprintf(os.Stderr, "wanderer: get scan: %v\n", err)
		return 1
	}

	subject := subjectOfScan(scan, st, ctx)
	for _, fw := range frameworks {
		rules := rulesForFramework(fw)
		a := &models.Assessment{
			ScanID:     scan.ID,
			Framework:  string(fw),
			Dimensions: assessor.Assess(scan.Findings, rules),
		}
		var reportBuf = &strBuf{}
		if err := assessor.RenderMarkdown(reportBuf, a, rules, subject); err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: render: %v\n", err)
			return 1
		}
		a.Report = reportBuf.String()

		if *persist {
			if err := st.CreateAssessment(ctx, a); err != nil {
				fmt.Fprintf(os.Stderr, "wanderer: persist: %v\n", err)
				return 1
			}
		}

		switch *format {
		case "markdown":
			err = assessor.RenderMarkdown(os.Stdout, a, rules, subject)
		case "json":
			err = assessor.RenderJSON(os.Stdout, a)
		case "text", "":
			err = assessor.RenderText(os.Stdout, a, rules, subject)
		default:
			fmt.Fprintf(os.Stderr, "wanderer: unknown --format %q (want text|markdown|json)\n", *format)
			return 2
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: render: %v\n", err)
			return 1
		}
		// Separator between frameworks for readability when --framework=both.
		if len(frameworks) > 1 {
			fmt.Fprintln(os.Stdout)
		}
	}
	return 0
}

// selectedFrameworks parses the --framework flag value and returns
// the ordered list of frameworks to assess.
func selectedFrameworks(flagValue string) ([]models.Framework, error) {
	switch strings.ToLower(strings.TrimSpace(flagValue)) {
	case "", "dictu":
		return []models.Framework{models.FrameworkDICTU}, nil
	case "eucsf", "seal":
		return []models.Framework{models.FrameworkEUCSF}, nil
	case "both":
		return []models.Framework{models.FrameworkDICTU, models.FrameworkEUCSF}, nil
	}
	return nil, fmt.Errorf("--framework: unknown value %q (want dictu|eucsf|both)", flagValue)
}

// rulesForFramework dispatches to the right rule pack.
func rulesForFramework(fw models.Framework) []assessor.Rule {
	switch fw {
	case models.FrameworkEUCSF:
		return eucsf.DefaultRules()
	default:
		return dictu.DefaultRules()
	}
}

// strBuf is a minimal io.Writer-backed string builder so we can
// reuse RenderMarkdown for both the persisted Report field and
// standalone CLI output.
type strBuf struct{ data []byte }

func (b *strBuf) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}
func (b *strBuf) String() string { return string(b.data) }

// subjectOfScan returns the domain of the scan's target, falling back
// to the scan ID if the target lookup fails. The rendered report uses
// this for its human-readable heading.
func subjectOfScan(scan *models.Scan, st *store.Store, ctx context.Context) string {
	row := st.DB().QueryRowContext(ctx, `SELECT domain FROM targets WHERE id = ?`, scan.TargetID)
	var domain string
	if err := row.Scan(&domain); err == nil && domain != "" {
		return domain
	}
	return scan.ID
}
