# Habitat run 03 — het vlootscherm (tasks 3.1–3.3)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/`
(specs/web-ui/spec.md, requirement "Het vlootscherm scoort x van n,
niet ja of nee").

## Scope — ONLY these tasks
- [x] 3.1 De vlootpagina (`/ui/orgs/{slug}/fleet`, run 02) toont per
  domein `x/n` uit `BuildFleetScore` (run 01, internal/ui/fleet_score.go),
  met het aantal onbeantwoorde vragen ernaast ("5/7 · 2 onbekend") en
  de zwaarste openstaande bevinding uit dezelfde functie. Bouw geen
  tweede telling.
- [x] 3.2 Het verschil sinds de vorige scan per domein: hoeveel vragen
  erbij of eraf, en welke stroom omsloeg. `internal/drift` doet al iets
  vergelijkbaars per target — kijk daar eerst of je het kunt hergebruiken
  in plaats van naast te zetten; zeg in je run-rapport wat je koos.
- [x] 3.3 Sorteren op score, op verandering en op laatste scan, via
  links met een query-parameter (geen JavaScript nodig — deze UI werkt
  zonder). De gekozen sortering blijft zichtbaar.

Een domein zonder scan toont "nog niet gescand" en verdwijnt niet naar
onderen bij sorteren op score; zet die apart onderaan.

## Out of scope
De regelpagina en de handelingen (run 04 en 05).

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen `go test ./internal/ui/...`; bewaar één
volledige build/vet/test voor het eind. De eerste volledige build duurt
minuten (gevendorde SQLite) — normaal.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-20-vloot-en-regels --strict` groen.
Budget is $5.
