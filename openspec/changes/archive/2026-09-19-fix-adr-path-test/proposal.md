# Proposal: ADR-pad na het docs-contract — maak main weer groen

## Why

`apply-docs-contract` verplaatste de ADR's van `docs/decisions/` naar
`docs/explanation/adr/`. De doc-lint-test
`TestPlaywrightCoverage_ADRsWithUISurface`
(`internal/ui/playwright_coverage_test.go`) leest nog het oude pad en faalt
sindsdien op elke `go test ./...` — lokaal én in CI. Main is dus rood, en elke
PR erft die rode test (waargenomen op #7).

De spec `project-hygiene` noemt op drie plekken nog `docs/decisions/`; de spec
beschrijft dus een map die niet meer bestaat.

## What Changes

- De test leest `docs/explanation/adr/`. Verder niets aan de testlogica.
- `project-hygiene`: drie requirements noemen voortaan
  `docs/explanation/adr/`.

Lokaal in een wegwerp-worktree gecontroleerd dat alleen de padwijziging de
test groen maakt (vier ADR's met `## UI surface`, alle vier met een spec).

## Scope

**In:** het pad in de test en in de spec.
**Out:** alles anders. Deze change staat los van de accountability-change.

## Build process

Eén habitat-run (builder), task-ref `runs/01-fix.md`. Branch
`fix/adr-path-test` draagt ook de vendor-commit, zodat de kooi offline kan
bouwen en deze PR de eerste groene CI op main oplevert.
