# Habitat run 05c — de vijf oude specs verhuisd (task 5.3, afgerond)

Contract: `openspec/changes/2026-09-20-answer-first-ui/`, task-ref
`runs/05b-oude-specs.md`.

## Wat er gedaan is

Alle vijf falende tests uit run 05b zijn aangepast. Twee verschillende
oorzaken bleken te spelen, niet één:

1. **Vier tests keken op de verkeerde pagina.** De vlootlijst
   (`section.targets-fleet`), de verdict-pillen en de sovereignty-rollup
   (`section.sovereignty-rollup`) zijn door run 02 van `/ui/` naar
   `/ui/trends` verhuisd. De tests gingen nog naar `/ui/`. Aangepast
   naar `/ui/trends`:
   - `dar.spec.ts` — "Overview slimness" test: nu twee stappen in één
     test: eerst bevestigen dat de deur (`/ui/`) geen van deze secties
     draagt (`toHaveCount(0)`), dan naar `/ui/trends` om te bevestigen
     dat ze daar wél staan.
   - `ui-personas.spec.ts` — "Overview leads with the target fleet" en
     "Clicking a target opens its domain-titled report": beide
     `page.goto("/ui/")` → `page.goto("/ui/trends")`. Describe-titel
     "Overview = Tourist" → "Trends = Tourist" omdat de Tourist-rol nu
     op Trends zit, niet op de deur.
   - `sovereignty-overview.spec.ts` — "instance dashboard rolls flows
     up across targets": `page.goto("/ui/")` → `page.goto("/ui/trends")`.

2. **Eén test beschreef markup die bewust is vervangen, niet verhuisd.**
   `sovereignty-overview.spec.ts`'s eerste test ("assessment page shows
   the synthesis panel") bleef op de assessment-pagina staan — die
   pagina zelf is niet verplaatst — maar run 04 (task 4.1, "bouw geen
   tweede weergave") heeft de oude `table.flows`-rijen in
   `section.sovereignty-overview` al vervangen door de gedeelde
   answer-row-markup (`article.answer-row`, `.answer-question`,
   `details.answer-evidence`) die ook de reasoning-pagina gebruikt. De
   kop is nu "Onderbouwing — soevereiniteit", niet "Sovereignty
   overview". Ik heb de assertions aangepast aan de nieuwe markup
   (`.answer-row`, `.answer-question` met een case-insensitive
   mail/dns/hosting-match op de Nederlandse vraagtekst) in plaats van de
   UI terug te zetten naar een tabel. De hub-and-spoke SVG-assertions
   (`svg.sov-diagram`, `circle.hub`, `circle.node`) waren en zijn
   ongewijzigd.

`internal/ui` is niet aangeraakt — geen van de vijf tests legde een
echt gat in de UI bloot; het was in alle gevallen de spec die de oude
indeling beschreef.

`answer-first-flow.spec.ts` is niet aangeraakt.

## Wat niet gemeten kon worden

Playwright kon in de kooi niet draaien (`npx playwright install` heeft
geen egress naar de browser-CDN; er is hier ook geen `node_modules`
voor de testsuite). De aanpassingen zijn puur op de spec-tekst gedaan,
geverifieerd door de daadwerkelijke Go-templates en -handlers te lezen
(`internal/ui/templates/{dashboard,trends,assessment}.tmpl`,
`internal/ui/ui.go`, `internal/ui/flows.go`) in plaats van te gokken:
elke aangepaste selector is nagetrokken tegen de exacte HTML die de
betreffende route nu rendert. De suite moet buiten de kooi opnieuw
gemeten worden.

## Done means — bewijs

- `go build ./...` → groen.
- `go vet ./...` → groen.
- `go test ./...` (offline, vendor) → alle packages `ok`, inclusief
  `internal/ui`.
- `openspec validate 2026-09-20-answer-first-ui --strict` → "Change
  '2026-09-20-answer-first-ui' is valid".

Kosten deze run: ruim onder het budget van $4 (research via
subagents + tekstuele spec-edits, geen Go-wijzigingen nodig).
