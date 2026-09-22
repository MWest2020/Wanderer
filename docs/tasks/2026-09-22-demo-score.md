# Habitat run — de demopagina leidt nog met ja/nee, zonder score

De demopagina (`https://wanderer.westerweel.work/demo`,
`internal/ui/templates/demo.tmpl` + `internal/ui/demo.go`) is de enige
pagina die een bezoeker zonder inlog ziet. Hij is dus het visitekaartje,
en hij mist precies wat we in `2026-09-22-drie-lagen-ciso` hebben
afgesproken.

Nagemeten op de live pagina, 2026-09-22, met v0.8.0 uitgerold:

- De kopzin is `Nee — Hosting: De hosting staat buiten de EER —
  apex-adressen in CA (Cloudflare, Inc.).` Eén nee, geen score.
- Er staat nergens een x/n. De antwoordpagina achter de inlog toont die
  wél ("1/5 · 2 onbekend"); de demopagina heeft een eigen sjabloon en
  is bij die wijziging overgeslagen.
- De zeven vragen, oordelen en handelingen staan er wél in het
  Nederlands, met het waargenomen feit erbij. Dat deel is goed.

## Scope — ONLY these tasks
- [ ] 1.1 De demopagina toont dezelfde score als de antwoordpagina:
  x van n beantwoorde vragen, met de onbeantwoorde apart. Hergebruik
  wat `BuildAnswer`/`BuildFleetScore` al berekenen — bouw geen derde
  telling. Als dat een refactor van `demo.go` vraagt om dezelfde
  bouwstenen te gebruiken als de antwoordpagina: doe dat, want twee
  tellingen die uiteen kunnen lopen is erger dan één verplaatsing.
- [ ] 1.2 De kopzin leidt met de score, niet met een kale nee. Welke
  vorm precies is aan jou; de eis is dat een lezer in één zin ziet
  hoeveel er goed staat én wat het zwaarste openstaande punt is.
- [ ] 1.3 Een test die vastlegt dat de demopagina een score toont. Zet
  hem in het juiste Playwright-project (`testMatch` — een spec die
  nergens in staat, draait niet) óf als Go-test op de handler; kies en
  zeg waarom.

## Out of scope
De vloot- en antwoordpagina (die zijn af), nieuwe metingen, styling.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen. Playwright kun je niet draaien — zeg
dat in je rapport, ik meet na. Budget is $6.
