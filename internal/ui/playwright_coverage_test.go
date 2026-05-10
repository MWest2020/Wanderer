package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPlaywrightCoverage_ADRsWithUISurface enforces the ADR-coverage
// contract from the add-playwright-adr-smoke-tests OpenSpec change.
// Every ADR file (`docs/decisions/NNNN-*.md`) that contains a
// `## UI surface` heading SHALL have a matching Playwright spec
// at `tests/playwright/specs/<adr-slug>.spec.ts`. ADRs without UI
// claims need no spec.
//
// The check runs in the ui package because that is where the UI
// scenarios end up rendered; the Go test runner picks the file
// up on `go test ./...` so a missing spec fails CI before merge.
func TestPlaywrightCoverage_ADRsWithUISurface(t *testing.T) {
	repoRoot := findRepoRoot(t)
	adrDir := filepath.Join(repoRoot, "docs", "decisions")
	specDir := filepath.Join(repoRoot, "tests", "playwright", "specs")

	entries, err := os.ReadDir(adrDir)
	if err != nil {
		t.Fatalf("read %s: %v", adrDir, err)
	}

	// ADR filename pattern: NNNN-slug.md. The slug becomes the
	// expected spec filename minus the leading number.
	adrName := regexp.MustCompile(`^(\d{4})-([a-z0-9-]+)\.md$`)
	uiSection := regexp.MustCompile(`(?m)^## *UI surface\b`)

	var missing []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := adrName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		body, err := os.ReadFile(filepath.Join(adrDir, e.Name()))
		if err != nil {
			t.Errorf("read ADR %s: %v", e.Name(), err)
			continue
		}
		if !uiSection.Match(body) {
			continue
		}
		// ADR has a UI surface section — a matching spec file must
		// exist. Accept either `<slug>.spec.ts` or
		// `<NNNN-slug>.spec.ts` (the file naming is operator
		// preference; both round-trip to the ADR).
		expectedA := m[2] + ".spec.ts"
		expectedB := m[1] + "-" + m[2] + ".spec.ts"
		if !fileExists(filepath.Join(specDir, expectedA)) &&
			!fileExists(filepath.Join(specDir, expectedB)) {
			missing = append(missing, e.Name()+
				" → expected "+expectedA+" or "+expectedB)
		}
	}
	if len(missing) > 0 {
		t.Errorf("ADRs with ## UI surface section but no Playwright spec:\n  - %s",
			strings.Join(missing, "\n  - "))
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// findRepoRoot walks up from the test's CWD until it finds a
// `go.mod`. The test runs from `internal/ui/` by default, so two
// levels up reaches the repo root; the loop tolerates running
// from elsewhere.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root (no go.mod ancestor)")
	return ""
}
