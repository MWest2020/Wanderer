# Habitat run 02 — de vloot bijhouden (tasks 2.1–2.3)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/`
(specs/web-ui/spec.md, requirement "Domeinen zijn bij te houden als
vloot").

## Scope — ONLY these tasks
- [x] 2.1 Een ingelogde gebruiker kan een domein aan de vloot van een
  organisatie toevoegen en er weer uit halen, zónder te scannen.
  Verwijderen haalt het domein uit het overzicht en laat de scans en
  oordelen staan — gooi geen geschiedenis weg.
- [x] 2.2 Per domein tonen: laatste scan (of "nog niet gescand") en
  onder welk schema het valt. Het schema komt nu uit het
  schedules-bestand (`internal/scheduler`); toon wat daar staat. Het
  schema vanuit de UI kunnen zetten is expliciet NIET deze run.
- [x] 2.3 Dezelfde poort als scannen: alleen voor ingelogde gebruikers.
  De statische controle die nieuwe muterende routes weigert, blijft —
  voeg de nieuwe routes toe aan de uitzondering zoals de scanroute dat
  is, en licht toe waarom.

## Out of scope
Het vlootscherm zelf met x/n en sortering (run 03), de regelpagina
(run 05), en het schema kunnen wijzigen.

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen de pakketten die je raakt
(`go test ./internal/ui/... ./internal/store/...`) en bewaar één
volledige build/vet/test voor het eind. De eerste volledige build duurt
minuten (gevendorde SQLite) — dat is normaal.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-20-vloot-en-regels --strict` groen.
Budget is $5.
