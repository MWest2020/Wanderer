# Tasks: Assessor

## 1. Data types

- [ ] 1.1 `pkg/models/assessment.go` — `Assessment`, `DimensionScore`, `Rationale`, `Score`, `Completeness` types + validation
- [ ] 1.2 Round-trip JSON tests for `Assessment`

## 2. Rule engine

- [ ] 2.1 `internal/assessor/rule.go` — `Rule`, `RuleResult`, engine entry point
- [ ] 2.2 `internal/assessor/engine.go` — aggregate per dimension, compute completeness, build rationale list
- [ ] 2.3 `internal/assessor/engine_test.go` — table-driven tests covering perimeter-only, partial, full completeness

## 3. DICTU rule set

- [ ] 3.1 `internal/assessor/dictu/rules.go` — seed rule set (≥10 rules spanning all 4 currently-reachable dimensions)
- [ ] 3.2 `internal/assessor/dictu/rules_test.go` — one test per rule with fabricated Findings

## 4. Report rendering

- [ ] 4.1 `internal/assessor/report.go` — markdown renderer
- [ ] 4.2 Golden-file test covering one complete + one partial dimension
- [ ] 4.3 JSON and plain-text renderers

## 5. Store integration

- [ ] 5.1 Extend `internal/store` schema with `assessments` table (migration)
- [ ] 5.2 `store.CreateAssessment`, `store.GetAssessment`, `store.ListAssessmentsForScan`
- [ ] 5.3 Store tests covering persistence + retrieval

## 6. CLI

- [ ] 6.1 `cmd/wanderer/assess.go` — `wanderer assess <scan-id> [--format text|markdown|json] [--db ...]`
- [ ] 6.2 Register `assess` subcommand in `cmd/wanderer/main.go`
- [ ] 6.3 Smoke test: assess an existing scan, verify exit codes for missing scan

## 7. HTTP API

- [ ] 7.1 `POST /scans/{id}/assessments` — runs assessor, persists, returns Assessment
- [ ] 7.2 `GET /assessments/{id}` — returns persisted Assessment
- [ ] 7.3 API test covering success + not-found + bad-scan-id

## 8. Docs + CHANGELOG

- [ ] 8.1 `docs/assessor.md` — how to interpret an Assessment, what the scores mean, how the DICTU rule set maps
- [ ] 8.2 Update `docs/README.md` index
- [ ] 8.3 Add CHANGELOG entry under `Added`
- [ ] 8.4 ADR `0004-assessor-rule-engine.md` documenting "rules are Go functions, not a DSL"
