# Delta for scanner

> Decided 2026-06-15 — architecture D (AppAPI ExApp), shipped as the
> separate downstream repo `MWest2020/wanderer-exapp`. The
> separately-tracked-product-surface requirement below now lands into
> the canonical scanner spec; it is satisfied by the separate repo.

## ADDED Requirements

### Requirement: Wanderer marketplace app is a separately-tracked product surface

A Wanderer Nextcloud marketplace app, if pursued, SHALL ship
as a separate top-level surface — its own directory, its own
release cadence, its own quality bar — and SHALL NOT introduce
PHP / Composer / Nextcloud-app dependencies into the core Go
codebase. The picked architecture (A: PHP shim + Go sidecar,
B: PHP reimplementation, C: WebAssembly) MUST be recorded in
`add-nextcloud-marketplace-app`'s status block before any
code lands.

#### Scenario: Marketplace surface does not pollute the core Go module

- **GIVEN** an active marketplace app
- **WHEN** a contributor runs `go test ./...` from the repo
  root
- **THEN** no PHP / Composer / Nextcloud-app dependency is
  required to pass
