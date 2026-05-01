# Design: per-check Reporting page

## Index page (`/ui/reporting`)

```
┌────────────────────────────────────────────────────────┐
│ NAV: Dashboard · Analysis · [Reporting]                │
├────────────────────────────────────────────────────────┤
│ Reporting · per-rule cross-target view                 │
│ Generated 2026-05-01T13:43Z                            │
├────────────────────────────────────────────────────────┤
│ Framework | Rule                       | sov | vol | afh | onb │
│-----------|-----------------------------|-----|-----|-----|-----│
│ wand      | wand.juridisch.cert_issuer  |  1  |  0  |  1  |  0  │
│ wand      | wand.data_ai.mx_jurisdiction|  1  |  0  |  1  |  0  │
│ eucsf     | eucsf.sov2.apex_jurisdiction|  1  |  0  |  1  |  0  │
│ eucsf     | eucsf.sov6.no_us_hyperscaler|  1  |  0  |  1  |  0  │
│ ...                                                            │
└────────────────────────────────────────────────────────────────┘
```

Each row is one rule that has produced at least one Rationale
across all persisted Assessments. Counts are distinct target
IDs, not occurrence counts (a rule firing twice on one target
counts once — same convention as Top concerns).

The rule cell is a link to the per-rule detail page. The
description text comes from the existing `lookupRule(framework,
criteriumID)` registry — same source the assessment page uses.

Stable row order: framework first (`wand` > `eucsf` >
alphabetical), then rule ID alphabetical within each framework.

## Detail page (`/ui/reporting/{framework}/{ruleID}`)

```
┌────────────────────────────────────────────────────────┐
│ NAV: Dashboard · Analysis · [Reporting]                │
├────────────────────────────────────────────────────────┤
│ wand.juridisch.cert_issuer_eea                         │
│ Dimension: juridisch                                   │
│                                                        │
│ Description: Apex certificate issuer must be in EEA    │
│                                                        │
│ Rationale: …prose explaining why this rule matters…    │
├────────────────────────────────────────────────────────┤
│ Target          | Score        | When        | Verdict │
│-----------------|--------------|-------------|---------│
│ conduction.nl   | afhankelijk  | 2026-05-01  | issued  │
│                 |              |             | in US   │
│ rijksoverheid.nl| soeverein    | 2026-05-01  | issued  │
│                 |              |             | in IE   │
└────────────────────────────────────────────────────────┘
```

Each row links the target column out to the originating scan's
assessment page (`/ui/scans/{scanID}/assessment`).

## Data shape

```go
// RuleSummaryRow is one row on /ui/reporting.
type RuleSummaryRow struct {
    Framework   string
    CriteriumID string
    Description string  // from lookupRule, "" if not registered
    Counts      map[models.Score]int  // distinct target counts per score
}

// RuleTargetRow is one row on /ui/reporting/{fw}/{rule}.
type RuleTargetRow struct {
    TargetID  string
    Domain    string
    ScanID    string  // for the assessment-page back-link
    Score     models.Score
    Verdict   string  // the Rationale.Verdict text
    When      time.Time
}

func RuleSummary(snaps []TargetSnapshot) []RuleSummaryRow
func RuleTargetRows(snaps []TargetSnapshot, framework, ruleID string) []RuleTargetRow
```

`RuleSummary` walks each snapshot, each Assessment, each
Dimension, each Rationale. It groups by `(framework, criteriumID,
score)` into a set of target IDs (sets, not bags) — the distinct
count-by-target convention from the dashboard's Top concerns.

`RuleTargetRows` filters to one (framework, ruleID), returning
one row per target where the rule fired (most recent Assessment
per target wins, matching the dashboard).

## Routes

```go
r.Get("/reporting", reportingIndexHandler(st, tmpl))
r.Get("/reporting/{framework}/{ruleID}", reportingRuleHandler(st, tmpl))
```

The index handler builds the same per-target snapshot the
dashboard does, then calls `RuleSummary`. The detail handler
resolves the rule via `lookupRule` (returns 404 if neither
framework knows the rule) and runs `RuleTargetRows`.

## Wiring the nav tab

The `dashboardView` field `HasReporting` set to `true` in this
proposal flips the nav-bar tab on. The dashboard handler is the
only place we compute the nav data; once the route is registered
in `Handler`, the field flips on for every page that uses
`nav.tmpl` (including this new pair). The other handlers — scan,
assessment, drift, targets — already pass through the partial,
so they pick up the Reporting tab on next render.

## Test strategy

- `TestRuleSummary_DistinctCounts` — same rule firing twice on
  one target counts once; counts split by score correctly.
- `TestRuleSummary_StableOrder` — wand rules before eucsf, then
  alphabetical within framework.
- `TestRuleTargetRows_Filtered` — only the requested rule shows
  up; targets without that rule are absent.
- `TestRuleTargetRows_NewestAssessmentWins` — when a target has
  multiple Assessments for the same framework, the newest one
  drives the row.
- Render test: `/ui/reporting` returns 200 and lists rules;
  `/ui/reporting/{fw}/{ruleID}` returns 200 with rule details
  and target rows; unknown rule → 404.
