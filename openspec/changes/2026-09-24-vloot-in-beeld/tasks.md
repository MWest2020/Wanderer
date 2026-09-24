# Tasks: vloot-in-beeld

## 1. De vlootlaag in beeld — habitat run 01
- [x] 1.1 Scan-invoerveld bovenaan `/ui/` en `/ui/orgs/{slug}`.
- [x] 1.2 Vlootscore als ring (SVG), segmenten in Go berekend, met label.
- [x] 1.3 Per stroom als gestapelde balk per stroom, met label.
- [x] 1.4 Domeinen als raster domein × stroom, slechtste eerst, rij linkt naar
      de antwoordpagina; de "zwaarste"-zin vervalt.
- [x] 1.5 Top-3 als stroom + handeling + balk; geen rationale op `/ui/`.
- [x] 1.6 Organisations-tabel alleen bij meer dan één organisatie.
- [x] 1.7 Tests voor elk scenario (Go + Playwright), elk één keer rood gezien
      met de reparatie eruit.

## 1b. Nagekeken op prod-data — habitat run 02
Screenshots van run 01 op een kopie van de prod-data (2026-09-24, licht en
mobiel) lieten vier dingen zien:
- [ ] 1b.1 Top-3 toont een kale regel-ID bij regels zonder stroom
      (`wand.operationeel.caa_restricts_issuance`,
      `wand.accountability.ns_holder_transparent`), terwijl beide een
      handeling hebben.
- [ ] 1b.2 Het aantal onbeantwoorde vragen staat nergens meer zichtbaar (alleen
      in het aria-label); de spec eist het apart.
- [ ] 1b.3 De letters in het raster (S/V/N/?) hebben geen legenda.
- [ ] 1b.4 Mobiel (390 px): de stroombalken zijn nul breed, en het raster laat
      de hele pagina horizontaal scrollen.

## 2. Uit
- [ ] 2.1 `make playwright`, lint nul, screenshot vóór/na op een kopie van de
      prod-data.
- [ ] 2.2 Release, deploy, nagekeken op de live instantie.
- [ ] 2.3 ADR-0018 bijwerken (rendered surface); archiveren.
