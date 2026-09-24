# Tasks: vloot-in-beeld

## 1. De vlootlaag in beeld — habitat run 01
- [ ] 1.1 Scan-invoerveld bovenaan `/ui/` en `/ui/orgs/{slug}`.
- [ ] 1.2 Vlootscore als ring (SVG), segmenten in Go berekend, met label.
- [ ] 1.3 Per stroom als gestapelde balk per stroom, met label.
- [ ] 1.4 Domeinen als raster domein × stroom, slechtste eerst, rij linkt naar
      de antwoordpagina; de "zwaarste"-zin vervalt.
- [ ] 1.5 Top-3 als stroom + handeling + balk; geen rationale op `/ui/`.
- [ ] 1.6 Organisations-tabel alleen bij meer dan één organisatie.
- [ ] 1.7 Tests voor elk scenario (Go + Playwright), elk één keer rood gezien
      met de reparatie eruit.

## 2. Uit
- [ ] 2.1 `make playwright`, lint nul, screenshot vóór/na op een kopie van de
      prod-data.
- [ ] 2.2 Release, deploy, nagekeken op de live instantie.
- [ ] 2.3 ADR-0018 bijwerken (rendered surface); archiveren.
