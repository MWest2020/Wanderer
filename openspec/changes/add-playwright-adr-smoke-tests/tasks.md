# Tasks: Playwright smoke tests (proposal pending sign-off)

> Every task is a design checkpoint until Mark approves the
> shape. The implementation PR opens after sign-off.

## 1. Open questions to resolve

- [ ] 1.1 Where do screenshots live durably? Recommendation:
  `docs/screenshots/` for curated commits, `tests/playwright/screenshots/`
  gitignored for scratch.
- [ ] 1.2 Run on every PR or nightly? Recommendation: every PR
  with binary cache; downgrade to a make-target if it ever
  exceeds 2 min.
- [ ] 1.3 Node lockfile policy? Recommendation: ship
  `package-lock.json`, install with `npm ci --ignore-scripts`
  per the global CLAUDE.md rule.

## 2. Sign-off checkpoints

- [ ] 2.1 Mark approves the scope (Chromium-only, no visual
  regression, screenshot-as-doc pipeline)
- [ ] 2.2 Mark approves the ADR-coverage doc-lint check
- [ ] 2.3 Mark approves the CI strategy

## 3. Pre-implementation inventory

- [ ] 3.1 Walk every existing ADR in `docs/decisions/` and
  identify which carry UI claims — those need a Playwright
  spec
- [ ] 3.2 List every archived OpenSpec change with UI scenarios
  (restructure-dar-layers, fix-nav-org-context, add-organisation-pivot,
  add-reporting-per-check, redesign-dashboard-pontificaal,
  add-assessment-ui, add-posture-dashboard) — these need
  Playwright spec files too

## 4. Open the implementation change

- [ ] 4.1 When 1.x + 2.x are resolved, scaffold
  `add-playwright-smoke-tests` (or rename of this change with
  a sibling code-carrying PR) and start coding
