# Habitat run 06 — bewijs en documentatie (tasks 6.1–6.2)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/`.

## Scope — ONLY these tasks
- [ ] 6.1 Playwright-spec: een domein toevoegen aan de vloot, het
  vlootscherm sorteren (score en laatste scan), en een regelpagina
  openen waarop een grens én een handeling staan. Zet de spec in het
  juiste project in `tests/playwright/playwright.config.ts` — een spec
  die in geen enkele `testMatch` staat, draait niet en bewijst niets
  (dat ging hier eerder mis). Vul de fixture aan als dat nodig is; de
  bestaande specs moeten blijven slagen.
- [ ] 6.2 `docs/` bijwerken en CHANGELOG onder `[Unreleased]`: het
  vlootscherm, x/n in plaats van ja/nee (inclusief waarom onbekend niet
  meetelt), de grenzen op de regelpagina, en de handelingen.

Kun je Playwright niet draaien in de kooi (geen egress naar de
browser-CDN), schrijf de spec dan, zeg dat in je run-rapport, en vink
6.1 NIET af als geslaagd — hij wordt buiten de kooi nagemeten.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-20-vloot-en-regels --strict` groen.
Budget is $5.
