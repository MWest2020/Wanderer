# Habitat run 02 — /ui/ wordt de vloot (tasks 2.1–2.3)

Contract: `openspec/changes/2026-09-22-drie-lagen-ciso/`
(specs/web-ui/spec.md, requirement "De vloot is de eerste laag").

Achtergrond: `/ui/` toont nu een invoerveld plus "recent beantwoord" —
drie losse antwoorden. Wie binnenkomt met veertig domeinen ziet geen
totaal. De vlootscore uit run 01 (`internal/ui/fleet_score.go`) levert
alles wat daarvoor nodig is.

## Scope — ONLY these tasks
- [ ] 2.1 `/ui/` (en `/ui/orgs/{slug}`) opent met de vloot van de
  gekozen organisatie:
  - de score voor het geheel als `x/n`, met het aantal onbeantwoorde
    vragen apart;
  - het aantal domeinen dat niet soeverein is, direct naast die score —
    dit is de valkuil uit de proposal, een hoge score mag een slecht
    domein niet verbergen;
  - de verdeling per stroom ("Mail: 3 van 5 buiten de EER");
  - de drie regels die over de vloot de meeste punten kosten;
  - de domeinen, standaard gesorteerd op de slechtste eerst.
- [ ] 2.2 Het invoerveld blijft, maar als actie binnen de pagina — niet
  als het eerste en grootste element.
- [ ] 2.3 Geen twee overzichten naast elkaar: "recent beantwoord"
  verdwijnt of gaat op in de vlootlijst. Het bestaande vlootbeheer
  (toevoegen/verwijderen op `/ui/orgs/{slug}/fleet`) blijft waar het is;
  verwijs ernaar.

## Out of scope
De antwoordpagina (run 03), de taal van de oordelen (run 04), en de
onderbouwing (run 05).

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen `go test ./internal/ui/...`; bewaar één
volledige build/vet/test voor het eind (de eerste duurt minuten).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-drie-lagen-ciso --strict` groen.
Budget is $5.
