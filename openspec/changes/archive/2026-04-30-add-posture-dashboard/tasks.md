## 1. Aggregation helpers

- [x] 1.1 Add `internal/ui/aggregate.go` with pure functions: `WorstScore(dim []models.AssessmentDimension) models.Score` (ignores onbekend, all-onbekend → onbekend), `PostureCounts(snaps []TargetSnapshot) PostureSummary`, `TopConcerns(snaps []TargetSnapshot, ruleLookup, maxRows) []ConcernRow`, plus `RecentActivity(scans []store.ScanRow, hasAssessment, maxRows) []ActivityRow`
- [x] 1.2 Unit tests in `internal/ui/aggregate_test.go` covering: worst-score with mixed dimensions, all-onbekend short-circuit, top-concerns target-counting (one rule firing 50× on one target counts once; same rule on 5 targets counts five), empty inputs, recent-activity ordering and cap, has-assessment flag

## 2. Dashboard handler + template

- [x] 2.1 Rename `indexHandler` → `targetsHandler`; mount at `r.Get("/targets", targetsHandler(st, tmpl))`; the existing `index.tmpl` is reused as the targets template
- [x] 2.2 Add `dashboardHandler(st, tmpl)`, mounted at `r.Get("/", dashboardHandler(st, tmpl))`. The handler reads `ListScans` + `ListAssessmentsForScan`, builds `[]TargetSnapshot`, feeds the aggregation helpers, and assembles a `dashboardView`
- [x] 2.3 Add `dashboard.tmpl` rendering the three sections; reuses the existing `score-<level>` CSS classes for badges
- [x] 2.4 The dashboard's "All targets" link points at `/ui/targets`; recent-activity rows link to `/ui/scans/{id}/assessment` when the scan has an Assessment, else `/ui/scans/{id}`

## 3. Tests

- [x] 3.1 `TestDashboard_PostureSummary`: seed three scans with mixed worst-scores, assert the response body contains `1 soeverein`, `1 afhankelijk`, `1 onbekend` in the DICTU block
- [x] 3.2 `TestDashboard_TopConcerns_TargetCounted`: seed one rule firing on 5 distinct targets, assert the rule's CriteriumID appears in the top-concerns block with a target count of 5
- [x] 3.3 `TestDashboard_RecentActivity`: seed 7 scans, assert exactly 5 rows appear, ordered newest-first
- [x] 3.4 `TestDashboard_EmptyStoreRendersEmptyHint`: empty store returns 200 and contains the empty-state copy ("run `wanderer scan`")
- [x] 3.5 `TestTargetsRoute_RendersTargetRow`: the URL the previous `/ui/` index served at is now reachable at `/ui/targets`, same content
- [x] 3.6 Static-analysis test in `internal/ui/ui_test.go` continues to pin the read-only contract (no new mutation handlers introduced)

## 4. Docs + CHANGELOG

- [ ] 4.1 Update `docs/architecture.md` "Read-only operator UI" section to describe the dashboard at `/ui/` and the auditor table at `/ui/targets`
- [ ] 4.2 Update / create `docs/operator.md` walkthrough — dashboard first, then drilldowns
- [x] 4.3 CHANGELOG entry under `### Added` (dashboard) and `### Changed` (the previous `/ui/` index URL is now `/ui/targets`)

## Notes

- 4.1 + 4.2 (docs walkthrough) are deliberately deferred: the
  dashboard's screenshots and the operator-doc walkthrough want
  a fresh demo run on a host with GeoLite2 wired so the
  technologie dimension is actually populated. The
  document-geoip-setup change made the wiring frictionless;
  the next operator-driven demo session is the right time to
  capture the screenshots and write the walkthrough. CHANGELOG
  carries the URL move so the change is discoverable.
- `WorstScore` ignores onbekend dimensions deliberately — a
  dimension we cannot evaluate is not "worst", it is unknown.
  All-onbekend → onbekend (treated as the unknown bucket on the
  posture summary).
- `TopConcerns` counts distinct targets, not rationales, so a
  noisy probe firing the same rule fifty times on one host
  doesn't dominate the dashboard. Tied target counts are sorted
  by CriteriumID for stable rendering.
- Posture summary iterates `AllScores` (best→worst→onbekend) and
  skips zero counts so a dashboard for an estate that scored
  zero `voldoende` doesn't render an empty `0 voldoende` row.
- `internal/ui/ui.go::dashboardHandler` consults
  `ListAssessmentsForScan` once per most-recent-per-target scan
  (small N) and lazily once per recent-activity scan that wasn't
  already covered. SQL-level aggregation can replace this
  in-Go path later without changing the handler signature, per
  Decision 5 in the design.
