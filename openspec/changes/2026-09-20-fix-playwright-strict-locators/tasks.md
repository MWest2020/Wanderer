# Tasks: fix-playwright-strict-locators

## 1. Specs — run 01
- [x] 1.1 `reporting-catalogue.spec.ts`, `container-image-sovereignty.spec.ts`,
  `eu-package-origin.spec.ts`, `host-side-scoring.spec.ts`,
  `nextcloud-as-target.spec.ts`: scope elke regel-ID-locator binnen
  `table.rule-catalogue` (of de tabel die de test bedoelt), zodat hij op
  één element wijst.
- [x] 1.2 `dar.spec.ts`: maak de kop-locator exact, zodat `/Verdict/i`
  niet twee koppen raakt.
- [x] 1.3 Geen wijziging aan de UI-templates of -code.

## 2. Bewijs
- [ ] 2.1 `make playwright` groen (37/37) — in de kooi niet te draaien;
  Mark's sessie meet het na en vinkt af.
- [ ] 2.2 `openspec validate 2026-09-20-fix-playwright-strict-locators
  --strict` groen.

## 3. Afronding
- [ ] 3.1 Delta toepassen en de change archiveren.
