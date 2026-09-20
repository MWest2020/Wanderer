# Habitat run 05 — bewijs en documentatie (tasks 5.1–5.2)

Contract: `openspec/changes/2026-09-20-answer-first-ui/`.

## Scope — ONLY these tasks
- [ ] 5.1 Playwright-spec: domein invullen → antwoord → onderbouwing,
  plus het geval "onbekend is geen ja". Zet de spec in het juiste
  project in `tests/playwright/playwright.config.ts` — een spec die in
  geen enkele `testMatch` staat, draait niet en bewijst niets (dat is
  hier eerder misgegaan). Vul de fixture aan als dat nodig is.
- [ ] 5.2 `docs/` bijwerken en CHANGELOG onder `[Unreleased]`; ADR-0017
  krijgt een addendum dat de antwoord-eerst-indeling beschrijft — het
  is een wijziging van de informatiearchitectuur die die ADR vastlegt,
  dus daar hoort het.

Kun je Playwright niet draaien in de kooi (geen egress naar de
browser-CDN), schrijf de spec dan, zeg dat in je run-rapport, en vink
5.1 NIET af als geslaagd — hij wordt buiten de kooi nagemeten.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` groen.
Budget is $4.
