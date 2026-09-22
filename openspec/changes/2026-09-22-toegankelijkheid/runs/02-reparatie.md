# Habitat run 02 — de reparatie (tasks 2.1–3.1)

Contract: `openspec/changes/2026-09-22-toegankelijkheid/`
(specs/web-ui/spec.md).

De gate uit run 01 staat en de suite is daardoor rood: 39 geslaagd,
6 gefaald. Gemeten buiten de kooi met Chromium:

```
.score-voldoende      1.82  (#84cc16 op #f0f9e3)   ← ergste
.score-afhankelijk    3.40  (#d9534f op #faeaea)
.score-soeverein      3.89  (#198754 op #e3f1ea)
.fleet-score-total    3.40
```

## Scope — ONLY these tasks
- [ ] 2.1 Alle oordeelkleuren halen ten minste 4,5:1 bij hun
  werkelijke grootte (13,6px, normaal gewicht). Kies donkerdere
  voorgrondkleuren of lichtere achtergronden — en reken het na in plaats
  van te schatten; de gate meet het toch.
- [ ] 2.2 Kleur is nooit het enige onderscheid: elk oordeel draagt ook
  zijn woord ("ja", "nee", "onbekend", "n.v.t."). Waar nu alleen een
  gekleurde pil staat zonder tekst, komt tekst erbij.
- [ ] 2.3 Geen horizontale paginascroll op 390px: de brede tabellen op
  trends en de onderbouwing scrollen binnen hun eigen kader.
- [ ] 2.4 De lege tabelkop op trends krijgt een naam.
- [ ] 2.5 De suite is groen — inclusief de axe-controles uit run 01.
- [ ] 3.1 `docs/` + CHANGELOG, met de zin die in de spec staat: een
  groene run betekent "geen geautomatiseerd te vinden fouten van dit
  type", niet "toegankelijk".

## Let op
Raak de oordeelsLOGICA niet aan — dit gaat alleen over hoe het eruitziet
en of het leesbaar is. Dezelfde findings horen dezelfde oordelen te
geven.

## Je kunt de suite hier niet draaien
Geen egress naar de Playwright-CDN. Reken het contrast uit (de formule
staat in WCAG 2.1, of gebruik een bekende tabel), zeg in je run-rapport
welke waarden je hebt gekozen en wat je berekende, en laat mij het
nameten.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-toegankelijkheid --strict` groen.
Budget is $5.
