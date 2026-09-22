# Habitat run 01 — de vlootscore (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-22-drie-lagen-ciso/`
(specs/web-ui/spec.md, requirement "De vloot is de eerste laag").

## Scope — ONLY these tasks
- [ ] 1.1 Eén functie in `internal/ui` die de domeinen van een
  organisatie omzet in een vlootscore: `som(x) / som(n)` over de
  laatste voltooide scan per domein, het aantal onbeantwoorde vragen
  apart, het aantal domeinen dat niet soeverein is, en de verdeling per
  stroom ("Mail: 3 van 5 buiten de EER"). Bouw op `BuildFleetScore`
  (internal/ui/fleet_score.go) en `classifyFlows` — geen derde telling
  naast de bestaande twee.
- [ ] 1.2 De drie regels die over de hele vloot de meeste punten
  kosten: per regel hoeveel domeinen erop niet soeverein scoren.
  `RuleSummary`/`RuleTargetRows` in `internal/ui/aggregate.go` doen al
  iets in die richting — kijk daar eerst en zeg in je run-rapport wat je
  hergebruikte of waarom niet.
- [ ] 1.3 Table-driven tests, met minstens deze gevallen:
  - negen domeinen soeverein en één die faalt → de score is hoog én het
    aantal niet-soevereine domeinen is 1 (dit is de valkuil uit de
    proposal: een goede score mag een slecht domein niet verbergen);
  - een domein zonder voltooide scan telt niet mee in x of n, en wordt
    apart geteld;
  - alles onbekend → `0/0` zonder deling door nul;
  - een lege organisatie geeft een leeg, niet een kapot, resultaat.

Geen templates en geen routes in deze run — dat is run 02.

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen `go test ./internal/ui/...`; bewaar één
volledige `go build ./...` + `go vet ./...` + `go test ./...` voor het
eind. De eerste volledige build duurt minuten (gevendorde SQLite).

## Done means
Die volledige build/vet/test groen en `openspec validate
2026-09-22-drie-lagen-ciso --strict` groen. Budget is $4.
