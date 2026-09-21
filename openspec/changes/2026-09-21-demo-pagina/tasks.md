# Tasks: demo-pagina

## 1. De route — run 01
- [ ] 1.1 `demo.target` in de serve-configuratie; leeg = geen demo, en
  dat staat één keer in het opstartlog.
- [ ] 1.2 `/demo` toont de laatste voltooide scan van dat domein:
  antwoord + onderbouwing, met de scandatum. Hergebruik de bestaande
  weergave; bouw geen tweede.
- [ ] 1.3 Geen ander domein in de pagina, geen links naar vloot,
  organisaties of regelpagina's, geen scanknop.
- [ ] 1.4 Geen scan te starten via de demoroute, ook niet met een
  handmatig POST-verzoek.
- [ ] 1.5 Tests: demo uit → 404; demo aan zonder scan → "nog geen
  meting"; demo aan met scan → het oordeel; en een test die faalt als de
  pagina de naam van een ander target bevat.

## 2. Uitrol
- [ ] 2.1 `demo.target: westerweel.work` in de homelab-configuratie, en
  westerweel.work op het scanschema.
- [ ] 2.2 De tunnel laat `/demo` door; de rest blijft zoals het is.
- [ ] 2.3 Van buitenaf nameten: `/demo` zonder sleutel werkt, `/ui/`
  blijft naar de login sturen.

## 3. Documentatie
- [ ] 3.1 docs + CHANGELOG: wat de demo is, en waarom er geen scanknop
  op staat.
