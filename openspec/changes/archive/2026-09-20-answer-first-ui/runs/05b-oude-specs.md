# Habitat run 05b — de oude specs beschrijven de oude deur (task 5.3)

Buiten de kooi gedraaid met Chromium: **36 geslaagd, 5 gefaald**. De
nieuwe spec (`answer-first-flow.spec.ts`) slaagt. De vijf die falen,
beschrijven allemaal de indeling die deze change juist vervangt:

```
dar.spec.ts:11                 h1 moet "all organisations" bevatten
ui-personas.spec.ts:13         section.targets-fleet moet zichtbaar zijn
ui-personas.spec.ts:29         rapport via de vlootlijst openen
sovereignty-overview.spec.ts:11  section.sovereignty-overview op de assessment-pagina
sovereignty-overview.spec.ts:40  de flow-rollup op het dashboard
```

`/ui/` is nu de deur: een invoerveld plus "Recent beantwoord". De
vlootlijst, de rollup en de matrix staan op `/ui/trends`. De specs
kijken dus op de verkeerde pagina — de UI is niet stuk.

## Scope — ONLY this task
- [ ] 5.3 Breng die vijf specs in lijn met de nieuwe indeling: wat op
  `/ui/` werd gezocht en daar niet meer hoort, zoek je voortaan op
  `/ui/trends`; wat over de deur gaat (kop, recent beantwoord) toets je
  op `/ui/`. Verander de UI NIET om een oude spec te plezieren — als
  een spec iets eist dat de change bewust heeft weggehaald, pas dan de
  spec aan en zeg in je run-rapport wat je hebt verplaatst.
- Laat `answer-first-flow.spec.ts` ongemoeid; die slaagt al.
- Raak `internal/ui` alleen aan als een spec een echt gat blootlegt;
  meld dat dan expliciet in je run-rapport.

## Je kunt de suite hier niet draaien
`npx playwright install` haalt ~112MB van de Playwright-CDN en die
egress is dicht. Werk op de spec-teksten, vink af wat je hebt gedaan,
en zeg in je run-rapport dat de suite buiten de kooi is nagemeten.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` groen.
Budget is $4.
