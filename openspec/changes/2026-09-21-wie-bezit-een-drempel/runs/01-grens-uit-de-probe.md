# Habitat run 01 — de probe geeft zijn grens mee (tasks 1.1–3.1)

Contract: `openspec/changes/2026-09-21-wie-bezit-een-drempel/`
(specs/assessor/spec.md, requirement "Een oordeelvlag draagt het getal
waarop hij berust").

Achtergrond: run 04 van de vloot-change vond dat `cert_validity` en
`variant_convergence` niet zelf op hun getallen beslissen. De 30 dagen
zitten in `internal/probe/tls`, de 8 paden en het budget van 24 in
`internal/probe/variants`. De regel leest een voorgekookte vlag, dus een
`Threshold` bij die regel zou nergens tegen vergeleken worden.

## Scope — ONLY these tasks
- [ ] 1.1 `internal/probe/tls`: de finding die `expiring_soon` zet,
  levert het getal en de eenheid mee waarop die vlag berust (30 dagen),
  uit DEZELFDE constante als de vergelijking. Bestaande attributen
  blijven staan; dit is additief, oude findings blijven leesbaar.
- [ ] 1.2 `internal/probe/variants`: idem voor het aantal paden (8) en
  het verbindingsbudget (24), uit de bestaande constanten.
- [ ] 1.3 Tests die het uiteenlopen vangen: als de vlag op een andere
  waarde wordt gezet dan het meegeleverde getal, falen ze. Leid de
  verwachting af uit de constante, niet uit een letterlijk getal in de
  test — anders verplaats je het probleem alleen.
- [ ] 2.1 De regelpagina toont zo'n grens met de vermelding dat de
  wáárneming hem toepaste (bijv. "de tls-waarneming hanteert 30 dagen"),
  in plaats van "deze regel kijkt of iets aanwezig is". Hergebruik de
  bestaande weergave van `Rule.Thresholds`; zet er geen tweede naast.
- [ ] 2.2 Een regel zonder enige grens — ook niet uit de finding —
  blijft zeggen dat hij op aanwezigheid kijkt.
- [ ] 3.1 docs + CHANGELOG onder `[Unreleased]`: de keuze (de probe
  bezit de grens en geeft hem mee) en het afgewezen alternatief (de
  regel beslist op ruwe data) kort vastleggen.

## Let op
Geen scoringsgedrag wijzigen: dezelfde findings horen dezelfde oordelen
te geven. De assessor blijft een pure lezer van findings — importeer
`internal/probe/...` NIET vanuit het assessor-pakket; de grens reist mee
in de finding.

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen de pakketten die je raakt; bewaar één
volledige build/vet/test voor het eind (de eerste duurt minuten).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-wie-bezit-een-drempel --strict` groen.
Budget is $5.
