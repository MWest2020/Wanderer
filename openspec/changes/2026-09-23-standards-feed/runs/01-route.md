# Habitat run 01 — `POST /imports/internetnl` (tasks 1.1–1.4)

Contract: `openspec/changes/2026-09-23-standards-feed/` (proposal.md,
specs/scanner/spec.md). Read the proposal first: the reason this is an HTTP
route at all is the line in the homelab Deployment,
`# RWO-volume + SQLite: nooit twee schrijvers`.

## Decisions already made — do not reopen them

1. **One implementation.** The import logic lives in
   `cmd/wanderer/import.go` as `importNetnlDomains(...)`, in package `main`,
   which `internal/api` cannot import. Move it into a shared package
   (`internal/scanner` next to `ParseNetnlFindings`, or a small new
   `internal/netnlimport`) and make BOTH the CLI and the route call it. The
   CLI's behaviour must not change; its existing tests must stay green
   untouched.
2. **Idempotency across both doors.** The CLI hashes the file bytes. The
   route SHALL hash the request body bytes the same way, so the same file is
   "already imported" whether it arrived by CLI or HTTP. Write a test that
   imports by CLI path and then posts the same bytes: the second must report
   already imported.
3. **Fail closed.** Look at how the agent intake does it
   (`internal/api/findings.go`, `FindingsIngestHandler`, and
   `agent.ingest.inactive` in `cmd/wanderer/serve.go`): with no secrets the
   route stays registered and refuses everything. Do the same with
   `WANDERER_IMPORT_TOKEN`: unset → every request 401, and one startup log
   line `import.internetnl.inactive`. Compare with
   `crypto/subtle.ConstantTimeCompare`.
4. **Body limit.** Cap the body (a few MiB is plenty for a fleet; a v1 file
   for ~40 domains is well under 1 MiB). An unbounded body on an
   unauthenticated-reachable port is a memory lever.

## Scope — ONLY these tasks
- [ ] 1.1 The route, calling the shared import function; response JSON with
      `imported`, `skipped_unknown`, `skipped_already`.
- [ ] 1.2 The token, as above.
- [ ] 1.3 Tests for every scenario in `specs/scanner/spec.md` of this change,
      plus the cross-door idempotency test from decision 2. Check each new
      test once with its fix removed, and say in your report that you did.
- [ ] 1.4 `docs/how-to/internetnl-ci.md`: the HTTP route next to the CLI, with
      the curl line and the token. CHANGELOG under [Unreleased].

## Out of scope
The CronJob, the netnl tenant, the secrets (homelab — done outside this run).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green, and
`openspec validate 2026-09-23-standards-feed --strict` green. golangci-lint
you cannot run in the cage; say so, it will be run afterwards. Budget $7.
