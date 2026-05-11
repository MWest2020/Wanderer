# Tasks: hermetic Playwright fixtures (design pending sign-off)

> Until Mark resolves the four open questions in proposal.md,
> every task here is a design checkpoint. The implementation
> PR opens after sign-off.

## 1. Resolve design decisions

- [ ] 1.1 Cmd binary vs internal/fixtures package. Recommendation:
  `cmd/wanderer-fixture` for discoverability.
- [ ] 1.2 One DB per scenario vs merged. Recommendation:
  one-per-scenario, three Playwright projects.
- [ ] 1.3 Schema-migration sync strategy. Recommendation: lean
  on store.Open's runner — no separate fixture migrations.
- [ ] 1.4 Keep agent-host.yaml side-by-side with the fixture
  seeder. Recommendation: yes; document the split.

## 2. Sign-off checkpoints

- [ ] 2.1 Mark approves the three scenarios (baseline,
  agent-host, empty-org).
- [ ] 2.2 Mark approves the binary location + naming
  (`cmd/wanderer-fixture` if 1.1 holds).
- [ ] 2.3 Mark approves the one-DB-per-scenario split if it
  changes how `make playwright` orchestrates Playwright
  projects.

## 3. Implementation — to scaffold after sign-off

- [ ] 3.1 `internal/fixtures/seed.go` with one exported
  `Build<Scenario>(*store.Store)` function per scenario.
  Pure Go — uses store.Insert* + assessor.DefaultRules.
- [ ] 3.2 `cmd/wanderer-fixture/main.go` — CLI accepts
  `--scenario <name>` and `--out <path>`, writes the SQLite,
  prints what it inserted.
- [ ] 3.3 Makefile `playwright-fixture` target that emits
  three DBs under `tests/playwright/fixtures/`.
- [ ] 3.4 `playwright.config.ts` gains three projects; each
  has its own webServer block on a different port.
- [ ] 3.5 Lift the three existing specs to assert on the
  deterministic fixture shape — drop demo-data assumptions.
- [ ] 3.6 Un-skip the "rules without rationale" placeholder
  assertion (covered by the `empty-org` scenario).
- [ ] 3.7 Un-skip the host-rule deep-dive smoke (covered by
  the `agent-host` scenario).
- [ ] 3.8 `.gitignore` excludes
  `tests/playwright/fixtures/*.db`.

## 4. Wrap-up

- [ ] 4.1 Update `docs/operator.md` with the seeded-DB
  workflow + how to add a scenario.
- [ ] 4.2 Refresh `add-host-side-scoring`'s archived
  references that point at the manual seed.
- [ ] 4.3 Commit + push (push needs explicit user OK).
- [ ] 4.4 Archive under
  `openspec/changes/archive/<YYYY-MM-DD>-add-playwright-fixtures/`.
