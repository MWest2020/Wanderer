# Proposal: Playwright smoke tests driven by ADR claims

> **Status:** Design proposal. The implementation lands after
> Mark signs off on the shape (browser binary footprint,
> screenshot-as-doc pipeline, CI strategy).

## Intent

Mark flagged on 2026-05-10: *"misschien ook tijd voor playwright
en dat je zelf kijkt (aan de hand van ADR's?)"*. The current
test surface uses `httptest` + `httptest.NewServer` + string
matching on HTML bodies, which catches structural bugs but
cannot tell whether something *looks* right or whether the
operator's navigation across the DAR layers actually flows.

Add a Playwright-driven smoke layer that:

1. Renders each UI page in a real browser (Chromium headless).
2. Asserts the *claims* from the relevant ADR or OpenSpec
   scenario — every ADR that touches the UI surface earns a
   matching Playwright spec that exercises the scenarios it
   describes.
3. Captures screenshots into `docs/screenshots/` so the
   operator documentation can ship visual reference material
   without manual capture.

## Scope

**In scope:**

- New top-level `tests/playwright/` directory with
  `playwright.config.ts`, a small `package.json`, and one
  spec file per UI-touching ADR or OpenSpec change.
- A `make playwright` target (and matching CI job) that:
  - starts a `wanderer serve` against a seeded `:memory:`
    SQLite (or a temp file)
  - runs the Playwright spec set
  - tears the server down
- Screenshots written to `docs/screenshots/<spec-name>/*.png`
  on every run. The path is gitignored by default; opting in
  to commit a screenshot is a deliberate `git add -f`. The
  test logs which screenshots were written so an operator
  refresh of the docs is a one-command flow.
- An "ADR coverage" doc note: every ADR that describes UI
  behaviour SHALL link to the Playwright spec that exercises
  its claims.

**Out of scope:**

- Visual regression diffs (pixel comparisons). The screenshot
  capture is for documentation; visual-regression tooling is
  its own beast and can land later if needed.
- Cross-browser matrix. Chromium headless is the contract.
- Mobile / responsive viewport testing. Wanderer's UI is
  desktop-first read-only.
- Performance / Lighthouse runs. Separate concern.
- Replacing the existing `httptest` go tests. Those stay as
  the unit/integration layer; Playwright is the end-to-end +
  doc-screenshot layer on top.

## ADR-driven coverage

Each ADR that touches the UI gets a corresponding Playwright
spec file. The mapping is enforced by a doc-lint check (a small
Go test that grep-scans `docs/decisions/` for `## UI
surface` sections and confirms a matching spec file exists).

Initial mapping for current ADRs:

| ADR                                   | Spec file                       | Scenarios                          |
|---------------------------------------|---------------------------------|------------------------------------|
| (none yet — see Open questions)       | `tests/playwright/specs/dar.spec.ts` | DAR nav, org scope persistence    |
| 0009 dual-framework assessor          | `tests/playwright/specs/dual-framework.spec.ts` | per-framework verdict pills on Dashboard |

(The current ADRs are mostly back-end architecture decisions
without UI claims. The mapping will fill out as new UI-shaping
ADRs land. The `restructure-dar-layers` and
`fix-nav-org-context` archived OpenSpec changes provide the
scenarios for the initial spec set.)

## Browser binary footprint

Playwright bundles browser binaries which inflate the dev
checkout by ~200–400 MB. Mitigations:

- The Playwright install runs only when `tests/playwright/` is
  active — a top-level `Makefile` target gates it, and
  `npm install` happens inside that directory only.
- CI caches the browser binary by Playwright version.
- The repo's `.gitignore` excludes
  `tests/playwright/node_modules` and the screenshot output.

## Open questions

1. **Where do screenshots live durably?** Gitignored by default
   (above), but at *some* point Mark may want a "this is what
   /ui/ looks like in 2026-05" doc artefact. Recommend: a
   `docs/screenshots/` directory with one curated PNG per major
   release, manually committed; the spec output stays a
   regenerable scratch dir.

2. **Run on every PR or nightly?** Playwright runs are slower
   than the Go unit tests (~30s for the current page set).
   Recommend: on every PR, with the browser binary cache to
   keep cold start fast. If it ever gets slow enough to matter
   (>2 min), gate to a `playwright` make-target invoked only
   on UI-touching PRs.

3. **Node tooling lockfile policy?** Per the global
   CLAUDE.md, `npm ci --ignore-scripts` is the install
   contract. The `package-lock.json` ships in the repo;
   `tests/playwright/package.json` pins Playwright at a
   specific version.

## Wand / EUCSF dimensions informed

None directly. Test infrastructure.

## Passive / active boundary

Test-only. Tests run against a local `wanderer serve` instance
seeded in `:memory:`; no production data touched.

## Parallel-safe (when implemented)

New `tests/playwright/` directory, one `Makefile` target,
adjustments to `.gitignore`, optional CI job. No changes to
`internal/ui/` or any production code path.
