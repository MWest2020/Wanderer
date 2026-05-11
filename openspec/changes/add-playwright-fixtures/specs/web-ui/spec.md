# Delta for web-ui

> Held active until Mark signs off on the design.

## ADDED Requirements

### Requirement: Playwright suite runs against a deterministic seeded DB

The Playwright suite SHALL produce its source-of-truth DB from
a hermetic seeder rather than depend on an operator's
hand-rolled scan history. A developer cloning the repo SHALL
be able to run `make playwright` end-to-end without any prior
`wanderer scan` or `wanderer agent` invocations.

#### Scenario: Fresh clone runs Playwright

- **Given** a clean checkout with no `/tmp/wanderer-demo.db`
  present
- **When** the developer runs `make playwright`
- **Then** the suite builds the fixture DB(s), boots
  `wanderer serve` against them, and every non-skip spec
  passes

#### Scenario: Schema migration breaks a fixture cleanly

- **Given** a schema change that the seeder's literal SQL no
  longer satisfies
- **When** `make playwright-fixture` runs
- **Then** the build fails at fixture compile time, before
  Playwright ever opens a browser, with an error pointing at
  the diverging table

### Requirement: Three fixture scenarios cover the spec set

The fixture seeder SHALL produce three independent SQLite
files, each representing one curated scenario: `baseline`
(two orgs, perimeter scans), `agent-host` (adds a real
inventory scan with one US-telemetry hit), and `empty-org`
(an org with zero targets so empty-state paths render).

#### Scenario: Host rule deep-dive shows the synthetic agent hit

- **Given** the `agent-host` fixture loaded
- **When** an operator opens
  `/ui/reporting/wand/wand.host.no_us_telemetry_packages`
- **Then** the per-target table contains an `alma` row scored
  `afhankelijk` with `datadog-agent` named in the verdict

#### Scenario: Empty-org placeholder renders

- **Given** the `empty-org` fixture loaded
- **When** an operator opens `/ui/orgs/acme-empty/reporting`
- **Then** the rule catalogue lists every registered rule
  with a "no rationale yet" placeholder in the description
  column
