# Tasks: drie-lagen-ciso

## 1. De vlootscore — run 01
- [x] 1.1 Eén functie die een organisatie omzet in een vlootscore:
  som(x)/som(n), onbeantwoorde vragen apart, aantal domeinen niet
  soeverein, en de verdeling per stroom. Bouw op `BuildFleetScore`.
- [x] 1.2 De drie regels die over de vloot de meeste punten kosten.
- [x] 1.3 Tests incl. de valkuil: negen goede domeinen en één slecht.

## 2. `/ui/` wordt de vloot — run 02
- [ ] 2.1 De vlootscore, de verdeling en de lijst (slechtste eerst).
- [ ] 2.2 Het invoerveld blijft, als actie binnen de pagina.
- [ ] 2.3 De oude "recent beantwoord"-lijst verdwijnt of gaat op in de
  vlootlijst; geen twee overzichten naast elkaar.

## 3. Het domein — run 03
- [ ] 3.1 x/n naast de oordeelzin op de antwoordpagina.
- [ ] 3.2 Per niet-soeverein punt de handeling uit de tekstentabel.

## 4. Taal — run 04
- [ ] 4.1 De oordelen van de wand-regels in het Nederlands, uit de
  bestaande tabel; Engels alleen nog in het bewijs.
- [ ] 4.2 Een test die een Engelstalig oordeel buiten het bewijs vangt.

## 5. Techniek naar achteren — run 05
- [ ] 5.1 De onderbouwing opent met de zeven vragen; de twee
  frameworktabellen achter één klik.
- [ ] 5.2 Opvolger van ADR-0017 met de nieuwe rolverdeling.

## 6. Bewijs
- [ ] 6.1 Playwright: vlootscore zichtbaar, slechtste domein bovenaan,
  handeling op de antwoordpagina. In de juiste `testMatch`.
- [ ] 6.2 docs + CHANGELOG.
