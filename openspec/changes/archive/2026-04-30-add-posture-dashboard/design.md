## Context

`internal/ui/ui.go` today builds the targets-list view by calling
`store.ListScans(ctx, store.Selectors{})`, grouping by `TargetID`,
picking the most recent scan per target, and looking up the latest
Assessment per framework via `store.ListAssessmentsForScan`. That
flow already produces the data the dashboard needs — it just
discards the aggregations after rendering one row per target.

`add-assessment-ui` (companion proposal) lands the per-scan
Analysis page and gives every Rule a `Rationale` paragraph. The
dashboard's "top concerns" rows render the same Rationale strings;
when both proposals are landed an operator drills from a top-line
concern → the Analysis page → the underlying Findings.

The persisted `models.Assessment` carries a `Dimensions[]` slice
with per-dimension Score + Completeness + Rationales. A target's
"worst-dimension score" is the lowest Score across populated
dimensions for that target's most recent Assessment per framework.
("Lowest" is per the score-rank ordering already present in
`models.Score.Rank()`.)

## Goals / Non-Goals

**Goals:**

- An operator opening `/ui/` reads the org's posture in one
  glance: counts per worst-dimension score, top concerns, recent
  activity.
- The dashboard renders correctly when only one framework
  (DICTU-only) is populated.
- The dashboard renders correctly with zero targets / zero
  assessments (initial-state UX).
- The auditor view (flat targets table) remains accessible at
  `/ui/targets` so an existing workflow is not broken.

**Non-Goals:**

- Time-series charts. The current store does not retain enough
  historical state to render meaningful trends (only the latest
  scan per target informs the dashboard); a sparkline / trend
  view is its own proposal.
- Custom widgets / drag-and-drop / saved views. The dashboard is
  a fixed three-section page.
- Per-user views. The UI is single-tenant; everyone sees the same
  dashboard.
- Filtering by tag / business-unit. Targets carry no
  organisational metadata yet; that is its own model change.

## Decisions

### Decision 1: Worst-dimension score is the headline number

For each (target, framework) the dashboard counts the **worst
populated dimension score** as the headline. Alternatives
considered:

1. **Best-dimension score.** Optimistic; an `afhankelijk
   juridisch + soeverein operationeel` target is still
   afhankelijk for a sovereignty audit.
2. **Average of dimensions.** Statistically attractive, but the
   DICTU score scale is ordinal not numeric — averaging
   `gedeeld` and `afhankelijk` is meaningless without a defined
   weight, and we do not want to ship a weighting opinion in the
   read-only UI.
3. **Worst-dimension (chosen).** The audit lens. A target is no
   more sovereign than its weakest dimension; that is also how
   `wanderer assess --format markdown` renders the bottom-line
   verdict today.

`onbekend` dimensions are **excluded** from the
worst-score calculation — a dimension we cannot evaluate is not
"worst", it is unknown. If every dimension is `onbekend`, the
target's worst-dimension score is `onbekend`.

### Decision 2: Top concerns counts targets, not findings

A rule that fires `afhankelijk` on 50 findings within one target
counts as **one** concern. A rule that fires `afhankelijk` on one
finding each across 10 targets counts as **ten**. The dashboard's
question is "how widespread is this concern across our estate";
counting findings rewards noisy probes.

### Decision 3: Recent activity is the last 5 scans, not 5 per target

A simple chronological ordering, capped at 5. Operators who want
"all scans for this target" go to `/ui/targets/{id}/drift` (still
present) or `/ui/scans/{id}` directly. The dashboard's job is the
30-second view.

### Decision 4: Move the flat table to `/ui/targets`, not delete

An operator accustomed to `/ui/` showing the flat table loses
that view if we just replace it. Keeping it under `/ui/targets`
preserves the auditor's mental model; the dashboard adds the
manager's view without breaking the auditor's. The CHANGELOG
entry under `### Changed` makes the move discoverable.

### Decision 5: Aggregation is in-Go, not in SQL

`internal/ui/aggregate.go` exposes pure functions over
`[]models.Scan` + `[][]models.Assessment` rather than custom SQL
joins. Reasons:

- Keeps the store layer slim — the existing
  `ListScans` + `ListAssessmentsForScan` already return what we
  need.
- Pure-Go aggregation is trivially unit-testable.
- The data volume is small (one row per target per scan; a public-
  sector estate has on the order of hundreds, not millions). SQL
  optimisation would be premature.

If/when the store grows beyond what an in-process aggregator can
hold, the aggregation can move to SQL views without changing the
handler signature — it is implementation, not contract.

## Risks / Trade-offs

[Risk] An operator's bookmark to `/ui/` now lands on a dashboard
they did not expect. → Mitigation: the change is announced in
CHANGELOG under `### Changed`; the dashboard's first link is
"All targets →" (`/ui/targets`) so the auditor flow is one click
away. The targets URL is also linked from each posture-summary
card for context.

[Risk] The aggregation produces zeros / empty cards on a
freshly-installed instance and looks broken. → Mitigation: empty-
state copy in each card explains the state and points at
`wanderer scan <domain>` / `wanderer assess <id>` so an operator
knows what to do next. Tested by an empty-store scenario in the
spec.

[Risk] "Worst-dimension score" misleads an operator who reads
`afhankelijk` as a global verdict instead of a per-dimension one.
→ Mitigation: each posture-summary card shows the dimension that
contributed the worst score next to the badge, not just a
free-floating colour. Tooltip / details-block explains the rule.

**Clever valkuil:**

1. **Gauge widgets / circular meters / heatmaps.** They look
   modern and they communicate poorly. The DICTU score scale is
   four discrete bins; a meter implies a continuum that is not
   there. Bar counts and badges match the data shape.
2. **Aggregating across DICTU and EUCSF into one big number.**
   The frameworks measure adjacent but distinct concerns; merging
   them flattens information operators rely on. Side-by-side
   blocks per framework are honest.
3. **Realtime auto-refresh / WebSocket push.** The data changes
   on a scan-tick cadence (minutes to hours). HTML refresh on
   page-reload is appropriate; pushing live updates would add a
   long-lived connection surface inconsistent with the read-only
   contract.

**External systems & failure modes:**

- `store.ListScans(ctx, Selectors{})` — same call the targets
  index uses today; failure mode unchanged (500 with the error
  via `http.Error`).
- `store.ListAssessmentsForScan(scanID)` — empty slice on no
  assessments; the dashboard handles the empty-state path.
- The browser — auto-escaping via `html/template` covers the
  user-controlled bits (target domains, rule descriptions).
