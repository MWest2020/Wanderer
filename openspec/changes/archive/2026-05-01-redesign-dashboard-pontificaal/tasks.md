# Tasks: pontificaal dashboard headline + intern/extern split

## 1. Aggregator + data shape

- [x] 1.1 New helper `Headline(snaps []TargetSnapshot, scans []store.ScanRow)` returning the headline counts and timestamps
- [x] 1.2 Extend `PostureCounts` (or add `PostureCountsByKind`) so the dashboard can request external-only / internal-only summaries
- [x] 1.3 Unit tests for the new aggregator helpers

## 2. Handler + view shape

- [x] 2.1 `dashboardView` gains `Headline`, `ExternalPostureBlocks`, `InternalPostureBlocks` fields
- [x] 2.2 `dashboardHandler` resolves Target.Kind for each unique TargetID and feeds the per-kind snapshots into the aggregator
- [x] 2.3 Empty-state copy for "no internal coverage yet" lives in the template

## 3. Templates + CSS

- [x] 3.1 `templates/dashboard.tmpl` — headline section at top, External / Internal posture sections, existing Top concerns + Activity below
- [x] 3.2 `templates/nav.tmpl` partial — Dashboard / Analysis / Reporting navigation
- [x] 3.3 Include `nav.tmpl` from dashboard / scan / assessment / drift / index templates
- [x] 3.4 `static/main.css` — `.headline`, `.scope-label`, `.nav-bar` styles

## 4. Tests

- [x] 4.1 `TestHeadline_Coverage` — counts correct for mixed kind targets
- [x] 4.2 `TestPostureCountsByKind` — split posture maps populated correctly
- [x] 4.3 `ui_test.go` render assertions — headline copy, empty-state, nav links

## 5. Docs + changelog

- [x] 5.1 `docs/architecture.md` "Read-only operator UI" section — update layout description to mention headline + scope split
- [x] 5.2 `CHANGELOG.md` entry under `### Changed`
