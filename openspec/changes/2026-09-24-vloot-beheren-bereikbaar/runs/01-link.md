# Habitat run 01 — vlootbeheer bereikbaar vanaf `/ui/` (tasks 1.1–1.2)

Contract: `openspec/changes/2026-09-24-vloot-beheren-bereikbaar/` (proposal,
specs/web-ui/spec.md). Read the proposal: this is a regression from
`vloot-in-beeld` task 1.6 — the Organisations table was the only route from
`/ui/` to the fleet manager.

## Where things are
- `internal/ui/templates/dashboard.tmpl`: the fleet section; the existing
  `{{if .ScopedOrganisation}}…FleetManageURL…vloot beheren…{{end}}` line.
- `internal/ui/ui.go`: `renderDoor` / `dashboardHandler` /
  `dashboardOrgHandler`, which fill `FleetManageURL` and decide on the
  Organisations table (`len(orgs) > 1`).

## Decisions already made
1. One organisation → `/ui/` gets the same "vloot beheren →" link that
   `/ui/orgs/{slug}` has, to that organisation's fleet page. Reuse
   `FleetManageURL`; do not add a second way to build it.
2. More organisations → unchanged: table on `/ui/`, link per org dashboard.
3. Nothing else changes: not the dashboard visuals, not the fleet page.

## Tests
- Go: both scenarios.
- Playwright (in the existing fleet spec): from `/ui/`, click "vloot
  beheren", remove a domain, and see it gone from the fleet page.
- Each new test once red with the fix removed; say so. CHANGELOG under
  [Unreleased], `### Fixed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-24-vloot-beheren-bereikbaar --strict` green.
Playwright and golangci-lint run afterwards. Budget $4.
