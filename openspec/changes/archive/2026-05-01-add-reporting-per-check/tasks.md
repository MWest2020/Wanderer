# Tasks: per-check Reporting page

## 1. Aggregator

- [x] 1.1 `RuleSummaryRow` + `RuleSummary` in
  `internal/ui/aggregate.go` — distinct-target counts per
  (framework, ruleID, score)
- [x] 1.2 `RuleTargetRow` + `RuleTargetRows` — per-target rows
  for one (framework, ruleID), newest-Assessment-wins
- [x] 1.3 Stable order helpers — wand > eucsf > alphabetical for
  framework, then ruleID alphabetical within
- [x] 1.4 Tests for both helpers (distinct counts, stable order,
  filter correctness, newest-Assessment-wins)

## 2. Handlers + routes

- [x] 2.1 `reportingIndexHandler` builds snapshots, calls
  `RuleSummary`, renders `reporting.tmpl`
- [x] 2.2 `reportingRuleHandler` parses `{framework}/{ruleID}`,
  resolves `lookupRule`, returns 404 on unknown rule, otherwise
  calls `RuleTargetRows` and renders `reporting_rule.tmpl`
- [x] 2.3 `Handler` registers the two routes
- [x] 2.4 Set `HasReporting=true` on the dashboard view so the
  nav tab renders

## 3. Templates

- [x] 3.1 `templates/reporting.tmpl` — index table (Framework,
  Rule, sov / vol / afh / onb counts)
- [x] 3.2 `templates/reporting_rule.tmpl` — rule detail header +
  per-target row table

## 4. Tests

- [x] 4.1 Aggregator unit tests (4.1.1 distinct counts,
  4.1.2 stable order, 4.1.3 newest-Assessment-wins,
  4.1.4 filter by rule)
- [x] 4.2 Render test for `/ui/reporting` — table contains rule
  IDs + per-score counts
- [x] 4.3 Render test for `/ui/reporting/{fw}/{ruleID}` — 200
  with rule heading + target rows; unknown rule returns 404

## 5. Docs + changelog

- [x] 5.1 `docs/architecture.md` "Read-only operator UI" — note
  that Reporting is now wired
- [x] 5.2 `CHANGELOG.md` entry under `### Added`
