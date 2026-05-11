# Proposal: hermetic Playwright fixtures

> **Status:** Implementation. Mark approved on 2026-05-11
> ("wat het beste uiteindelijk is in maintainability en
> verkoopbaar boring & auditable product"). Picked the
> recommendations on every open question, except Q1 where
> "auditable + verkoopbaar" tips the balance from
> `cmd/wanderer-fixture` to `internal/fixtures`:
>
> 1. **Seeder location:** `internal/fixtures` invoked from a
>    Makefile target. No extra production binary surface, no
>    risk of an operator shipping or running the wrong binary,
>    a reviewer looking at `cmd/` sees only one binary —
>    auditable.
> 2. **One DB per scenario:** three independent SQLite files
>    under `tests/playwright/fixtures/`. Three Playwright
>    projects in `playwright.config.ts`, each pointing at its
>    own DB. Independent scenarios = independent assertions.
> 3. **Migration sync:** lean on `store.Open`'s migration
>    runner; no separate "fixture migration" concept. A
>    schema change that breaks a fixture's literal SQL fails
>    at `make playwright-fixture` build time.
> 4. **agent-host.yaml stays:** the existing manual seed flow
>    is a different tool (operator wants to smoke against a
>    real host). Document the split in `docs/operator.md`.

## Intent

The Playwright smoke layer currently runs against
`/tmp/wanderer-demo.db` — a DB the operator builds by hand
(`wanderer scan example.nl` a few times, sometimes
`wanderer agent --once` against the host). That works for
"does the binary render the page" but is a liability for
anything that depends on specific finding shapes:

- The "Status column shows worst score" assertion only works
  because Mark happened to run scans against
  conduction.nl + mijnoverheid.us before invoking Playwright.
- The new host-side-scoring smoke ships `gated` on a manual
  `wanderer agent --once` run with a fixture config that's
  next to the specs (`tests/playwright/fixtures/agent-host.yaml`)
  — convenient for one developer, opaque for everyone else.
- A test asserting "rules without rationale render the
  placeholder" is `test.skip`-ed because no demo data exercises
  that path.

The fix: make the demo DB reproducible. A small Go binary
seeds a SQLite with a deterministic set of Findings + Targets
+ Assessments. Playwright runs against the seeded DB; specs
become free to assert on specific scores, specific orgs, and
specific rule outcomes without an operator pre-running scans.

## Scope

**In scope:**

- An `internal/fixtures/` package + a Makefile entry point
  (`make playwright-fixture`) that runs the package via
  `go run`. Each scenario exposes a `Build<Scenario>(
  store.Store) error` function that composes the same
  `store.Insert*` calls the scanner uses today. No new
  storage logic, no new production binary, no `cmd/` surface.
- Three fixture "scenarios", each producing a self-contained
  DB:
  1. **`baseline`** — minimal happy path. Two orgs (conduction
     + acme), one domain target per org, one perimeter scan
     each, scored under wand + eucsf. The current dar.spec
     and reporting-catalogue.spec assertions hold against this
     deterministically (no more "depends on what Mark scanned
     last week").
  2. **`agent-host`** — adds the `alma` host target under
     conduction with a synthetic agent scan: a handful of
     inventory.packages.rpm + inventory.systemd.service
     findings, including one US-telemetry hit (`datadog-agent`)
     so the host rule deep-dive shows `afhankelijk`.
  3. **`empty-org`** — adds an `acme-empty` org with zero
     targets so the "no scans yet" + "rules without rationale"
     placeholder paths render and can be pinned.
- `make playwright-fixture` Makefile target that compiles the
  seeder and writes
  `tests/playwright/fixtures/playwright.db` (gitignored).
- `make playwright` invokes the fixture build automatically so
  a fresh checkout runs the specs without manual seeding.
- Update existing specs to assert on the deterministic shape
  (drop "scope persistence" assertions that depend on demo
  data; lift `test.skip` blocks where the fixture now covers
  the path).

**Out of scope:**

- Fixture coverage of egress / flow / drift Findings (those
  rules and their UI surface are not yet exercised by
  Playwright at all; they can join when needed).
- A general-purpose factory framework for tests. The
  scenarios are written as plain Go using existing store APIs
  — a third scenario can be added in a single PR.
- Hermetic Go test fixtures. Go-level tests have their own
  in-memory store helper today; this change is about
  Playwright reproducibility, not Go test reproducibility.

## Open questions

1. **Where does the seeder live — `cmd/wanderer-fixture` or a
   build-only `internal/fixtures` package invoked via
   `go run`?** Cmd is more discoverable and round-trips
   through `wanderer help`; an internal package keeps the
   binary surface clean. Recommendation: `cmd/wanderer-fixture`
   with a clear "not for production" stderr warning at start
   so the wrong-binary risk stays low.

2. **One DB per scenario, or one DB with every scenario's data
   merged?** One-per-scenario keeps assertions readable
   (`/ui/orgs/acme-empty` is empty by construction). Merged
   means Playwright can run the whole suite against one
   webServer boot but couples scenarios that should stay
   independent. Recommendation: one DB per scenario, three
   `playwright.config.ts` projects each pointing at its own
   DB. Cost is three webServer boots per run (~6s total) —
   acceptable.

3. **How do scenarios stay in sync with schema migrations?**
   The seeder calls the same `store.Open` path the binary
   uses, so migrations run on first open. A schema change
   that breaks a fixture's literal SQL would fail
   `make playwright-fixture` before Playwright even runs.
   Recommendation: lean on store.Open's migration runner; no
   separate "fixture migration" concept.

4. **What about the `alma` host target the existing
   `add-host-side-scoring` change relies on?** The current
   `tests/playwright/fixtures/agent-host.yaml` is a stop-gap
   that requires the operator to run `wanderer agent --once`
   manually. If this change lands, the gated host-rule spec
   flips from `test.skip` to mandatory — and the
   `agent-host.yaml` fixture stays in-repo as a "I want to
   smoke this against my real host" tool, not as the
   Playwright seed source. Recommendation: keep both
   side-by-side; document the split in `docs/operator.md`.

## Passive / active boundary

Active in dev/CI only. The seeder is build-time tooling and
never runs in production. Output DB is gitignored.

## Risks

- **Fixture drift.** A schema change or a rule rename can
  silently desync the seeded DB from the live system. Mitigation:
  the seeder uses the public store API + DefaultRules; both
  go through the same code paths the binary uses, so a
  desync surfaces at `make playwright-fixture` build time, not
  at spec time.
- **Coverage gap.** Specs that depend on the operator's
  hand-rolled `/tmp/wanderer-demo.db` may regress when
  switched to a deterministic fixture. Mitigation: lift each
  spec one at a time, keep the old DB path as a fallback for
  one release, then drop it.

## Parallel-safe

Touches `cmd/wanderer-fixture/` (new), `Makefile`,
`tests/playwright/playwright.config.ts`, the three existing
specs, and `.gitignore`. No production code paths.
