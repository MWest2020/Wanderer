# Tasks: hermetic Playwright fixtures

> Mark approved on 2026-05-11. Open questions resolved in
> the proposal's status block — implementation steps below.

## 1. Resolved design decisions (recorded)

- [x] 1.1 Seeder lives in `internal/fixtures/`, invoked from a
  Makefile target. No `cmd/wanderer-fixture` binary —
  auditable + nothing extra to ship.
- [x] 1.2 One DB per scenario, three Playwright projects.
- [x] 1.3 Migration sync rides on `store.Open`'s runner.
- [x] 1.4 `tests/playwright/fixtures/agent-host.yaml` stays
  side-by-side as the "smoke against my real host" tool.

## 2. Seeder package

- [x] 2.1 `internal/fixtures/seed.go` defines a `Scenario`
  enum and one exported `Build<Name>(*store.Store) error`
  function per scenario.
- [x] 2.2 `internal/fixtures/baseline.go` — two orgs
  (conduction + acme), one domain target per org, one scored
  scan each under wand + eucsf.
- [x] 2.3 `internal/fixtures/agent_host.go` — adds `alma`
  host target under conduction with a synthetic agent scan:
  a curated handful of `inventory.packages.rpm` +
  `inventory.systemd.service` findings including one
  `datadog-agent` hit so the host rule deep-dive shows
  `afhankelijk`.
- [x] 2.4 `internal/fixtures/empty_org.go` — `acme-empty` org
  with zero targets so empty-state paths render.
- [x] 2.5 `internal/fixtures/main.go` is the `go run` entry
  point: `--scenario <name>` + `--out <path>`.

## 3. Makefile + Playwright config

- [x] 3.1 `make playwright-fixture` target emits three DBs
  under `tests/playwright/fixtures/{baseline,agent-host,empty-org}.db`.
- [x] 3.2 `playwright-fixture` is a prerequisite of the
  `playwright` target so a fresh checkout builds the DBs
  before running the suite.
- [x] 3.3 `playwright.config.ts` gains three projects, each
  with its own webServer block + port.
- [x] 3.4 `.gitignore` excludes
  `tests/playwright/fixtures/*.db`.

## 4. Spec lifts

- [x] 4.1 `dar.spec.ts` switches its scope-persistence checks
  to the deterministic `baseline` DB (drop reliance on
  whatever Mark scanned last week).
- [x] 4.2 `reporting-catalogue.spec.ts` runs against
  `baseline` for the "every registered rule" assertion and
  against `empty-org` for the previously-skipped "rules
  without rationale" placeholder check.
- [x] 4.3 `host-side-scoring.spec.ts` runs against
  `agent-host`; the previously-gated host-scan smoke flips
  from `test.skip` to mandatory.

## 5. Verification

- [x] 5.1 `go test ./internal/fixtures/...` covers the
  seeder per scenario.
- [x] 5.2 `go test ./...` clean across the repo.
- [x] 5.3 `make playwright-fixture && make playwright` runs
  end-to-end on a fresh checkout (no `/tmp/wanderer-demo.db`
  needed).
- [x] 5.4 `openspec validate add-playwright-fixtures --strict`
  passes.

## 6. Wrap-up

- [x] 6.1 Update `docs/operator.md` with the seeded-DB
  workflow + how to add a scenario.
- [ ] 6.2 Commit + push (push needs explicit user OK).
- [x] 6.3 Archive under
  `openspec/changes/archive/2026-05-11-add-playwright-fixtures/`.
