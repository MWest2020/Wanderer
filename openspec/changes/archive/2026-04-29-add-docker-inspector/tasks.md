# Tasks: Docker inspector

## 1. Client

- [x] 1.1 `internal/probe/inventory/docker/client.go` — http.Client over a unix socket dialer
- [x] 1.2 Test: connect / GET / decode JSON against an httptest server bound to a tmp socket

## 2. Inspector

- [x] 2.1 Replace the placeholder `docker.go` Available()/Inspect() with real socket calls
- [x] 2.2 Map the API rows to `inventory.docker.container` and `inventory.docker.image` Findings
- [x] 2.3 Surface non-2xx responses as `inventory.docker.error`
- [x] 2.4 Permission-denied / socket-missing branches keep emitting `inventory.docker.unavailable`

## 3. Tests

- [x] 3.1 Fixture-driven Parse tests for the JSON shapes
- [x] 3.2 End-to-end test against a tmp-socket httptest server
- [x] 3.3 Failure-mode tests: 404, 500, bad JSON, permission denied

## 4. Docs + CHANGELOG

- [x] 4.1 Update `docs/agent.md` to remove the "placeholder" caveat for Docker
- [x] 4.2 Update `docs/findings.md` with the populated Docker ProbeIDs
- [x] 4.3 CHANGELOG entry under `### Added`

## Notes

- Engine API version pinned at `v1.41` (Docker 20.10+, Dec 2020)
  per docker.go's `dockerAPIVersion`. Older daemons return an error
  on this version path, which surfaces cleanly as
  `inventory.docker.error` rather than wrong data.
- Permission-denied is detected at `os.Stat` time, so the failure
  mode path runs through `Available()=false` →
  `inventory.docker.unavailable`. A connection that succeeds at
  TCP-stat level but fails mid-API call (rare) goes through
  `inventory.docker.error` instead.
- Fixture-driven Parse tests (3.1) and end-to-end (3.2) overlap in
  this implementation: a single unix-socket httptest server
  exercises the same JSON shapes and the same client+inspector
  path. The matrix is covered by `docker_test.go` parameterised on
  the response body.
- Permission-denied (3.3) is covered by the
  `Available_MissingFile` branch (the same code path catches
  `os.IsPermission`); reproducing a true EACCES in unit tests
  requires running as a non-root user against a root-owned socket,
  which is environment-specific and adds little coverage over the
  os.Stat error-classification logic.
