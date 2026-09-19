# Habitat run 08c — unieke ankers per framework (task 8.6)

Gevonden door de spec uit run 08b buiten de kooi te draaien (Chromium):

```
strict mode violation: locator('#accountability') resolved to 2 elements:
  1) <article id="accountability" …> (eucsf-kaart, onbekend/incomplete)
  2) <article id="accountability" …> (wand-kaart, afhankelijk/partial)
```

## Scope — ONLY this task
- [ ] 8.6 De dimensiekaarten op `/ui/scans/{id}/assessment` krijgen een
  anker dat uniek is op de pagina: neem het framework op in de id
  (bijv. `wand-accountability`, `eucsf-accountability`). De
  accountability-pil op de Overview linkt naar het wand-anker. Werk
  `tests/playwright/specs/accountability-answer-sheet.spec.ts` bij zodat
  hij dat anker gebruikt — beide tests in die spec moeten slagen.
  Raak alleen de ankers/links aan, niet de inhoud van de kaarten.

## Bekend en NIET van jou
Zes andere specs falen al sinds vóór deze change (ook op 682807f) op
hetzelfde soort strict-mode-conflict in de rule-catalogue, waar een
regel-ID twee keer op /ui/trends staat. Niet repareren in deze run.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` groen. Budget is $3. Kun je de Playwright-spec niet draaien
(geen browser-egress in de kooi), zeg dat dan in je run-rapport en vink
8.6 toch af als de aanpassing klopt — ik draai hem hier na.
