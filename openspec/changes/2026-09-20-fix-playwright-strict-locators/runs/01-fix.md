# Habitat run 01 — strict-mode locators (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-20-fix-playwright-strict-locators/`.

Gemeten op 2026-09-20 buiten de kooi (Chromium 147): 31 geslaagd, 6
gefaald; identiek op `682807f`, dus dit staat los van de
accountability-change.

## Scope — ONLY these tasks
- [ ] 1.1 Scope de regel-ID-locators binnen de bedoelde tabel
  (`table.rule-catalogue`) in: `reporting-catalogue.spec.ts`,
  `container-image-sovereignty.spec.ts`, `eu-package-origin.spec.ts`,
  `host-side-scoring.spec.ts`, `nextcloud-as-target.spec.ts`.
  `/ui/trends` toont hetzelfde regel-ID vier keer (twee tabellen), dus
  een kale `text=`-locator kán niet uniek zijn.
- [ ] 1.2 `dar.spec.ts`: `getByRole('heading', { name: /Verdict/i })`
  raakt twee koppen op `/ui/`. Maak de match exact op de bedoelde kop.
- [ ] 1.3 Raak de UI-templates en -code NIET aan. De pagina klopt; de
  test doet een verouderde aanname.

Je kunt de suite in de kooi niet draaien: `npx playwright install`
haalt ~112MB van de Playwright-CDN en die egress is dicht. Vink 1.1–1.3
af als de aanpassing klopt, laat 2.1 open, en zeg in je run-rapport dat
je de suite niet hebt gedraaid. Mark's sessie meet het na.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-fix-playwright-strict-locators
--strict` groen. Budget is $3.
