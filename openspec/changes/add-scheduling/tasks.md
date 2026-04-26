# Tasks: Scheduling + Drift

## 1. Schedules config

- [x] 1.1 `internal/scheduler/config.go` — YAML parsing + cron validation (via robfig/cron/v3 Parser)
- [x] 1.2 Unit tests covering invalid cron, missing target, duplicate schedule names

## 2. Scheduler runtime

- [x] 2.1 `internal/scheduler/scheduler.go` — register jobs with robfig/cron, start/stop lifecycle
- [x] 2.2 Hook into `cmd/wanderer/serve.go` lifecycle (start after server, stop before server)
- [x] 2.3 SIGHUP handler to re-read schedules file
- [x] 2.4 Panic recovery per job, emit `scheduler.panic` Finding

## 3. Drift engine

- [x] 3.1 `internal/drift/diff.go` — entry point: `Compute(ctx, store, currentScan) ([]Finding, error)` — finds previous scan, runs rules
- [x] 3.2 `internal/drift/rules.go` — seed rule set (issuer, mx_set, ns_set, country, third_party, security_headers, days_left)
- [x] 3.3 Per-rule tests with fabricated Scan pairs
- [x] 3.4 `drift.baseline_established` and `drift.no_changes` emission

## 4. Store integration

- [x] 4.1 `store.PreviousScanForTarget(ctx, targetID, before time.Time)` returning the most recent scan before `before`, if any
- [x] 4.2 `store.ListDriftForTarget(ctx, targetID, since time.Time)` for API queries
- [x] 4.3 Tests

## 5. HTTP API

- [x] 5.1 `GET /targets/{id}/drift?since=<RFC3339>` returning drift Findings
- [x] 5.2 API tests

## 6. CLI

- [x] 6.1 `cmd/wanderer/diff.go` — `wanderer diff <scan-a> <scan-b>` (reads both scans, runs diff, prints markdown)
- [x] 6.2 Register `diff` in `cmd/wanderer/main.go`
- [x] 6.3 Smoke test against two stored scans

## 7. Docs + CHANGELOG + ADR

- [x] 7.1 `docs/scheduling.md` — example schedules file, SIGHUP semantics, systemd unit example
- [x] 7.2 `docs/drift.md` — what counts as drift, which rules fire, how the assessor consumes drift
- [x] 7.3 Update `docs/findings.md` with drift ProbeIDs
- [x] 7.4 `docs/decisions/0006-in-process-scheduler.md` — why robfig/cron in-process, not k8s CronJob
- [x] 7.5 CHANGELOG entry under `Added`
