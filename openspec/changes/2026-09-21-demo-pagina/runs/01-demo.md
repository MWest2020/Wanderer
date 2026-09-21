# Habitat run 01 — de demoroute (tasks 1.1–1.5, 3.1)

Contract: `openspec/changes/2026-09-21-demo-pagina/`
(specs/web-ui/spec.md).

## Scope — ONLY these tasks
- [x] 1.1 `demo.target` in `internal/serveconfig`; leeg betekent geen
  demo, en dat staat één keer in het opstartlog (zoals
  `agent.ingest.inactive` dat doet).
- [x] 1.2 `/demo` (op de root-router, NIET onder `/ui`, want `/ui` zit
  achter de login) toont de laatste VOLTOOIDE scan van dat domein: de
  antwoordzin uit `BuildAnswerVerdict` en de onderbouwing zoals de
  assessment-pagina die rendert, plus de datum van de scan. Hergebruik
  die weergave; bouw geen tweede.
- [x] 1.3 Op die pagina komt geen ander domein voor — ook niet in een
  link, een regeloverzicht of een navigatiebalk. De gewone nav (Overview
  / Vloot / Trends) hoort hier niet; zet er een eigen, minimale kop
  boven die zegt dat het een voorbeeld is en waar je het zelf draait.
- [x] 1.4 Vanaf de demo is geen scan te starten, ook niet met een
  handmatig POST-verzoek naar de scanroute: die blijft achter de login.
- [x] 1.5 Tests: demo uit → 404; demo aan zonder voltooide scan → "nog
  geen meting"; demo aan met scan → het oordeel en de datum; en een test
  die de pagina afzoekt op de naam van een ánder target in de
  testdatabase en faalt als die erin staat.
- [x] 3.1 docs + CHANGELOG: wat de demo is, en waarom er geen scanknop
  op staat (een open scanknop maakt de instantie een scanner-voor-derden).

## Let op
De demo is een smal, apart oppervlak — geen uitzondering op de
inlogregel. Raak de bestaande poort voor `/ui` niet aan. De statische
controle die nieuwe muterende routes weigert blijft gelden; de demo is
alleen-lezen.

## Over bouwen en testen in de kooi
Draai tijdens het werk `go test ./internal/ui/... ./internal/serveconfig/...`;
bewaar één volledige build/vet/test voor het eind (de eerste duurt
minuten).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-demo-pagina --strict` groen. Budget is $5.
