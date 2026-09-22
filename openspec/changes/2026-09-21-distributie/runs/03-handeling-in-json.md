# Habitat run 03 — de handeling in de JSON-uitvoer (tasks 3.1–3.2)

Contract: `openspec/changes/2026-09-21-distributie/`.

Waargenomen 2026-09-22 door de Action echt te draaien: de
job-samenvatting meldde bij elk falend punt "Geen kant-en-klare
handeling beschikbaar", terwijl `internal/assessor/wand/accountability_nl.yaml`
**28 handelingen** bevat en `TestEveryRuleHasHandeling` alle regels dekt.
De UI leest die tabel rechtstreeks; de Action leest de JSON-uitvoer van
`wanderer assess`, en daar zit de handeling niet in.

## Scope — ONLY these tasks
- [ ] 3.1 Elke rationale die niet soeverein scoort draagt in de
  JSON-uitvoer de bijbehorende handeling. Denk aan de scheiding: de
  handelingen-tabel woont in `internal/assessor/wand` (pakket-eigen
  copy), de JSON-weergave in `internal/assessor`. Laat het
  wand-pakket zijn handeling meegeven bij het bouwen van de rationale
  in plaats van de renderer in de tabel te laten graaien — motiveer je
  keuze in het run-rapport als je het anders oplost.
  Veldnaam: `handeling`, weggelaten wanneer leeg (additief; oude
  uitvoer blijft leesbaar).
- [ ] 3.2 `scripts/action-report.sh` (+ `action-report.jq`) gebruikt dat
  veld en toont de plaatsvervangende zin alleen nog als er echt geen
  handeling is.
- [ ] Tests: een falende regel levert JSON mét handeling; een soevereine
  regel heeft er geen; het jq-filter toont de echte tekst.

## Let op
Geen scoringsgedrag wijzigen. `--format markdown` en de UI blijven
werken zoals ze doen.

## Over bouwen en testen in de kooi
`jq` zit in de image. Draai tijdens het werk
`go test ./internal/assessor/...` en bewaar één volledige build/vet/test
voor het eind.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-distributie --strict` groen. Budget is $4.
