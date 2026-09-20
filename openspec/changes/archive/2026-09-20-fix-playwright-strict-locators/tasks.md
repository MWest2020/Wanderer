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
- [x] 2.1 `make playwright` groen: **37 van 37** (Chromium 147, 2026-09-20, buiten de kooi — de kooi kan de browser niet downloaden).

- [x] 2.2 `openspec validate 2026-09-20-fix-playwright-strict-locators
  --strict` groen.

## 3. Afronding
- [x] 3.1 Delta toepassen en de change archiveren.
