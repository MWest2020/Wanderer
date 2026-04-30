## Why

The current `/ui/` is a flat targets table — one row per target,
last-scan status, and a per-framework score badge. That is the
**Reporting** layer in DAR terms (a list of records); it is not
a **Dashboard** (a posture summary you can read at a glance).
An operator opening the UI today cannot answer the questions a
manager asks first: "How many targets are sovereign right now?
What changed this week? Where are the worst concerns?"

The data already exists (`store.ListScans`,
`store.ListAssessmentsForScan`, the persisted Rationales per
Dimension); only the aggregation and presentation are missing.
With `add-assessment-ui` filling in the **Analysis** middle layer,
this proposal closes the DAR triad by replacing the current `/ui/`
with a real **Dashboard** and demoting the targets table to a
secondary `/ui/targets` route for the auditor view.

## What Changes

- Replace `/ui/` with a dashboard view that renders, for the most
  recent Assessment per (target, framework):
  - **Posture summary** — counts of targets per worst-dimension
    score (soeverein / gedeeld / afhankelijk / onbekend), per
    framework. SEAL totals shown alongside DICTU when both are
    populated.
  - **Top concerns** — the rule rationales most commonly scored
    `afhankelijk` across targets, with a count of how many
    targets each fired against. Limited to ~5 rows.
  - **Recent activity** — the five most recent scans with a
    timestamp, target, status, and worst-dimension score badge,
    each linking to the per-scan Analysis page from
    `add-assessment-ui`.
- Add `/ui/targets` route serving the existing flat table the way
  `/ui/` rendered before this change. The dashboard links to it
  ("All targets →"). Keeps the auditor surface intact.
- The existing `/ui/scans/{id}` and `/ui/targets/{id}/drift` and
  the new `/ui/scans/{id}/assessment` routes are unchanged.
- Read-only contract holds: every new handler is GET-only; the
  static-analysis test continues to pin it.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `web-ui`: replaces the flat `/ui/` index with an aggregate
  posture dashboard; preserves the previous flat-table view at
  `/ui/targets` so the auditor surface is not lost.

## Impact

**Code**:
- `internal/ui/`: one new handler (`dashboardHandler`), one
  rename of the current handler from `indexHandler` to
  `targetsHandler` and a new route `/targets`. New
  `dashboard.tmpl` template; the existing `index.tmpl` is reused
  under the new route. CSS gets a small set of card styles
  matching the existing visual language.
- New helpers in `internal/ui/aggregate.go` (or similar) that
  compute the posture summary and top-concerns counts from the
  scan + assessment store calls. These are pure functions of
  `[]models.Scan` + `[][]models.Assessment` so they unit-test
  cleanly.

**APIs**: none. The HTTP API stays unchanged; the dashboard reads
the same store paths the targets index already used.

**Dependencies**: none. Stdlib + `html/template`.

**Read-only contract**: preserved. All new handlers GET-only.

**DICTU dimensions informed**: the dashboard surfaces aggregate
posture across all five dimensions; it does not introduce new
rules.

**Passive/active boundary**: N/A — UI rendering only.

**Compatibility note**: an operator with bookmarked links to
`/ui/` lands on the dashboard instead of the targets list.
`/ui/targets` is the new home of the flat table; the change is
called out in CHANGELOG under `### Changed`.
