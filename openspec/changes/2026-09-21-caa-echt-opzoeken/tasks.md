# Tasks: caa-echt-opzoeken

## 1. De opzoeking — run 01
- [ ] 1.1 `LookupCAA` doet een echte DNS-vraag (CAA, type 257).
- [ ] 1.2 De boom omhoog tot het registreerbare domein; de finding legt
  vast op welke naam de records stonden.
- [ ] 1.3 De verdicttekst noemt de herkomst bij erven ("geërfd van
  voorbeeld.nl").
- [ ] 1.4 Tests met een stub-resolver: eigen records, geërfde records,
  nergens iets, en een test die faalt op een resolver die altijd leeg
  teruggeeft.

## 2. Afronding
- [ ] 2.1 Nalopen of er meer opzoekingen zijn die stilzwijgend niets
  teruggeven, en het antwoord opschrijven in het run-rapport.
- [ ] 2.2 docs + CHANGELOG: wat er mis was, sinds wanneer, en wat de
  eerdere CAA-oordelen waard waren.
