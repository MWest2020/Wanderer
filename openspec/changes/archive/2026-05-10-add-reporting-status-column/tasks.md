# Tasks: compact status column on the Reporting catalogue

## 1. Aggregator

- [x] 1.1 `WorstScoreFromCounts(counts) (score, atWorst, total)`
  helper that turns a RuleSummary counts-map into the triage
  triple

## 2. Handler

- [x] 2.1 `reportingCatalogueHandler` resolves `?org=<slug>`,
  builds snapshots, runs `RuleSummary`, joins counts back into
  the catalogue rows from `ListAllRules`
- [x] 2.2 Rules with no recorded Rationale leave `Status` empty
  so the template renders the "no rationale yet" placeholder

## 3. Template

- [x] 3.1 New "Current state" column in `reporting.tmpl`
- [x] 3.2 Scope-pill shown when the catalogue is filtered by
  `?org=<slug>`, mirroring the existing Analysis / Targets
  pattern

## 4. Tests

- [x] 4.1 `/ui/reporting?org=conduction` body contains
  `score-` badges and target counts on rules that have fired
- [x] 4.2 Rules that have not fired render with "no rationale
  yet"

## 5. Docs

- [x] 5.1 `CHANGELOG.md` entry under `### Changed` noting the
  catalogue gained a triage column
