# Habitat run 08b — de Playwright-spec echt laten draaien (tasks 8.4–8.5)

Run 08 schreef `tests/playwright/specs/accountability-answer-sheet.spec.ts`
maar kon hem niet draaien (geen browser in de kooi) — eerlijk vermeld in
het bestand zelf. Buiten de kooi gedraaid met Chromium: de spec draait
nog steeds niet, want hij staat in geen enkel `testMatch` van
`tests/playwright/playwright.config.ts`. Een spec die nergens in staat,
bewijst niets.

## Scope — ONLY these tasks
- [ ] 8.5 Zet `accountability-answer-sheet.spec.ts` in het
  `baseline`-project in `playwright.config.ts`, en vul de
  `baseline`-fixture (`internal/fixtures`) aan met een scan die alle vier
  de antwoordtoestanden laat zien: ja (bijv. securitytxt), nee
  (afhankelijk), onbekend met reden `probe_unavailable`, en n.v.t. met
  reden `registry_redacted`. De andere scenario's blijven ongemoeid, en
  bestaande specs mogen niet van de nieuwe fixture-rijen omvallen.
- [ ] 8.4 Werk de spec bij zodat hij tegen die fixture slaagt en
  daadwerkelijk toetst: bewijs uitklappen (`<details>`), en onbekend
  versus n.v.t. als zichtbaar verschillende toestanden. Haal de NOTE
  bovenin weg die zegt dat hij niet gedraaid is — maar alleen als je hem
  in deze run écht hebt gedraaid. Kun je hem niet draaien, laat de NOTE
  staan en zeg dat in je run-rapport.

Let op: `npx playwright install chromium` haalt ~112MB van de
Playwright-CDN. Heeft de kooi die egress niet, dan is 8.4 niet af te
ronden in de kooi — vink hem dan NIET af en meld het.

## Bekend en NIET van jou
Zes specs falen al sinds vóór deze change (ook op commit 682807f), op
strict-mode locator-conflicten in de rule-catalogue ("resolved to 2
elements"). Niet repareren in deze run; alleen niet erger maken.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` groen. Budget is $4.
