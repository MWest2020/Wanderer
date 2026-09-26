# Habitat run 01 — verwijderen waar je kijkt (tasks 1.1–1.4)

Contract: `openspec/changes/2026-09-26-verwijderen-waar-je-kijkt/`
(proposal, specs/web-ui/spec.md).

## Where things are
- `internal/ui/templates/fleet.tmpl`: the remove form is the last `<td>`
  (`{{if $.AllowEdit}}`), the add form at the top.
- `internal/ui/templates/dashboard.tmpl`: the domain grid (`.table-scroll`
  already wraps it); whether the viewer may edit is the same condition as
  `allowFleetEdit` in `internal/ui/ui.go` (gate != nil) — thread it into the
  dashboard view the same way the fleet page gets `AllowEdit`.
- `internal/ui/ui.go`: `fleetRemoveHandler` (redirects to the fleet page
  today), `store.RemoveFleetDomain` (soft: stamps removed_at).

## Decisions already made
1. **Next to the domain name**, not a last column: same markup in both
   places (a small shared template block).
2. **No JavaScript.** `<details class="remove"><summary>verwijderen</summary>`
   holding the POST form with a button "Ja, haal {domein} uit de vloot" and a
   muted line "de scans blijven bewaard".
3. **Return target** is a hidden form field. Accept exactly: `/ui/`,
   `/ui/orgs/{slug}`, `/ui/orgs/{slug}/fleet` for the org in the URL. Anything
   else — absolute URLs, `//host`, other orgs — falls back to the fleet page.
   Parse it, don't prefix-match strings.
4. Fleet page table goes inside `.table-scroll` (existing CSS class).
5. Nothing else changes: not the remove semantics, not the store.

## Tests
- Go: the return-target whitelist (each accepted form, and at least
  `https://elders.example/`, `//elders.example/`, another org's fleet page).
- Playwright: from `/ui/` remove a domain and land on `/ui/` without it; on
  390×844 every remove summary on the fleet page is inside the viewport and
  `scrollWidth <= innerWidth`.
- Each new test once red with its fix removed; say so. CHANGELOG under
  [Unreleased], `### Fixed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-26-verwijderen-waar-je-kijkt --strict` green.
Playwright and golangci-lint run afterwards. Budget $6.
