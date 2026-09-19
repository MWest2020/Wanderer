# Run report — 08c anchor ids (task 8.6)

## Root cause
`assessor.Assess` (`internal/assessor/engine.go`) always emits one
`models.DimensionScore` per `WandDimensions` entry — including
`accountability` — for *every* framework it scores, even when that
framework's ruleset has zero rules for the dimension (it just comes
back `onbekend`/incomplete, not `NotApplicable`, since
`NotApplicable` requires `len(rules) > 0`). `assessmentHandler`
(`internal/ui/ui.go`) special-cases `models.DimensionAccountability`
for *any* framework, not just `wand`. Since the baseline fixture
scores every scan under both `wand` and `eucsf`
(`internal/fixtures/seed.go: addCompletedScan`), both framework cards
on `/ui/scans/{id}/assessment` render an accountability answer sheet,
and the template hardcoded `id="accountability"` on both — hence the
strict-mode "resolved to 2 elements" failure.

## Fix (anchors/links only, card content untouched)
- `internal/ui/templates/assessment.tmpl`: capture `{{$fw := .Framework}}`
  before entering `{{with .Accountability}}` and use it to build the
  id: `id="{{$fw}}-accountability"` → `wand-accountability` /
  `eucsf-accountability`.
- `internal/ui/ui.go`: the Overview accountability pill
  (`dashboardTargetRow.AccountabilityLink`) now points at
  `ReportURL + "#wand-accountability"` (was `"#accountability"`) —
  the pill's data always comes from the `wand` framework's dimensions,
  so it links at the wand card specifically.
- `tests/playwright/specs/accountability-answer-sheet.spec.ts`: both
  `page.locator("#accountability")` calls updated to
  `"#wand-accountability"`.

Did not touch dimension-card content, the generic (non-accountability)
`dimension-card` markup, or the six pre-existing rule-catalogue
strict-mode failures on `/ui/trends` mentioned as out of scope.

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — all packages `ok` (including `internal/ui`,
  `internal/fixtures`).
- Playwright: **could not run**. `tests/playwright/node_modules` does
  not exist in this cage and there is no browser egress to install
  Chromium — same constraint noted in run 08b. The spec's own
  top-of-file NOTE (still present, unchanged) already documents this;
  I left it as-is per the same convention used in 08b (only remove the
  NOTE once the spec has actually been run green). Ticking 8.6 off per
  the task-ref's instruction that you'll run it there.
- `openspec validate 2026-09-19-propose-accountability-dimension --strict`:
  **fails**, but on two pre-existing, unrelated requirements —
  `assessor/spec.md: "Rule results carry generic reason codes"` and
  `web-ui/spec.md: "Answer-sheet copy lives in one Dutch string table"`
  — both flagged as missing a literal SHALL/MUST even though they
  contain one ("Every code SHALL be registered…"), which looks like an
  openspec-CLI parsing quirk on those requirement bodies, not a
  regression from this run. `assessor/spec.md` hasn't been touched
  since commits `5e5fa41`/`dc0f38c`, well before this run, and I made
  no edits under `openspec/specs` or the change's `specs/` deltas.
  Not in task 8.6's scope (only ankers/links) — noted here, not fixed.

## Tasks.md
Ticked 8.6.
