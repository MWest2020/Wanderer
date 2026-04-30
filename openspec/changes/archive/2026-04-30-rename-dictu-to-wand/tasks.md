## 1. Package + rule rename

- [x] 1.1 Move `internal/assessor/dictu/` to `internal/assessor/wand/`; rename `package dictu` → `package wand` in every file
- [x] 1.2 String-replace `dictu.` → `wand.` in every rule's `ID` field in `internal/assessor/wand/rules.go`
- [x] 1.3 String-replace `dictu.` → `wand.` in every test file under `internal/assessor/wand/`
- [x] 1.4 Update all importing packages: `cmd/wanderer/assess.go`, `internal/api/api.go`, `internal/mcp/tools.go`, `internal/ui/registry.go`, `pkg/models/seal.go` (FrameworkDICTU → FrameworkWand), and the integration-test harness in `internal/assessor/wand/integration_test.go`

## 2. Schema migration

- [x] 2.1 Add migration version 4 to `internal/store/migrations.go`: `UPDATE assessments SET framework = 'wand', dimensions = REPLACE(dimensions, '"dictu.', '"wand.') WHERE framework = 'dictu'`
- [x] 2.2 Test in `internal/store/store_test.go::TestMigration_RenameDictuToWand`: seed an assessment row with `framework='dictu'` + `dictu.*` rationale-ID, run the migration SQL, assert the row was rewritten to `wand` and the criterium-ID inside the JSON was rewritten too
- [x] 2.3 Idempotency test: re-running the migration SQL leaves already-migrated rows untouched

## 3. CLI deprecated alias

- [x] 3.1 `cmd/wanderer/assess.go::selectedFrameworks` accepts `wand`, `dictu` (alias), `eucsf`, `seal`, `both`; the dictu arm resolves to FrameworkWand
- [x] 3.2 `warnIfDeprecatedFramework` emits one stderr line beginning with `warning:` when `--framework dictu` is used; silent for `wand` / `eucsf`
- [x] 3.3 Tests in `cmd/wanderer/assess_framework_test.go`: alias resolves to wand, deprecation warning fires, canonical paths are silent

## 4. UI registry alias

- [x] 4.1 `internal/ui/registry.go::lookupRule` switch arm: `case "wand", "dictu":` both resolve to `wand.DefaultRules()`
- [x] 4.2 `internal/ui/registry_test.go::TestLookupRule_DictuAlias` pins the alias path

## 5. ADRs

- [x] 5.1 Wrote `docs/decisions/0011-rename-dictu-to-wand.md` covering legal motivation, the wand naming rationale, the migration path, alternatives considered, and the credit to DICTU's framework as inspiration
- [x] 5.2 Appended addenda to `docs/decisions/0009-dual-framework-assessor.md` and `docs/decisions/0004-assessor-rule-engine.md` pointing at ADR-0011

## 6. Documentation sweep

- [x] 6.1 Updated `docs/assessor.md` — added "Inspired by" paragraph crediting DICTU; replaced `dictu` → `wand` in CLI examples + rule table
- [x] 6.2 Updated `docs/architecture.md` — diagram references wand, "How to add a wand rule" section, framework table updated
- [x] 6.3 Updated `docs/operator.md` — wand assessor mentioned, dimension reference uses neutral language
- [x] 6.4 Updated `docs/tutorial.md` — heritage credit for DICTU toets, sovereignty-dimension wording
- [x] 6.5 Updated `docs/findings.md` — `dictu.*` rule examples → `wand.*`; `internal/assessor/dictu` paths → `wand`; "DICTU dimension" → "sovereignty dimension"
- [x] 6.6 Updated `README.md` — wand pack credited as Conduction's; DICTU toetsingsinstrument credited as inspiration

## 7. Tests + CHANGELOG

- [x] 7.1 Full suite green: 25/25 packages OK after rename
- [x] 7.2 `TestFrameworkRename_RegressionGuard` in `internal/assessor/wand/rules_test.go` asserts every registered rule starts with `wand.`, none start with `dictu.`
- [x] 7.3 CHANGELOG entries: `### Changed (breaking)` for the rename, `### Added` for the deprecated dictu alias on CLI + UI registry; the migration is described in the Changed entry

## 8. Validation

- [x] 8.1 `openspec validate --specs --strict` — 9 / 9 passed
- [x] 8.2 Manual smoke: rebuilt /tmp/wanderer, opened the demo DB from the previous session (/tmp/w.db, which contained `framework='dictu'` rows from before the rename); migration v4 fired on Open, the export now reports `framework: "wand"` and every criterium_id is `wand.*`. The `--framework dictu` alias warning fires; `--framework wand` is silent and produces a wand-labelled assessment.
- [x] 8.3 Archive moved to `openspec/changes/archive/2026-04-30-rename-dictu-to-wand/`

## Notes

- Historical CHANGELOG entries (the ones that documented earlier
  changes against the `dictu` package, before this rename) were
  not rewritten. Per change-log convention they record what was
  true at the time; the current `### Changed (breaking)` entry
  for this rename is the forward-looking signal.
- ADR-0004 ("today: `dictu`") gained an addendum rather than an
  edit — same convention as ADR-0009.
- `cmd/wanderer/assess.go::selectedFrameworks` returns
  `models.Framework` — `FrameworkDICTU` was deleted (renamed to
  `FrameworkWand`); no remaining `FrameworkDICTU` reference exists.
- `internal/store/store_test.go` retains the literal string
  `framework='dictu'` and `dictu.juridisch.cert_issuer_eea` in
  the migration test's seed payload, since the test's job is to
  prove the rewrite SQL transforms exactly those legacy values.
