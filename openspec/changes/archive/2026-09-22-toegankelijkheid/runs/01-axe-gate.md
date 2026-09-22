# Habitat run 01 — axe als gate (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-22-toegankelijkheid/`
(specs/project-hygiene/spec.md, requirement "De suite toetst
toegankelijkheid en faalt erop").

## Scope — ONLY these tasks
- [ ] 1.1 axe-core in `tests/playwright`: voeg `@axe-core/playwright`
  toe aan `package.json` + lockfile (`npm install --ignore-scripts`,
  zoals `playwright-install` in de Makefile doet), met een helper die
  een pagina toetst en faalt op bevindingen van niveau `serious` of
  `critical`. De foutmelding noemt het scherm, de regel-id van axe, het
  element en de gemeten waarde — anders kan niemand hem repareren.
- [ ] 1.2 Pas die helper toe op de schermen die de suite al bezoekt:
  vloot (`/ui/`), antwoord, onderbouwing, trends, regelpagina, demo.
- [ ] 1.3 De bestaande fouten worden daarmee zichtbaar. **Repareer ze
  NIET in deze run.** Laat de suite falen, en schrijf in je run-rapport
  welk scherm waarop faalt. De reparatie is run 02, zodat de gate en de
  reparatie los van elkaar te lezen zijn.

## Wat je gaat vinden (gemeten 2026-09-22, buiten de kooi)
`/ui/` 3× contrast, antwoord 6×, trends 49× + een lege tabelkop,
onderbouwing 61×. Voorbeeld: `.answer-badge-nee` haalt 3,4 waar 4,5
nodig is (#d9534f op #faeaea, 13,6px). Kom je op andere aantallen,
zeg dat dan — dat is nuttige informatie, geen fout.

## Je kunt de suite hier niet draaien
De kooi heeft geen egress naar de Playwright-CDN, dus `npx playwright
install` faalt. Schrijf de helper en de aanroepen, en zeg in je
run-rapport dat je ze niet hebt kunnen draaien. Ik meet het na.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (die raken niet
aan deze wijziging) en `openspec validate 2026-09-22-toegankelijkheid
--strict` groen. Budget is $4.
