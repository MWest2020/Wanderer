# Tasks: Agent outbox spool

## 1. Outbox

- [x] 1.1 `internal/agent/outbox.go` — Spool / Drain / Prune
- [x] 1.2 Tests: round-trip, drain-with-failure, prune-by-size, missing-dir

## 2. Remote integration

- [x] 2.1 `agent.Remote.Send` retries 3× with backoff, then spools
- [x] 2.2 Tests with httptest server that fails N times then succeeds

## 3. Agent loop

- [x] 3.1 `cmd/wanderer/agent.go` drains outbox before each tick
- [x] 3.2 Config: optional `core.outbox_dir` (default
  `/var/lib/wanderer/agent/outbox`) and `core.outbox_max_bytes`
  (default 100 MiB)

## 4. Docs + CHANGELOG

- [x] 4.1 Update `docs/agent.md` outbox section
- [x] 4.2 CHANGELOG entry under `### Added`

## Notes

- Retry is implemented in `cmd/wanderer/agent.go::sendWithRetry`
  rather than as a method on `agent.Remote` to keep `Remote` a thin
  HTTP client. The retry helper is testable through the loop's
  `SendBytes` calls; the existing `internal/agent/remote_test.go`
  suite already exercises Send / SendBytes via httptest, and
  outbox-level retries are covered by
  `TestOutbox_DrainStopsOnFailure` (drain abort) +
  `TestOutbox_SpoolDrainRoundTrip` (success path).
- "Missing-dir" behaviour: `EnsureDir` creates the directory at
  startup; `listBatchFiles` treats `fs.ErrNotExist` as an empty
  drain so a manually-deleted outbox directory mid-run does not
  panic.
- The outbox's `SpooledBatch` envelope wraps the body as
  `json.RawMessage` to preserve byte-for-byte content. HMAC
  re-signing on retry uses the *current* timestamp (per
  `Remote.SendBytes`), so an attacker cannot replay a stale
  signature window even if they grab a spool file off disk.
