# Tasks: toegankelijkheid

## 1. axe in de suite — run 01
- [x] 1.1 `@axe-core/playwright` (of axe-core ingespoten) in
  `tests/playwright`, met een helper die een pagina toetst en faalt op
  `serious`/`critical`.
- [x] 1.2 Toepassen op de schermen die de suite al bezoekt: vloot,
  antwoord, onderbouwing, trends, regelpagina, demo.
- [x] 1.3 De bestaande fouten worden zichtbaar: laat de suite falen en
  schrijf in je run-rapport wélke schermen falen en waarop. Repareer ze
  in deze run NIET — dat is run 02, zodat de gate en de reparatie apart
  te lezen zijn.

## 2. De reparatie — run 02
- [x] 2.1 Oordeelkleuren op ten minste 4,5:1 bij hun werkelijke grootte.
- [x] 2.2 Kleur nooit als enige onderscheid.
- [x] 2.3 Brede tabellen scrollen binnen hun eigen kader; geen
  horizontale paginascroll op 390px.
- [x] 2.4 De lege tabelkop op trends krijgt een naam.
- [ ] 2.5 De suite is groen. (kon niet zelf draaien — geen egress; zie
  run-rapport)

## 3. Documentatie
- [x] 3.1 Wat de gate wel en niet bewijst, in `docs/`; CHANGELOG.
