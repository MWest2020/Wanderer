# Tasks: verwijderen-waar-je-kijkt

## 1. Code — habitat run 01
- [x] 1.1 Verwijderhandeling naast de domeinnaam, in het overzichtsraster en op
      het vlootscherm; `<details>`-bevestiging.
- [x] 1.2 Terugadres: alleen `/ui/`, `/ui/orgs/{slug}` of het vlootscherm van
      dezelfde organisatie; al het andere → vlootscherm.
- [x] 1.3 Vlootscherm-tabel in `.table-scroll`.
- [x] 1.4 Tests: Go voor het terugadres (ook de open-redirect), Playwright voor
      verwijderen vanaf `/ui/` en voor 390 px; elk één keer rood gezien.

## 1b. Nagekeken — habitat run 02
`make playwright` na run 01: drie rood.
- [x] 1b.1 Het overzicht toont een verwijderd domein nog. `/ui/` bouwt uit
      scans (`buildSnapshots` → `ListScans`), en alleen `ListFleetDomains`
      filtert op `removed_at`. Het domein telt dus ook nog mee in de
      vlootscore. De bestaande spec zegt al "verdwijnt uit het overzicht".
- [x] 1b.2 Twee sorteertests vergeleken de hele eerste cel met de domeinnaam;
      daar staat nu ook de verwijderhandeling in.

## 2. Uit
- [ ] 2.1 Release, deploy, nagekeken op prod-data; archiveren.
