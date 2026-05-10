# Proposal: compact status column on the Reporting catalogue

## Intent

The `restructure-dar-layers` change (archived
2026-05-10) put the rule × score-counts matrix on Analysis and
left Reporting as a pure rule catalogue with no scoring data —
the spec literally said "MUST NOT carry scoring data". Mark
revised the model in the same session: *"denk dan alleen op
reporting ook de score per rule. Wel verwacht ik veel meer
rules, though."*

The reasoning is operational: with many more rules in the
future, the catalogue becomes a long list. Without any signal
of *which* rules are problem rules, an operator has to bounce
back to Analysis to triage. A compact per-rule status — worst
score reached + how many targets sit at that score — is a
triage hint that costs little and saves a lot of clicks.

## Scope

**In scope:**

- The Reporting catalogue at `/ui/reporting` gains a new
  "Current state" column showing the worst score reached
  across the snapshots in scope, plus an "X of Y targets" hint.
- The catalogue handler resolves the optional `?org=<slug>`
  filter so the status column is org-aware. The catalogue
  rows themselves (framework / dimension / rule ID /
  description / rationale) stay scope-independent.
- Rules with no recorded Rationale render a "no rationale yet"
  placeholder, not a fake score.
- Spec modification: the "Reporting layer is the rule catalogue"
  requirement is relaxed to allow a triage-only status column,
  with explicit language that the catalogue MUST NOT carry the
  full per-score matrix (that stays on Analysis).

**Out of scope:**

- A sortable / filterable catalogue. The catalogue rows are
  ordered by framework + dimension + ID; sorting by status is
  a future improvement when the rule count grows.
- A "show only afhankelijk rules" filter on the catalogue. Same
  reason — straightforward to add later when needed.
- Recomputing the matrix on Reporting. The full matrix lives
  on Analysis; the catalogue's status column is a hint, not a
  replacement.

## Wand / EUCSF dimensions informed

None directly. Operator UX.

## Passive / active boundary

UI rendering only.

## Parallel-safe

Touches `internal/ui/` plus the `web-ui` spec. No schema, no
new dependency, no probe surface change.
