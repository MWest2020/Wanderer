# Tasks: drift-land-per-adres

## 1. Code — habitat run 01
- [x] 1.1 `ipCountryChanged` vergelijkt per adres (`ip.asn`-attribuut
      `address`); het adres komt in de attributen van de melding.
- [x] 1.2 Tests voor de drie scenario's; de bestaande test blijft groen; de
      eerste nieuwe test één keer rood gezien met de oude code.

## 2. Uit
- [x] 2.1 Nagemeten op een kopie van de prod-data: de scans van 2026-09-28
      opnieuw door `Diff` → 0 landwissels. Release, deploy, archiveren.
      (westerweel.work 7 → 0, rijksoverheid.nl 1 → 0, beide drift.no_changes;
      de oude binary geeft op hetzelfde paar de 7 terug. v0.12.2 live.)
