# Tasks: drift-land-per-adres

## 1. Code — habitat run 01
- [x] 1.1 `ipCountryChanged` vergelijkt per adres (`ip.asn`-attribuut
      `address`); het adres komt in de attributen van de melding.
- [x] 1.2 Tests voor de drie scenario's; de bestaande test blijft groen; de
      eerste nieuwe test één keer rood gezien met de oude code.

## 2. Uit
- [ ] 2.1 Nagemeten op een kopie van de prod-data: de scans van 2026-09-28
      opnieuw door `Diff` → 0 landwissels. Release, deploy, archiveren.
