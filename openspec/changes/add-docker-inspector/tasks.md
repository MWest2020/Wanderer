# Tasks: Docker inspector

## 1. Client

- [ ] 1.1 `internal/probe/inventory/docker/client.go` — http.Client over a unix socket dialer
- [ ] 1.2 Test: connect / GET / decode JSON against an httptest server bound to a tmp socket

## 2. Inspector

- [ ] 2.1 Replace the placeholder `docker.go` Available()/Inspect() with real socket calls
- [ ] 2.2 Map the API rows to `inventory.docker.container` and `inventory.docker.image` Findings
- [ ] 2.3 Surface non-2xx responses as `inventory.docker.error`
- [ ] 2.4 Permission-denied / socket-missing branches keep emitting `inventory.docker.unavailable`

## 3. Tests

- [ ] 3.1 Fixture-driven Parse tests for the JSON shapes
- [ ] 3.2 End-to-end test against a tmp-socket httptest server
- [ ] 3.3 Failure-mode tests: 404, 500, bad JSON, permission denied

## 4. Docs + CHANGELOG

- [ ] 4.1 Update `docs/agent.md` to remove the "placeholder" caveat for Docker
- [ ] 4.2 Update `docs/findings.md` with the populated Docker ProbeIDs
- [ ] 4.3 CHANGELOG entry under `### Added`
