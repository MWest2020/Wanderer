// Package fixtures composes deterministic SQLite databases for the
// Playwright suite. Each exported Build* function seeds one
// scenario via the public store API — no new storage logic, just
// curated inputs. The DBs are throwaway: Playwright builds them on
// every `make playwright` invocation and the .gitignore excludes
// them.
//
// Adding a scenario:
//
//  1. Add a Build<Name> function in its own file.
//  2. Register it in Scenarios.
//  3. Add a Playwright project in tests/playwright/playwright.config.ts
//     pointing at the new DB path.
//
// Not for production use. `cmd/wanderer` deliberately does not
// import this package; the seeder runs only via `go run
// ./internal/fixtures`.
package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/eucsf"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Scenarios is the registry of built-in scenarios. The CLI
// (`internal/fixtures` main) iterates this map to honour
// `--scenario <name>`.
var Scenarios = map[string]func(context.Context, *store.Store) error{
	"baseline":   BuildBaseline,
	"agent-host": BuildAgentHost,
	"empty-org":  BuildEmptyOrg,
}

// baseTime anchors every fixture's timestamps so two runs of the
// same scenario produce byte-identical DBs (modulo SQLite header
// metadata). Using a fixed point in early 2026 keeps the demo
// dates plausible without coupling to "now".
var baseTime = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

// upsertOrg is a small helper that swallows the "already exists"
// case the seeder hits when re-running into a non-fresh DB.
func upsertOrg(ctx context.Context, st *store.Store, slug, name string) (*models.Organisation, error) {
	o := &models.Organisation{Slug: slug, Name: name}
	if err := st.UpsertOrganisation(ctx, o); err != nil {
		return nil, fmt.Errorf("fixtures: upsert org %q: %w", slug, err)
	}
	return o, nil
}

// upsertTarget creates the target if absent, leaves the row alone
// otherwise. Returns the persisted row so callers can chain a
// scan against its ID.
func upsertTarget(ctx context.Context, st *store.Store, domain string, kind models.TargetKind, orgID string) (*models.Target, error) {
	t := &models.Target{
		Domain:         domain,
		Kind:           kind,
		OrganisationID: orgID,
	}
	if err := st.UpsertTarget(ctx, t); err != nil {
		return nil, fmt.Errorf("fixtures: upsert target %q: %w", domain, err)
	}
	return t, nil
}

// addCompletedScan creates a scan, appends findings, marks it
// success, then scores it under both frameworks. The persisted
// scan + assessments mirror what `wanderer scan` + `wanderer
// assess` produce in production.
func addCompletedScan(ctx context.Context, st *store.Store, target *models.Target, when time.Time, findings []models.Finding) (*models.Scan, error) {
	sc, err := st.CreateScan(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("fixtures: create scan: %w", err)
	}
	// Backdate StartedAt so timestamps are deterministic.
	if _, err := st.DB().ExecContext(ctx,
		`UPDATE scans SET started_at = ? WHERE id = ?`, when, sc.ID); err != nil {
		return nil, fmt.Errorf("fixtures: backdate scan: %w", err)
	}
	sc.StartedAt = when

	if err := st.AppendFindings(ctx, sc.ID, findings); err != nil {
		return nil, fmt.Errorf("fixtures: append findings: %w", err)
	}
	if err := st.FinishScan(ctx, sc.ID, models.ScanStatusComplete, ""); err != nil {
		return nil, fmt.Errorf("fixtures: finish scan: %w", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`UPDATE scans SET ended_at = ? WHERE id = ?`, when.Add(2*time.Minute), sc.ID); err != nil {
		return nil, fmt.Errorf("fixtures: backdate scan end: %w", err)
	}

	// Score under wand + eucsf so the UI's assessment pill renders.
	if err := scoreScan(ctx, st, sc, "wand", wand.DefaultRules(), findings, when); err != nil {
		return nil, err
	}
	if err := scoreScan(ctx, st, sc, "eucsf", eucsf.DefaultRules(), findings, when); err != nil {
		return nil, err
	}
	return sc, nil
}

func scoreScan(ctx context.Context, st *store.Store, sc *models.Scan, framework string, rules []assessor.Rule, findings []models.Finding, when time.Time) error {
	// AppendFindings assigns IDs; re-read the scan's findings so the
	// assessor sees the persisted IDs (Evidence citations need them).
	persisted, err := readFindings(ctx, st, sc.ID)
	if err != nil {
		return err
	}
	dims := assessor.Assess(persisted, rules)
	a := &models.Assessment{
		ScanID:     sc.ID,
		Framework:  framework,
		Dimensions: dims,
		CreatedAt:  when.Add(3 * time.Minute),
	}
	if err := st.CreateAssessment(ctx, a); err != nil {
		return fmt.Errorf("fixtures: persist assessment (%s): %w", framework, err)
	}
	return nil
}

// readFindings returns the persisted Findings for a scan with
// their IDs populated. The store's GetScan returns the scan
// envelope; we need the Finding rows specifically.
func readFindings(ctx context.Context, st *store.Store, scanID string) ([]models.Finding, error) {
	sc, err := st.GetScan(ctx, scanID)
	if err != nil {
		return nil, fmt.Errorf("fixtures: re-read scan: %w", err)
	}
	return sc.Findings, nil
}

// mkFinding produces a SourceModusPerimeter Finding with sane
// defaults. Scenario code only sets the bits it cares about.
func mkFinding(probeID, subject string, dim models.DimensionHint, attrs map[string]any) models.Finding {
	return models.Finding{
		ProbeID:       probeID,
		Subject:       subject,
		Severity:      models.SeverityFinding,
		SourceModus:   models.SourceModusPerimeter,
		DimensionHint: dim,
		Attributes:    attrs,
	}
}

// mkInventoryFinding mirrors mkFinding but tags the SourceModus
// as inventory so the assessor's completeness calculation routes
// the Finding correctly.
func mkInventoryFinding(probeID, subject string, attrs map[string]any) models.Finding {
	return models.Finding{
		ProbeID:       probeID,
		Subject:       subject,
		Severity:      models.SeverityInfo,
		SourceModus:   models.SourceModusInventory,
		DimensionHint: models.DimensionTechnologie,
		Attributes:    attrs,
	}
}
