# Tasks: vloot-en-regels

## 1. x van n — run 01
- [x] 1.1 Eén functie die een assessment omzet in `x/n` plus het aantal
  onbeantwoorde vragen; onbekend telt nooit als geslaagd en nooit in n.
- [x] 1.2 De zwaarste openstaande bevinding erbij (hergebruik wat de
  antwoordkop al bepaalt).
- [x] 1.3 Table-driven tests incl. alles onbekend, alles soeverein, en
  een gelijke score met verschillend aantal onbekenden.

## 2. De vloot bijhouden — run 02
- [x] 2.1 Domeinen toevoegen en verwijderen vanuit de UI, per
  organisatie; verwijderen raakt de scangeschiedenis niet.
- [x] 2.2 Per domein: laatste scan en het schema waaronder het valt.
- [x] 2.3 Alleen voor ingelogde gebruikers, net als scannen.

## 3. Het vlootscherm — run 03
- [x] 3.1 Alle domeinen met `x/n`, onbeantwoorde vragen apart, en de
  zwaarste openstaande bevinding.
- [x] 3.2 Verschil sinds de vorige scan per domein.
- [x] 3.3 Sorteren op score, op verandering, op laatste scan.

## 4. Drempels uit de code — run 04
- [x] 4.1 `Rule` draagt zijn grenzen als gegeven (naam, waarde,
  eenheid); regels zonder grens hebben een lege lijst.
- [x] 4.2 De bestaande regels vullen hun grenzen in; een test vangt het
  uiteenlopen van grens en vergelijking.

- [ ] 4.3 Open punt uit run 04: `variant_convergence` (8 paden, 24
  verbindingen) en `cert_validity` (30 dagen) beslissen NIET zelf op die
  getallen — die zitten in de probe, en de regel leest een voorgekookte
  vlag. Een drempel bij die regels zetten zou onfalsifieerbaar zijn.
  Eigen ontwerpvraag: wie bezit een drempel, de waarnemer of de
  beoordelaar? Niet in deze change oplossen.

## 5. De regelpagina — run 05
- [ ] 5.1 Per regel: wat, waarom, welke waarneming, welke grens, en wie
  aan welke kant staat.
- [ ] 5.2 De handeling ("hoe los ik dit op") bij een falend oordeel,
  ook op de onderbouwingspagina.
- [ ] 5.3 Alle regels krijgen hun handeling in de tekstentabel; een
  regel zonder handeling faalt de tests.

## 6. Bewijs en documentatie — run 06
- [ ] 6.1 Playwright: domein toevoegen, vloot sorteren, regelpagina met
  grens en handeling. In de juiste `testMatch`.
- [ ] 6.2 docs + CHANGELOG.
