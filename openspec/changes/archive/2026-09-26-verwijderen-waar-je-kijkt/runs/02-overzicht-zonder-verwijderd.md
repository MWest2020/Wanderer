# Habitat run 02 — het overzicht laat verwijderde domeinen weg (tasks 1b.1–1b.2)

Contract: this change (proposal, specs/web-ui/spec.md) and the existing
requirement "Domeinen zijn bij te houden als vloot" in
`openspec/specs/web-ui/spec.md`, scenario "Verwijderen laat de geschiedenis
staan": the domain "verdwijnt uit het overzicht en blijven de scans
raadpleegbaar". Run 01 is merged.

## 1b.1 — decisions already made
- `/ui/` and `/ui/orgs/{slug}` build from `buildSnapshots` (internal/ui/ui.go)
  → `st.ListScans(ctx, sel)`. Add a selector field to `store.Selectors`
  (e.g. `ActiveTargetsOnly bool`) that adds `targets.removed_at IS NULL` to
  ListScans' query, and set it in `buildSnapshots` only. Everything that
  derives from those snapshots (ring, bars, top 3, grid, headline counts)
  then leaves removed domains out — that is intended: a removed domain no
  longer counts toward the fleet.
- Do NOT set it for exports, Trends, drift, the scan page or MCP: history
  stays reachable (the same scenario says so).
- Store test: a removed target's scans are left out with the selector and
  kept without it.

## 1b.2
- `tests/playwright/specs/vloot-en-regels.spec.ts` lines ~55 and ~67 compare
  the whole first `td` with the domain name. Compare the domain name
  element instead (the link or the element that holds the name), not the
  cell.

## Tests
The failing Playwright test "verwijderen op /ui/ landt weer op /ui/ zonder
dat domein" must pass because of 1b.1 — do not change its assertion. Each
new test once red with its fix removed; say so. CHANGELOG `### Fixed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-26-verwijderen-waar-je-kijkt --strict` green.
Playwright runs afterwards. Budget $5.
