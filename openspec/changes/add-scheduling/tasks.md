# Tasks: Scheduling + Drift

## 1. Schedules config

- [ ] 1.1 `internal/scheduler/config.go` — YAML parsing + cron validation (via robfig/cron/v3 Parser)
- [ ] 1.2 Unit tests covering invalid cron, missing target, duplicate schedule names

## 2. Scheduler runtime

- [ ] 2.1 `internal/scheduler/scheduler.go` — register jobs with robfig/cron, start/stop lifecycle
- [ ] 2.2 Hook into `cmd/wanderer/serve.go` lifecycle (start after server, stop before server)
- [ ] 2.3 SIGHUP handler to re-read schedules file
- [ ] 2.4 Panic recovery per job, emit `scheduler.panic` Finding

## 3. Drift engine

- [ ] 3.1 `internal/drift/diff.go` — entry point: `Compute(ctx, store, currentScan) ([]Finding, error)` — finds previous scan, runs rules
- [ ] 3.2 `internal/drift/rules.go` — seed rule set (issuer, mx_set, ns_set, country, third_party, security_headers, days_left)
- [ ] 3.3 Per-rule tests with fabricated Scan pairs
- [ ] 3.4 `drift.baseline_established` and `drift.no_changes` emission

## 4. Store integration

- [ ] 4.1 `store.PreviousScanForTarget(ctx, targetID, before time.Time)` returning the most recent scan before `before`, if any
- [ ] 4.2 `store.ListDriftForTarget(ctx, targetID, since time.Time)` for API queries
- [ ] 4.3 Tests

## 5. HTTP API

- [ ] 5.1 `GET /targets/{id}/drift?since=<RFC3339>` returning drift Findings
- [ ] 5.2 API tests

## 6. CLI

- [ ] 6.1 `cmd/wanderer/diff.go` — `wanderer diff <scan-a> <scan-b>` (reads both scans, runs diff, prints markdown)
- [ ] 6.2 Register `diff` in `cmd/wanderer/main.go`
- [ ] 6.3 Smoke test against two stored scans

## 7. Docs + CHANGELOG + ADR

- [ ] 7.1 `docs/scheduling.md` — example schedules file, SIGHUP semantics, systemd unit example
- [ ] 7.2 `docs/drift.md` — what counts as drift, which rules fire, how the assessor consumes drift
- [ ] 7.3 Update `docs/findings.md` with drift ProbeIDs
- [ ] 7.4 `docs/decisions/0008-in-process-scheduler.md` — why robfig/cron in-process, not k8s CronJob
- [ ] 7.5 CHANGELOG entry under `Added`
