## 1. Package + rule rename

- [ ] 1.1 Move `internal/assessor/dictu/` to `internal/assessor/wand/`; rename `package dictu` → `package wand` in every file
- [ ] 1.2 String-replace `dictu.` → `wand.` in every rule's `ID` field in `internal/assessor/wand/rules.go`
- [ ] 1.3 String-replace `dictu.` → `wand.` in every test file under `internal/assessor/wand/`
- [ ] 1.4 Update all importing packages: `cmd/wanderer/assess.go`, `cmd/wanderer/agent.go`, `internal/ui/registry.go`, the integration test harness in `internal/assessor/dictu/integration_test.go` (now `wand/integration_test.go`)

## 2. Schema migration

- [ ] 2.1 Add migration version 4 to `internal/store/migrations.go`: `UPDATE assessments SET framework = 'wand', dimensions = REPLACE(dimensions, '"dictu.', '"wand.') WHERE framework = 'dictu'`
- [ ] 2.2 Test in `internal/store/store_test.go`: seed an assessment row with `framework='dictu'` + `dictu.*` rationale-ID, run `Open` against the new code, assert the row was rewritten to `wand`
- [ ] 2.3 Idempotency test: a second `Open` does not re-rewrite (the migration runner skips the version on round 2)

## 3. CLI deprecated alias

- [ ] 3.1 `cmd/wanderer/assess.go` parses `--framework dictu` as `wand` plus a stderr deprecation line beginning with `warning:`
- [ ] 3.2 `--framework wand` (and `eucsf`, `both`) produce no deprecation noise
- [ ] 3.3 Test in `cmd/wanderer/assess_test.go` (or similar): the alias path produces the warning, the canonical path does not

## 4. UI registry alias

- [ ] 4.1 `internal/ui/registry.go::lookupRule` accepts both `wand` and `dictu` as keys for one release; both resolve against `wand.DefaultRules()`
- [ ] 4.2 Test in `internal/ui/registry_test.go`: a `dictu`-keyed lookup against a known `wand.*` rule still finds the rule

## 5. ADRs

- [ ] 5.1 Write `docs/decisions/0011-rename-dictu-to-wand.md` covering legal motivation, the wand naming rationale, the migration path, and the credit to DICTU's framework as inspiration
- [ ] 5.2 Append a one-paragraph addendum to `docs/decisions/0009-dual-framework-assessor.md` pointing at ADR-0011 and noting the new `wand` identifier

## 6. Documentation sweep

- [ ] 6.1 Update `docs/assessor.md` — replace `dictu` with `wand` in usage examples; add an "Inspired by" paragraph crediting DICTU's Toetsingsinstrument
- [ ] 6.2 Update `docs/architecture.md` — three-modi diagram + assessor section reference the `wand` package and rule prefix
- [ ] 6.3 Update `docs/operator.md` — `wanderer assess --framework wand|eucsf|both` examples; mention the deprecated alias once
- [ ] 6.4 Update `docs/tutorial.md` — first-run example uses `--framework wand`
- [ ] 6.5 Sweep README and any other doc for stray `dictu` mentions; replace with `wand` (preserving DICTU mentions in inspiration / heritage paragraphs)

## 7. Tests + CHANGELOG

- [ ] 7.1 Run the full suite — every `TestEveryRuleHasRationale`, integration test, UI test, store test must remain green
- [ ] 7.2 Add `TestFrameworkRename_RegressionGuard` somewhere central that asserts `wand` rules are registered, all rule IDs start with `wand.`, and no rule registered under `wand.DefaultRules()` carries a `dictu.*` ID — protects against accidental partial-rename in future edits
- [ ] 7.3 CHANGELOG entries: `### Changed (breaking)` for the rename of the persisted Framework field and the rule-ID prefix; `### Added` for the deprecated `dictu` alias on CLI + UI registry; `### Migration` (or note inside Changed) describing what the schema migration does

## 8. Validation

- [ ] 8.1 `openspec validate --specs --strict` passes
- [ ] 8.2 Manual smoke: build the new binary, open the demo DB from the previous session (`/tmp/w.db`), assert that the dashboard now shows `wand` framework labels and the existing rationales render against the renamed rules
- [ ] 8.3 Archive the change: `openspec/changes/archive/2026-04-30-rename-dictu-to-wand/`
