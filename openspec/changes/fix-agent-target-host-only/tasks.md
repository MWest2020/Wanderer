# Tasks: Agent accepts host-only Targets

## 1. Model

- [ ] 1.1 Add `TargetKind` type and constants in `pkg/models/target.go`
- [ ] 1.2 Update `Target.Validate` to accept host-only Targets when `Kind == "host"`
- [ ] 1.3 Add tests for both kinds plus invalid-kind rejection

## 2. Store

- [ ] 2.1 Add `kind` column to `targets` table; idempotent migration
- [ ] 2.2 `UpsertTarget` and `GetScan` round-trip Kind
- [ ] 2.3 Tests

## 3. Agent

- [ ] 3.1 `cmd/wanderer/agent.go` sets `Kind: TargetKindHost` on the bootstrap Target
- [ ] 3.2 Manual smoke: `wanderer agent --once` on a host whose hostname has no dot

## 4. Docs + CHANGELOG

- [ ] 4.1 Update `docs/agent.md` to note that bare hostnames are accepted
- [ ] 4.2 CHANGELOG entry under `### Fixed` (host targets) and `### Added` (TargetKind type)
