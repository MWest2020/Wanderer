# Habitat run — de CLI is de enige die de correlatie niet gebruikt

Run 03b heeft `Store.FindingsForAssessment` gebouwd en aangesloten op
de API, de UI, de scheduler en MCP. Eén aanroeper bleef over:

    cmd/wanderer/assess.go:72
        Dimensions: assessor.Assess(scan.Findings, rules),

Die leest de bevindingen van de scan zelf, dus de standards-dimensie
blijft daar "niet gemeten" ook als er wél een import op het doel
staat.

## Waarom juist deze telt

De CLI is de gedocumenteerde route voor deze hele functie. De how-to
die we schrijven luidt:

    internetnl results <id> --format findings --findings-out f.json
    wanderer import internetnl --db wanderer.db f.json
    wanderer assess <scan-id> --db wanderer.db

Stap drie is precies de aanroeper die de correlatie mist. Gemeten op
een verse database met beide imports: alle zes regels melden "no
internetnl.* findings imported — not measured", terwijl de
bevindingen in dezelfde database staan.

## Scope — ONLY this
- [ ] 1.1 `cmd/wanderer/assess.go` gebruikt
  `st.FindingsForAssessment(ctx, scan)` in plaats van
  `scan.Findings`, net als de andere vier aanroepers.
- [ ] 1.2 Een test op CLI-niveau: importeer web + mail op een doel,
  beoordeel een perimeter-scan via het commando, en verwacht dat
  `wand.standards.rpki` tien tests ziet en `wand.standards.ipv6`
  negen. Controleer hem één keer mét de reparatie eruit en zeg in je
  rapport dat je dat deed.
- [ ] 1.3 Kijk de overige aanroepers van `assessor.Assess(` na
  (`grep -rn 'assessor.Assess(' --include='*.go' .`). Op dit moment:
  api.go, mcp/tools.go, scheduler/assess.go, ui.go (2x) gebruiken de
  correlatie; `internal/fixtures/seed.go` gebruikt `persisted` — kijk
  of dat daar juist is (het is testdata-seeding, waarschijnlijk wel)
  en zeg wat je concludeert.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen. Lint kun je
niet draaien; zeg dat, ik meet na. Budget is $5.
