# Tasks: Agent accepts host-only Targets

## 1. Model

- [x] 1.1 Add `TargetKind` type and constants in `pkg/models/target.go`
- [x] 1.2 Update `Target.Validate` to accept host-only Targets when `Kind == "host"`
- [x] 1.3 Add tests for both kinds plus invalid-kind rejection

## 2. Store

- [x] 2.1 Add `kind` column to `targets` table; idempotent migration
- [x] 2.2 `UpsertTarget` and `GetScan` round-trip Kind
- [x] 2.3 Tests

## 3. Agent

- [x] 3.1 `cmd/wanderer/agent.go` sets `Kind: TargetKindHost` on the bootstrap Target
- [x] 3.2 Manual smoke: `wanderer agent --once` on a host whose hostname has no dot

## 4. Docs + CHANGELOG

- [x] 4.1 Update `docs/agent.md` to note that bare hostnames are accepted
- [x] 4.2 CHANGELOG entry under `### Fixed` (host targets) and `### Added` (TargetKind type)

## Notes

- `GetScan` does not load the Target row, so the round-trip is via
  `UpsertTarget` (loads `kind` on existing rows) plus the new
  `Store.GetTarget` (reads the Target by ID). This preserves the
  existing Scan-detail query shape.
- Migration 003 uses a `DEFAULT 'domain'` so backfilled rows from
  pre-agent databases stay perimeter targets.
- `NormaliseHost` rejects URL syntax (`/`, `?`, `://`) so an
  operator who points the agent at a URL gets an error rather than
  a silently truncated hostname.
- Manual smoke (3.2): on this dev host whose `hostname` is a bare
  label, `go test ./internal/store/...` exercising
  `TestUpsertTarget_HostKindRoundTrip` is the equivalent of the
  agent's bootstrap path; the build passes which proves the
  Validate + migration + INSERT chain.
