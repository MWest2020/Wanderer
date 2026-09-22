# Tasks: drie-lagen-ciso

## 1. De vlootscore — run 01
- [x] 1.1 Eén functie die een organisatie omzet in een vlootscore:
  som(x)/som(n), onbeantwoorde vragen apart, aantal domeinen niet
  soeverein, en de verdeling per stroom. Bouw op `BuildFleetScore`.
- [x] 1.2 De drie regels die over de vloot de meeste punten kosten.
- [x] 1.3 Tests incl. de valkuil: negen goede domeinen en één slecht.

## 2. `/ui/` wordt de vloot — run 02
- [x] 2.1 De vlootscore, de verdeling en de lijst (slechtste eerst).
- [x] 2.2 Het invoerveld blijft, als actie binnen de pagina.
- [x] 2.3 De oude "recent beantwoord"-lijst verdwijnt of gaat op in de
  vlootlijst; geen twee overzichten naast elkaar.

- [x] 2.4 De lijst "kost de vloot de meeste punten" toont nu de
  `Description` van de regel, en die is geformuleerd als de gewenste
  eindtoestand ("The domain has a direct registrar relationship, with no
  reseller layer. — 3 domeinen"). Daardoor leest een falende regel als
  een geslaagde. Toon het probleem, niet het doel: "faalt op 3 van 3
  domeinen" met de probleemzin uit de regel. De telling zelf klopt
  (TopConcerns telt alleen `afhankelijk`).
- [x] 2.5 Twee frameworks leveren dezelfde bevinding dubbel in die top 3
  ("TLS certificate issued by a CA registered in the EU" naast "issued
  by an authority in the EEA"). Ontdubbel op stroom/onderwerp, of toon
  één regel per onderwerp met de frameworks erbij.

## 3. Het domein — run 03
- [x] 3.1 x/n naast de oordeelzin op de antwoordpagina.
- [x] 3.2 Per niet-soeverein punt de handeling uit de tekstentabel.

## 4. Taal — run 04
- [x] 4.1 De oordelen van de wand-regels in het Nederlands, uit de
  bestaande tabel; Engels alleen nog in het bewijs.
- [x] 4.2 Een test die een Engelstalig oordeel buiten het bewijs vangt.

## 4b. Wat er ná run 04b nog Engels is (live gezien op de
## antwoordpagina, 2026-09-22)
- [x] 4.4 De stroomlabels zijn nog Engels: "Certificate", "Transit
  path", "Third parties", "CDN / hyperscaler". Ze staan in
  `flowRules` (`internal/ui/flows.go`). De oordelen eronder zijn
  Nederlands, dus de regel leest nu half om.
- [x] 4.5 De navigatie mengt talen: "Overview · Vloot · Trends".
- [x] 4.6 De kopzin dubbelt: "Nee — Hosting: De hosting staat buiten de
  EER — apex-adressen in CA". De stroomnaam staat er twee keer in.
  Kies één vorm.
- [x] 4.7 Een ja-regel begint met kleine letter ("Ja — de mail wordt
  binnen de EER gerouteerd"), een nee-regel met een hoofdletter ("De
  hosting staat buiten de EER"). Eén vorm kiezen.

## 5. Techniek naar achteren — run 05
- [x] 5.1 De onderbouwing opent met de zeven vragen; de twee
  frameworktabellen achter één klik.
- [x] 5.2 Opvolger van ADR-0017 met de nieuwe rolverdeling.
- [x] 5.3 `assessment.tmpl` opent met `<html lang="en">` terwijl de
  pagina Nederlands is; op `nl` gezet waar de inhoud Nederlands is, de
  overige templates gecontroleerd op hetzelfde.

## 6. Bewijs
- [x] 6.0 Twee specs uit `answer-first-flow.spec.ts` zoeken
  `form.door-form`, terwijl het scanformulier sinds run 02
  `class="scan-form"` heet en onderaan de vlootpagina staat. Zij falen
  daarop (gemeten 2026-09-22: 43 geslaagd, 2 gefaald; de
  contrastfouten waren toen al weg). Breng die twee in lijn met de
  nieuwe indeling — verander de UI NIET om een oude spec te plezieren.
- [x] 6.1 Playwright: vlootscore zichtbaar
  (`answer-first-flow.spec.ts:60`, `.fleet-score-total`), slechtste
  domein bovenaan (`vloot-en-regels.spec.ts:46`), handeling op de
  antwoordpagina (`vloot-en-regels.spec.ts:93`,
  `.answer-remediation`). Alle drie in het `scan-dev`-project, dat ze
  in zijn `testMatch` noemt — nagemeten: 45 geslaagd, 0 gefaald.
- [x] 6.2 docs + CHANGELOG. `docs/how-to/operator.md` beschreef nog de
  DAR-indeling die ADR-0018 vervangt; er staan nu drie lagen met de
  routes erbij. CHANGELOG onder [0.8.1].
