# Proposal: per-check Reporting page

## Intent

The dashboard answers "how is my instance doing overall?". The
per-scan assessment page answers "how is *this* target doing on
every rule?". Neither answers the third natural question:

> "How is *this rule* doing across every target?"

That is the **per-check Reporting** view. An operator wanting to
sweep their portfolio for "everyone whose certificate issuer is
US-based" or "every host whose hyperscaler dependency is set" has
no surface today. They have to open every assessment page and
scan visually for the rule.

This change adds:

1. `/ui/reporting` — a single page listing every rule that has
   produced at least one Rationale across the persisted
   Assessments, with per-rule columns for `soeverein` /
   `voldoende` / `afhankelijk` / `onbekend` target counts.
2. `/ui/reporting/{framework}/{ruleID}` — the per-rule deep dive:
   the rule's name, dimension, rationale paragraph, plus a table
   of every target the rule fired on with the verdict and a
   link back to the underlying scan's assessment page.

The Reporting nav tab introduced in
`redesign-dashboard-pontificaal` lights up when this proposal
lands.

## Scope

**In scope:**

- New routes `/ui/reporting` and
  `/ui/reporting/{framework}/{ruleID}` mounted by
  `internal/ui/Handler`.
- New aggregator helpers `RuleSummary` and `RuleTargetRows` in
  `internal/ui/aggregate.go`. Both read the same
  `[]TargetSnapshot` the dashboard already builds.
- Flip the dashboard's `HasReporting` view field to true so
  `nav.tmpl` renders the Reporting tab.
- Two new templates: `templates/reporting.tmpl` (index list) and
  `templates/reporting_rule.tmpl` (per-rule detail).
- Tests: aggregator unit tests + render assertions.

**Out of scope:**

- Filtering by dimension / framework on the index page. A future
  proposal can add query params; the v1 page lists all rules in
  a stable order.
- Export-to-CSV from the page. Operators who need that already
  have `wanderer export` for the same data; duplicating it in the
  UI is scope creep.
- Time-series view ("how did this rule's pass rate trend?"). The
  store has multiple Assessments per scan-target line; trend
  rendering is its own feature.

## Why now

The dashboard redesign just made room for the Reporting tab, and
the data is already in the store — every persisted Assessment
carries a `[]Rationale` with a `CriteriumID` and `Score`. Building
the view is a pure read-side aggregator + two templates; no
schema, no probe, no rule pack change.

## Wand / EUCSF dimensions informed

None directly. The Reporting page is operator ergonomics on top
of existing Assessments.

## Passive / active boundary

UI-only. No new outbound calls; no new data ingested.

## Parallel-safe

Touches `internal/ui/` plus two new templates. Reuses the
`lookupRule` registry that the existing assessment page already
uses to attach human-readable Description and Rationale. No
schema changes.
