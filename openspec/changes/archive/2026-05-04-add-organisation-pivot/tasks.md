# Tasks: Organisation pivot — implementation

## 1. Model + migration

- [x] 1.1 `pkg/models/organisation.go` — `Organisation` struct
  with `ID`, `Slug`, `Name`, `Description`, `CreatedAt`. Slug
  validation (2–40 lowercase alphanumeric + hyphens, no
  leading/trailing hyphen, unique)
- [x] 1.2 `pkg/models/organisation_test.go` — slug rules pinned
- [x] 1.3 Migration 005 in `internal/store/migrations.go`:
  create `organisations` table, seed `default` row, add
  `targets.organisation_id` column with FK to organisations,
  backfill every existing row to `o_default`, recreate the
  table to enforce NOT NULL on the column
- [x] 1.4 Migration test: seed pre-migration data, run, assert
  every Target has `organisation_id = 'o_default'`, the unique
  slug constraint holds

## 2. Store helpers

- [x] 2.1 `UpsertOrganisation(ctx, *models.Organisation) error`
- [x] 2.2 `GetOrganisationBySlug(ctx, slug) (*models.Organisation, error)`
- [x] 2.3 `ListOrganisations(ctx) ([]models.Organisation, error)`
- [x] 2.4 `RenameOrganisation(ctx, oldSlug, newSlug, newName) error`
  (covers Q1's escape hatch)
- [x] 2.5 `Selectors.OrganisationID` — extend the existing
  selector type so `ListScans` filters by org
- [x] 2.6 `UpsertTarget` accepts `OrganisationID`; empty falls
  back to `o_default`
- [x] 2.7 Unit tests for each helper

## 3. CLI: `wanderer org` subcommand

- [x] 3.1 `cmd/wanderer/org.go` — dispatch for
  `add` / `list` / `show` / `rename` / `add --from-yaml`
- [x] 3.2 `wanderer org add --slug <s> --name <n> [--description <d>]`
- [x] 3.3 `wanderer org list` — table output (slug, name, target
  count)
- [x] 3.4 `wanderer org show <slug>` — shows org + lists targets
- [x] 3.5 `wanderer org rename --slug <old> --new-slug <new>
  [--name <n>]`
- [x] 3.6 `wanderer org add --from-yaml <path>` — bulk seed,
  idempotent
- [x] 3.7 Wire into `cmd/wanderer/main.go` dispatcher
- [x] 3.8 Subcommand tests

## 4. `wanderer scan` flag wiring

- [x] 4.1 `--organisation <slug>` flag on scan; `WANDERER_ORGANISATION`
  env var; precedence flag > env > serve-yaml-fallback >
  `default`
- [x] 4.2 Resolve slug → organisation ID at scan start; unknown
  slug fails fast
- [x] 4.3 Pass `OrganisationID` through to `UpsertTarget`
- [x] 4.4 Tests covering the precedence ladder

## 5. `wanderer agent` config wiring

- [x] 5.1 `core.organisation` field in `internal/agent/config.go`
- [x] 5.2 Agent passes the slug through to the API on Target
  upsert; control plane validates the slug exists and rejects
  unknown ones with a 4xx
- [x] 5.3 Tests

## 6. `serve.yaml` fallback + schedules

- [x] 6.1 `scan.organisation` field in `internal/serveconfig/`
- [x] 6.2 Resolved through the existing flag/env/yaml/default
  helper into the scan invocation
- [x] 6.3 `Schedule.Organisation` field in
  `internal/scheduler/config.go` — optional, falls back to
  serve config; both unset → startup error naming the schedule
- [x] 6.4 Tests

## 7. UI

- [x] 7.1 New route `/ui/orgs/{slug}` rendering the same
  Dashboard template scoped to that organisation
- [x] 7.2 Instance-wide `/ui/` headline mentions "all
  organisations" and lists registered orgs as a sub-section,
  each linking to `/ui/orgs/{slug}`
- [x] 7.3 Reporting page `/ui/reporting?org=<slug>` filters
  rules to that organisation's targets
- [x] 7.4 Render tests

## 8. MCP

- [x] 8.1 `org.list` method
- [x] 8.2 `org.show` method
- [x] 8.3 `org.targets` method (lists targets for an org)
- [x] 8.4 Method tests

## 9. Docs + CHANGELOG

- [x] 9.1 `docs/operator.md` — Organisation section: how to
  create / list / show / rename, the `default` migration story,
  the YAML seed
- [x] 9.2 `docs/architecture.md` — Organisation as the top-of-
  hierarchy entity; UI / CLI / MCP cross-reference
- [x] 9.3 `CHANGELOG.md` entry under `### Added` for the pivot
  + under `### Changed` noting that existing data attaches to
  `default`
- [x] 9.4 First-run nudge stderr line implemented in
  `wanderer serve` and `wanderer scan` startup
