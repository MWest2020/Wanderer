# Proposal: restructure DAR layers — Dashboard / Analysis / Reporting

## Intent

The earlier DAR shape mounted the rule × score-counts matrix on
**Reporting** and used Dashboard for posture summaries + Top
concerns + Recent activity. Mark refined the model on 2026-05-10:

> *"Dus dan bij dashboarding enkel goed/niet goed, analysis zie
> je de matrix en reporting de regels."*

So:

- **Dashboard** = "is dit goed of niet" — high-level health
- **Analysis** = the steering matrix (rule × score-counts), the
  per-target slice, the per-scan and per-assessment deep-dives
- **Reporting** = the rule catalogue + per-rule deep dive — what
  is being measured and why

This change moves the existing matrix view from
`/ui/reporting` to `/ui/analysis`, replaces `/ui/reporting`
with a rule catalogue, and trims Dashboard down to the
verdict-level widgets that answer "do I need to act?".

## Scope

**In scope:**

- New route `/ui/analysis` that renders the rule × score-counts
  matrix that today lives at `/ui/reporting`. The view supports
  the existing `?org=<slug>` filter unchanged.
- `/ui/targets` becomes a sub-link from `/ui/analysis` (the
  per-target slice of the same data). Both routes keep working
  — they are sibling Analysis views.
- Per-scan (`/ui/scans/{id}`) and per-assessment
  (`/ui/scans/{id}/assessment`) pages stay where they are; they
  are already Analysis-tier deep dives. The nav `Active`
  attribute stays "analysis".
- New `/ui/reporting` page renders a rule catalogue: every
  registered rule's framework, ID, dimension, description, and
  rationale, with a link to the per-rule detail page. No
  scoring data on this page — it is reference, not steering.
- `/ui/reporting/{framework}/{ruleID}` keeps its current shape
  (the per-rule deep dive with target rows), but its breadcrumb
  link points back to `/ui/reporting` (the catalogue) rather
  than `/ui/analysis`.
- Dashboard (`/ui/`, `/ui/orgs/{slug}`) replaces its current
  posture-summary blocks + Top concerns + Recent activity
  sections with:
  - The headline-stats strip (kept)
  - One verdict-pill block per framework: the worst score that
    any target reached, plus a one-line "X of N targets at this
    score" summary
  - The Organisations list (instance-wide, kept)
- Nav links stay scope-aware — Analysis and Reporting carry
  `?org=<slug>` through, same precedence pattern as before.
- Spec delta updates the Reporting requirements to describe
  the catalogue shape and adds Analysis requirements for the
  matrix.

**Out of scope:**

- Renaming the JSON / MCP shapes. The MCP `org_*` methods and
  the API endpoints stay unchanged.
- Adding a new aggregator. The matrix view reuses the existing
  `RuleSummary` helper; the rule catalogue reuses
  `lookupRule` plus a new `ListAllRules` helper that walks the
  registered packs.
- A redesigned drill-from-Dashboard flow with rich graphs.
  Dashboard stays deliberately thin.

## Why now

The reshuffle is a one-shot edit at the point where every
relevant page already exists and the data shape is stable. The
alternative — let the layers drift and try to migrate later
when more code references the routes — costs more.

## Wand / EUCSF dimensions informed

None directly. Operator-facing UX restructure.

## Passive / active boundary

UI rendering only. No new outbound calls; no new probes; no
schema or API change.

## Parallel-safe

Touches `internal/ui/` (handlers, templates, CSS) plus an
`internal/assessor/registry.go` helper that already exists in
spirit (`lookupRule` uses it). No new package; no new
dependency.
