# Habitat run 02 — import scans are not "the latest scan" (tasks 4.1–4.2)

Contract: `openspec/changes/2026-09-23-standards-feed/`, specifically
`specs/web-ui/spec.md` and `specs/scheduling/spec.md`. Read tasks.md section 4
for how this was found: on prod, after the first import, four domains showed
0/0 on the fleet screen.

## What is wrong

An import (`scanner.ImportNetnlDomains`, via CLI or `POST /imports/internetnl`)
creates a scan of its own holding only `source_modus = 'import'` Findings and
no assessment. Several places take "the newest scan of a target" by
`started_at` without looking at that:

- `internal/ui/ui.go` — around lines 410, 550, 726 (fleet), 1015
- `internal/ui/aggregate.go` — around lines 41 and 356
- `internal/ui/demo.go` — around line 141
- `internal/store/drift.go` — `PreviousScanForTarget`, used by the fleet delta
  (`ui.go` ~752) and by the drift engine (`internal/drift/diff.go`
  `Compute`, called from `internal/scheduler/scheduler.go`)

Line numbers are from commit 48e8d56; verify each site yourself and look for
any this list missed (`grep -rn "StartedAt.After\|started_at DESC"`).

## Decisions already made — do not reopen them

1. **One definition, in the store.** "Import scan" is decided in one place in
   `internal/store` and every site above uses it. Do not re-derive it in each
   handler. An import scan never contains a non-import Finding, and a
   perimeter scan never contains an import Finding — so "has an import
   Finding" is a sufficient test and needs no migration. If you find that
   invariant broken anywhere in the code, stop and say so in your report
   instead of adding a column.
2. **Skip, don't hide.** Import scans stay in the store and stay reachable at
   `/ui/scans/{id}`; they are only excluded from "latest"/"previous"
   selection. Do not delete or rewrite anything.
3. **`LatestScanByModus` / `LatestImportFindings` /
   `FindingsForAssessment` stay as they are** — that is how imported evidence
   reaches the perimeter scan's assessment, and it works (verified on prod
   2026-09-23).

## Scope — ONLY these tasks
- [ ] 4.1 The store-level definition and every selection site using it.
- [ ] 4.2 Tests for every scenario in `specs/web-ui/spec.md` and
      `specs/scheduling/spec.md` of this change, at the level where the bug
      lives (store for `PreviousScanForTarget`, the fleet handler for the
      fleet row). Check each new test once with its fix removed, and say in
      your report that you did. CHANGELOG under [Unreleased], `### Fixed`.

## Out of scope
Release, homelab, unsuspending the CronJob (task 4.3 — done afterwards).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green, and
`openspec validate 2026-09-23-standards-feed --strict` green. golangci-lint
you cannot run in the cage; say so, it will be run afterwards. Budget $7.
