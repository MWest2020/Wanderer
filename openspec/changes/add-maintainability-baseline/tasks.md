# Tasks: Maintainability Baseline

## 1. Changelog

- [ ] 1.1 Create `CHANGELOG.md` at project root with Keep-a-Changelog header, `## [Unreleased]` section, and a seed entry noting `init-mvp-scanners` landed
- [ ] 1.2 Add entry for this change itself under `Added`

## 2. Decision records

- [ ] 2.1 Create `docs/decisions/` and `docs/decisions/README.md` index
- [ ] 2.2 `docs/decisions/0000-template.md` — blank MADR template
- [ ] 2.3 `docs/decisions/0001-openspec-workflow.md`
- [ ] 2.4 `docs/decisions/0002-api-stability-classes.md`
- [ ] 2.5 `docs/decisions/0003-dependency-policy.md`

## 3. Contributor guidance

- [ ] 3.1 `docs/maintainability.md` — single entry point: CHANGELOG discipline, ADR discipline, testing baseline, doc baseline, OpenSpec discipline, dependency policy, commit style
- [ ] 3.2 `CODEOWNERS` with default owner (`* @MWest2020`)
- [ ] 3.3 Link `docs/maintainability.md` from `docs/README.md`

## 4. Verification

- [ ] 4.1 Confirm `openspec validate add-maintainability-baseline` passes (or `openspec status` if no validate command)
- [ ] 4.2 Confirm CHANGELOG + ADRs + maintainability.md render cleanly on GitHub markdown
