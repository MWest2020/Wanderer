# Habitat run 05 — de regelpagina met advies (tasks 5.1–5.3)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/`
(specs/web-ui/spec.md, requirements "Een regel legt zichzelf uit,
inclusief de drempel" en "Bij elk oordeel 'nee' staat wat je eraan
doet").

## Scope — ONLY these tasks
- [ ] 5.1 De regelpagina (`/ui/reporting/{framework}/{ruleID}` bestaat
  al) toont per regel: wat hij controleert (`Description`), waarom het
  uitmaakt (`Rationale`), welke waarneming hij gebruikt, en zijn
  grenzen uit `Rule.Thresholds` (run 04) met hun uitleg. Heeft een regel
  geen grens, zeg dat dan ("deze regel kijkt of iets aanwezig is") in
  plaats van een leeg kopje te tonen. Daaronder: welke domeinen nu aan
  welke kant staan — `RuleTargetRows` levert dat al.
- [ ] 5.2 Bij een falend oordeel staat één concrete handeling, zowel op
  de regelpagina als op de onderbouwingspagina. Volg het patroon van
  `accountability_nl.yaml`: de tekst staat in de tabel, niet in Go of in
  een template, en noemt het domein via een parameter.
- [ ] 5.3 Alle regels van het wand-pakket krijgen hun handeling in die
  tabel, en een test faalt als een regel er geen heeft — net zoals
  `TestEveryRuleHasRationale` dat nu voor `Rationale` doet. Schrijf
  handelingen die zeggen wát te doen en waar ("vraag je registrar de
  privacyproxy op {domein} te verwijderen"), niet een doel herhalen
  ("zorg voor een herkenbare registrant"). Kun je voor een regel geen
  eerlijke handeling bedenken omdat de oplossing buiten de macht van de
  operator ligt, schrijf dat dan op als handeling en noem hem in je
  run-rapport.

## Out of scope
Het eucsf-pakket (alleen wand in deze run), en de openstaande
ontwerpvraag uit taak 4.3 (drempels die in de probe zitten).

## Over bouwen en testen in de kooi
Draai tijdens het werk `go test ./internal/ui/... ./internal/assessor/...`;
bewaar één volledige build/vet/test voor het eind (eerste build duurt
minuten).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-20-vloot-en-regels --strict` groen.
Budget is $6.
