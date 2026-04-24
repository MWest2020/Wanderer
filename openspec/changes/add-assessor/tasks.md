# Tasks: Assessor

## 1. Data types

- [x] 1.1 `pkg/models/assessment.go` — `Assessment`, `DimensionScore`, `Rationale`, `Score`, `Completeness` types + validation
- [x] 1.2 Round-trip JSON tests for `Assessment`

## 2. Rule engine

- [x] 2.1 `internal/assessor/rule.go` — `Rule`, `RuleResult`, engine entry point
- [x] 2.2 `internal/assessor/engine.go` — aggregate per dimension, compute completeness, build rationale list
- [x] 2.3 `internal/assessor/engine_test.go` — table-driven tests covering perimeter-only, partial, full completeness

## 3. DICTU rule set

- [x] 3.1 `internal/assessor/dictu/rules.go` — seed rule set (≥10 rules spanning all 4 currently-reachable dimensions)
- [x] 3.2 `internal/assessor/dictu/rules_test.go` — one test per rule with fabricated Findings

## 4. Report rendering

- [x] 4.1 `internal/assessor/report.go` — markdown renderer
- [x] 4.2 Golden-file test covering one complete + one partial dimension
- [x] 4.3 JSON and plain-text renderers

## 5. Store integration

- [x] 5.1 Extend `internal/store` schema with `assessments` table (migration)
- [x] 5.2 `store.CreateAssessment`, `store.GetAssessment`, `store.ListAssessmentsForScan`
- [x] 5.3 Store tests covering persistence + retrieval

## 6. CLI

- [x] 6.1 `cmd/wanderer/assess.go` — `wanderer assess <scan-id> [--format text|markdown|json] [--db ...]`
- [x] 6.2 Register `assess` subcommand in `cmd/wanderer/main.go`
- [x] 6.3 Smoke test: assess an existing scan, verify exit codes for missing scan

## 7. HTTP API

- [x] 7.1 `POST /scans/{id}/assessments` — runs assessor, persists, returns Assessment
- [x] 7.2 `GET /assessments/{id}` — returns persisted Assessment
- [x] 7.3 API test covering success + not-found + bad-scan-id

## 8. Docs + CHANGELOG

- [x] 8.1 `docs/assessor.md` — how to interpret an Assessment, what the scores mean, how the DICTU rule set maps
- [x] 8.2 Update `docs/README.md` index
- [x] 8.3 Add CHANGELOG entry under `Added`
- [x] 8.4 ADR `0004-assessor-rule-engine.md` documenting "rules are Go functions, not a DSL"
