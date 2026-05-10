# Tasks: restructure DAR layers

## 1. Routes + handlers

- [x] 1.1 New `/ui/analysis` route + `analysisHandler` rendering
  the rule × score-counts matrix (current
  `reportingIndexHandler` content)
- [x] 1.2 Replace `reportingIndexHandler` with a rule-catalogue
  view: list every rule from wand + eucsf with description,
  dimension, rationale, link to detail
- [x] 1.3 `reportingRuleHandler` keeps its content but its
  back-link points at `/ui/reporting` (the catalogue)
- [x] 1.4 `dashboardHandler` (and `dashboardOrgHandler` via
  `renderDashboard`) trims posture blocks, Top concerns,
  Recent activity; renders a per-framework verdict pill
  instead

## 2. Aggregator helper

- [x] 2.1 New `ListAllRules()` that returns the union of
  `wand.DefaultRules() + eucsf.DefaultRules()`, sorted by
  framework + dimension + ID
- [x] 2.2 `WorstByFramework(snaps)` computing per-framework
  the worst score reached across all targets, plus the count
  of targets at that score

## 3. Templates

- [x] 3.1 New `templates/analysis.tmpl` (the matrix view —
  basically the old reporting.tmpl content)
- [x] 3.2 Rewrite `templates/reporting.tmpl` to render the rule
  catalogue: framework / dimension / ID / description /
  rationale rows
- [x] 3.3 `templates/reporting_rule.tmpl` back-link → `/ui/reporting`
- [x] 3.4 Rewrite `templates/dashboard.tmpl` per-framework
  verdict pills + organisations list; drop posture blocks,
  Top concerns, Recent activity sections
- [x] 3.5 Nav `Active` "analysis" maps to `/ui/analysis` for
  the link

## 4. Tests

- [x] 4.1 `/ui/analysis` returns 200 and contains rule × score
  matrix (regression: same content the old `/ui/reporting`
  rendered)
- [x] 4.2 `/ui/reporting` returns 200 and contains rule
  descriptions for at least one wand and one eucsf rule
- [x] 4.3 `/ui/` dashboard contains the per-framework verdict
  pill and does NOT contain "External posture" / "Top concerns"
  / "Recent activity" headings
- [x] 4.4 Nav-bar tests still pass (Analysis link points at
  `/ui/analysis`, Reporting at `/ui/reporting`, scope
  threading honoured)

## 5. Docs

- [x] 5.1 `docs/architecture.md` "Read-only operator UI" — the
  three layer roles match the new shape
- [x] 5.2 `CHANGELOG.md` entry under `### Changed`
