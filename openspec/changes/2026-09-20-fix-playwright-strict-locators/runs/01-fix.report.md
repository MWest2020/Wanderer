# Run report — 01 fix (tasks 1.1–1.3)

## Root cause
`/ui/reporting` 302-redirects to `/ui/trends` (`redirectToTrends` in
`internal/ui/ui.go`), and `trends.tmpl` renders two tables that both
show every rule's `CriteriumID` and `Description`: `table.rule-catalogue`
(the rule catalogue section) and `table.reporting-rules` (the score
matrix section). A bare `page.locator("text=<rule-id>")` therefore
resolves to two elements on any page that ends up on `/ui/trends`,
including the four specs that call `page.goto("/ui/reporting")`.

On `/ui/`, `dashboard.tmpl` has two `<h2>` headings whose accessible
name matches `/Verdict/i`: `<h2>Verdict <span>is this OK?</span></h2>`
(the intended one) and `<h2>Targets <span>your fleet, last scan &amp;
verdict</span></h2>` — the latter's scope-label text ends in the
lowercase word "verdict", which the case-insensitive unanchored regex
also matches.

## Fix (specs only, no UI/template changes — task 1.3)
- `reporting-catalogue.spec.ts`, `container-image-sovereignty.spec.ts`,
  `eu-package-origin.spec.ts`, `host-side-scoring.spec.ts`,
  `nextcloud-as-target.spec.ts`: each rule-ID (and, in
  `reporting-catalogue.spec.ts`, the rule description) locator is now
  chained off `const catalogue = page.locator("table.rule-catalogue")`
  instead of a bare `page.locator("text=...")` on the whole page.
- `dar.spec.ts`: `getByRole("heading", { name: /Verdict/i })` →
  `getByRole("heading", { name: /^Verdict\b/i })`, anchored so it can't
  also match the "Targets ... verdict" heading.

Did not touch `internal/ui/templates/*`, `internal/ui/ui.go`, or
`tests/playwright/playwright.config.ts` (all six specs were already
present in a `testMatch` array — nothing to add there).

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — all packages `ok`.
- `openspec validate 2026-09-20-fix-playwright-strict-locators --strict`
  — `Change '2026-09-20-fix-playwright-strict-locators' is valid`.
- Playwright: **not run**. Per this task-ref, `npx playwright install`
  needs ~112MB from the Playwright CDN and the cage has no egress to
  it. Ticking 1.1–1.3 off per the task-ref's instruction; task 2.1
  (`make playwright` 37/37) is left open for Mark's session to measure.

## Tasks.md
Ticked 1.1, 1.2, 1.3. Left 2.1 and 2.2 open (2.2 I ran manually above
and it's green, but I'm leaving the checkbox for Mark's session per
the task-ref, since 2.1 in the same section is also gated on that
session and I don't want to partially tick section 2).
