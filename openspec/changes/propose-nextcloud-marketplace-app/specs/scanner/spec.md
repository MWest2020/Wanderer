# Delta for scanner

> Paper-only. This proposal documents the decision space for
> marketplace distribution; no requirement lands until Mark
> picks Q1.

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
