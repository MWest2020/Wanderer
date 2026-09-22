# Habitat run 05 — techniek naar achteren (tasks 5.1–5.2)

Contract: `openspec/changes/2026-09-22-drie-lagen-ciso/`.

## Scope — ONLY these tasks
- [ ] 5.1 De onderbouwingspagina (`/ui/scans/{id}/assessment`) opent al
  met de zeven vragen; daaronder staan twee frameworktabellen open en
  bloot. Dat is de explorer-laag, niet de boer-laag. Zet ze achter één
  klik (`<details>`, dicht bij het laden, met een samenvatting die zegt
  wat eronder zit en hoeveel regels — niet alleen "details"). De
  vragen-sectie blijft onaangeroerd.
- [ ] 5.2 Schrijf de opvolger van `docs/explanation/adr/0017-ui-personas.md`
  als een nieuwe ADR (0018). 0017 verdeelde Overview/Report/Trends over
  toerist/explorer/boer; sinds deze change is de verdeling anders:
  `/ui/` = vloot (toerist), de antwoordpagina = één domein met de zeven
  punten (boer), de onderbouwing = techniek (explorer). Zet 0017 op
  `superseded by 0018` en laat hem staan — niet verwijderen.
- [ ] 5.3 `assessment.tmpl` opent met `<html lang="en">` terwijl de
  pagina Nederlands is. Een schermlezer spreekt hem dan in het Engels
  uit. Zet het op `nl` waar de inhoud Nederlands is, en controleer de
  andere templates op hetzelfde. Voeg een test toe die het vastlegt.

- [ ] 4.4 De stroomlabels zijn nog Engels: "Certificate", "Transit
  path", "Third parties", "CDN / hyperscaler". Ze staan in `flowRules`
  (`internal/ui/flows.go`). De oordelen eronder zijn sinds run 04b
  Nederlands, dus elke regel leest nu half om.
- [ ] 4.5 De navigatie mengt talen: "Overview · Vloot · Trends".
- [ ] 4.6 De kopzin dubbelt de stroomnaam: "Nee — Hosting: De hosting
  staat buiten de EER — apex-adressen in CA". Kies één vorm; de
  stroomnaam hoeft er niet twee keer in.
- [ ] 4.7 Een ja-regel begint met kleine letter ("Ja — de mail wordt
  binnen de EER gerouteerd"), een nee-regel met een hoofdletter ("De
  hosting staat buiten de EER"). Eén vorm.

Die vier zag ik op de gerenderde pagina, niet in de code — draai het
zelf na met `go test ./internal/ui/...` en lees de verwachte tekst in
de tests.

## Out of scope
Nieuwe gegevens verzamelen, de vlootpagina, de taaltabel (run 04).

## Over testen in de kooi
Playwright kan hier niet draaien (geen browser-egress). Zeg in je
rapport welke tests je wél hebt gedraaid; ik meet de browser-kant na.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-drie-lagen-ciso --strict` groen.
Budget is $5.
