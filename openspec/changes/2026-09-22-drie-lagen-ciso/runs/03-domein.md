# Habitat run 03 — het domein (tasks 3.1–3.2, 2.4, 2.5)

Contract: `openspec/changes/2026-09-22-drie-lagen-ciso/`
(specs/web-ui/spec.md, requirement "Het domein toont zijn score en wat
eraan te doen is").

## Scope — ONLY these tasks
- [ ] 3.1 De antwoordpagina van één domein toont `x/n` naast de
  oordeelzin (uit `BuildFleetScore`, internal/ui/fleet_score.go — niet
  opnieuw tellen), met de onbeantwoorde vragen erbij.
- [ ] 3.2 Bij elk niet-soeverein punt staat de handeling. Die zit sinds
  vandaag óók in de JSON-uitvoer (`handeling` per rationale), maar de UI
  leest hem rechtstreeks uit de tabel — houd dat zo, en vul de
  plaatshouder `{domein}` in.
- [ ] 2.4 De lijst "kost de vloot de meeste punten" (op `/ui/`) toont nu
  de `Description` van de regel, en die beschrijft de GEWENSTE toestand:
  "The domain has a direct registrar relationship, with no reseller
  layer. — 3 domeinen" leest alsof het goed gaat, terwijl die drie
  domeinen er juist op falen. Toon het probleem: de regel die faalt, op
  hoeveel van hoeveel domeinen, in de formulering van het gebrek. De
  telling zelf klopt (TopConcerns telt alleen `afhankelijk`) — raak die
  niet aan.
- [ ] 2.5 In diezelfde lijst staat hetzelfde certificaatprobleem twee
  keer, één keer per framework ("issued by a CA registered in the EU" en
  "issued by an authority in the EEA"). Ontdubbel op onderwerp en noem
  de frameworks erbij, zodat een lezer één bevinding ziet.

## Out of scope
De taal van de oordelen (run 04) en de onderbouwing (run 05).

## Over bouwen en testen in de kooi
Draai tijdens het werk `go test ./internal/ui/...`; bewaar één volledige
build/vet/test voor het eind.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-drie-lagen-ciso --strict` groen.
Budget is $5.
