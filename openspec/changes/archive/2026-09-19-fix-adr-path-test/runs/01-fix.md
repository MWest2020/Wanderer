# Habitat run 01 — fix-adr-path-test (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-19-fix-adr-path-test/`.

## Scope — ONLY these tasks
- [ ] 1.1 `internal/ui/playwright_coverage_test.go`: read
  `docs/explanation/adr/` instead of `docs/decisions/`; messages and
  comments in that file name the new path. No other logic changes.
- [ ] 1.2 `go build ./...`, `go vet ./...`, `go test ./...` fully green
  (offline from vendor/, see openspec/config.yaml).
- [ ] 1.3 `openspec validate 2026-09-19-fix-adr-path-test --strict` green.

Done = these three checkboxes checked, and 1.1–1.3 ticked in tasks.md.
Task 2.1 (apply deltas + archive) is NOT part of this run.

## Out of scope
Any other file under `internal/`, `vendor/`, or `docs/`. If anything else
fails, note it in the run report and stop.
