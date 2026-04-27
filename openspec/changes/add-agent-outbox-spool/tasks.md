# Tasks: Agent outbox spool

## 1. Outbox

- [ ] 1.1 `internal/agent/outbox.go` — Spool / Drain / Prune
- [ ] 1.2 Tests: round-trip, drain-with-failure, prune-by-size, missing-dir

## 2. Remote integration

- [ ] 2.1 `agent.Remote.Send` retries 3× with backoff, then spools
- [ ] 2.2 Tests with httptest server that fails N times then succeeds

## 3. Agent loop

- [ ] 3.1 `cmd/wanderer/agent.go` drains outbox before each tick
- [ ] 3.2 Config: optional `core.outbox_dir` (default
  `/var/lib/wanderer/agent/outbox`) and `core.outbox_max_bytes`
  (default 100 MiB)

## 4. Docs + CHANGELOG

- [ ] 4.1 Update `docs/agent.md` outbox section
- [ ] 4.2 CHANGELOG entry under `### Added`
