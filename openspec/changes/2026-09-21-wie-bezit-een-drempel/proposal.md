# Proposal: wie bezit een drempel — de waarnemer of de beoordelaar?

## Why

Bij het zichtbaar maken van beslisgrenzen (change
`2026-09-20-vloot-en-regels`, run 04) bleek dat van circa dertig regels
er maar **twee** zelf op een vast getal beslissen. De rest leest een
voorgekookte vlag van een probe:

- `wand.operationeel.cert_validity` oordeelt op `expiring_soon`, en die
  vlag wordt op **30 dagen** gezet in `internal/probe/tls`.
- `wand.operationeel.variant_convergence` telt paden die de probe
  aanleverde; **8 paden** en een budget van **24 verbindingen** staan in
  `internal/probe/variants`.

De regelpagina toont nu netjes "deze regel kijkt of iets aanwezig is —
er is geen numerieke grens". Dat klopt voor de regel, en het is
misleidend voor de lezer: de grens bestaat wél, hij staat alleen
ergens anders. Iemand die wil weten waarom zijn certificaat "bijna
verlopen" heet, krijgt geen getal te zien.

De run die dit vond, weigerde terecht een drempel te verzinnen: een
`Threshold` bij een regel die er nooit tegen vergelijkt, maakt de test
die drift moet vangen onfalsifieerbaar. Het probleem is dus niet
vergeten, het is **verkeerd geplaatst**.

## De vraag

Een drempel hoort bij degene die hem toepast. Bij `domain_expiry` is
dat de regel. Bij `cert_validity` is dat de probe — die beslist wat
"binnenkort" betekent en levert een vlag. Twee mogelijke antwoorden:

1. **De probe bezit hem en geeft hem mee.** De vlag wordt vergezeld van
   het getal waarop hij is gezet (`expiring_soon: true`,
   `expiring_soon_days: 30`). De regel hoeft niets te weten; de UI toont
   de grens uit de finding. Eerlijk over wie beslist, en het werkt ook
   voor findings van een agent op een andere versie.
2. **De regel bezit hem en de probe levert ruwe data.** De probe meldt
   `days_left`, de regel beslist op 30. Zuiverder scheiding, maar het
   verplaatst werk naar elke regel en breekt bestaande findings.

Voorkeur: **(1)**. Het sluit aan bij de bestaande scheiding — de
assessor is een pure lezer van findings, en `accountability_rules.go`
gaat nu al uit zijn weg om `internal/probe/variants` niet te importeren.
Het maakt de grens bovendien zichtbaar op de plek waar hij is toegepast,
en dat is precies wat een lezer nodig heeft.

## What Changes

- Probes die een oordeelvlag zetten, zetten het getal erbij waarop die
  vlag berust.
- De UI toont die grens op de regelpagina, met dezelfde uitleg als bij
  een regel-eigen grens, en zegt erbij dat de waarneming hem toepaste.
- Een test vangt het uiteenlopen: de vlag en het meegeleverde getal
  komen uit dezelfde constante.

## Scope / Not in scope

**In:** `cert_validity` (30 dagen) en `variant_convergence` (8 paden,
24 verbindingen) als eerste twee, plus de weergave.

**Out:** alle probes in één keer omzetten. Twee gevallen bewijzen het
patroon; de rest volgt wanneer iemand er tegenaan loopt.
