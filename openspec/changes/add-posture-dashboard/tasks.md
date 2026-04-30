## 1. Aggregation helpers

- [ ] 1.1 Add `internal/ui/aggregate.go` with pure functions: `WorstScore(dim []models.AssessmentDimension) models.Score` (ignores onbekend, all-onbekend → onbekend), `PostureCounts(scans []models.Scan, assess [][]models.Assessment) PostureSummary`, `TopConcerns(assessments [][]models.Assessment, ruleLookup func(framework, criteriumID string) (assessor.Rule, bool)) []ConcernRow`
- [ ] 1.2 Unit tests in `internal/ui/aggregate_test.go` covering: worst-score with mixed dimensions, all-onbekend short-circuit, top-concerns target-counting (one rule firing 50× on one target counts once; same rule on 5 targets counts five), empty inputs

## 2. Dashboard handler + template

- [ ] 2.1 Rename `indexHandler` → `targetsHandler`; mount at `r.Get("/targets", targetsHandler(st, tmpl))`; the existing `index.tmpl` is reused as the targets template
- [ ] 2.2 Add `dashboardHandler(st, tmpl)`, mounted at `r.Get("/", dashboardHandler(st, tmpl))`. The handler reads `ListScans` + `ListAssessmentsForScan` (one call per most-recent scan), feeds the aggregation helpers, builds a `dashboardView` struct
- [ ] 2.3 Add `dashboard.tmpl` rendering the three sections; reuse the existing `score-<level>` CSS classes for badges
- [ ] 2.4 The dashboard's "All targets" link points at `/ui/targets`; the recent-activity rows link to `/ui/scans/{id}/assessment` when `HasAssessment`, else `/ui/scans/{id}`

## 3. Tests

- [ ] 3.1 `TestDashboard_RendersPostureSummary` in `internal/ui/ui_test.go`: seed three scans with mixed worst-scores, assert the response body contains `1 soeverein`, `1 afhankelijk`, `1 onbekend` in the DICTU block
- [ ] 3.2 `TestDashboard_TopConcernsCountsByTarget`: seed one rule firing on 5 distinct targets, assert the rule's CriteriumID appears in the top-concerns block with a target count of 5
- [ ] 3.3 `TestDashboard_RecentActivity`: seed 7 scans, assert exactly 5 rows appear, ordered newest-first
- [ ] 3.4 `TestDashboard_EmptyStore`: empty store returns 200 and contains the empty-state copy ("run `wanderer scan`")
- [ ] 3.5 `TestTargetsRouteServesPreviousIndex`: the URL the previous `/ui/` index served at is now reachable at `/ui/targets`, same content
- [ ] 3.6 Static-analysis test in `internal/ui/ui_test.go` continues to pin the read-only contract (no new mutation handlers introduced)

## 4. Docs + CHANGELOG

- [ ] 4.1 Update `docs/architecture.md` "Read-only operator UI" section to describe the dashboard at `/ui/` and the auditor table at `/ui/targets`
- [ ] 4.2 Update / create `docs/operator.md` walkthrough — dashboard first, then drilldowns
- [ ] 4.3 CHANGELOG entry under `### Added` (dashboard) and `### Changed` (the previous `/ui/` index URL is now `/ui/targets`)
