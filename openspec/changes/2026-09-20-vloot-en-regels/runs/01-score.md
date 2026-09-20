# Habitat run 01 — x van n (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/` (proposal
"Niet ja/nee, maar x/n"; specs/web-ui/spec.md, requirement "Het
vlootscherm scoort x van n, niet ja of nee").

## Scope — ONLY these tasks
- [ ] 1.1 Eén functie in `internal/ui` die een Assessment omzet in een
  score: `x` = het aantal vragen dat soeverein is beantwoord, `n` = het
  aantal dat beantwoord kón worden, plus apart het aantal
  onbeantwoorde vragen. Een onbeantwoorde vraag telt NOOIT mee in `n`
  en geldt NOOIT als geslaagd. Bouw voort op wat er al is:
  `BuildAnswerVerdict` (internal/ui/answer.go) bepaalt al welke stromen
  meetellen en welke onbekend zijn — hergebruik die indeling, maak geen
  tweede telling die morgen uiteenloopt.
- [ ] 1.2 De functie geeft ook de zwaarste openstaande bevinding terug
  (stroom + de zin uit het oordeel). `BuildAnswerVerdict` kiest die al
  voor de kop; lever hem mee in plaats van hem opnieuw te bepalen.
- [ ] 1.3 Table-driven tests: alles soeverein (7/7, 0 onbekend); één
  afhankelijk (6/7 met die stroom als zwaarste); alles onbekend (0/0 met
  alles onbekend, en dat mag geen deling door nul geven); twee
  assessments met dezelfde x/n maar verschillend aantal onbekenden zijn
  niet gelijk.

Geen templates en geen routes in deze run — dat is run 03. Deze run
levert de functie en de tests.

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen `go test ./internal/ui/...` en bewaar één
volledige `go build ./...` + `go vet ./...` + `go test ./...` voor het
eind. De eerste volledige build duurt minuten (gevendorde SQLite); dat
is normaal. Zet GOFLAGS of GOPROXY niet om.

## Done means
Die volledige build/vet/test groen en `openspec validate
2026-09-20-vloot-en-regels --strict` groen. Budget is $4.
