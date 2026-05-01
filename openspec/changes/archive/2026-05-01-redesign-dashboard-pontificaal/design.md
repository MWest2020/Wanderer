# Design: pontificaal dashboard headline + intern/extern split

## Layout (top to bottom)

```
┌────────────────────────────────────────────────────────┐
│ NAV: [Dashboard] · [Analysis] · [Reporting]            │
├────────────────────────────────────────────────────────┤
│                                                        │
│   Wanderer · sovereignty observation                   │
│   Last scan: 2026-05-01 14:43 UTC · 5 scans            │
│   ────────────────────────────────────────────         │
│                                                        │
│   External coverage: 2 perimeter targets               │
│   Internal coverage: no agent hosts reporting          │
│   Frameworks scored: wand, eucsf                       │
│                                                        │
├────────────────────────────────────────────────────────┤
│ EXTERNAL POSTURE (perimeter)                           │
│   eucsf:  1 soeverein · 1 afhankelijk                  │
│   wand:   2 afhankelijk                                │
├────────────────────────────────────────────────────────┤
│ INTERNAL POSTURE (agent host inspectors)               │
│   No agent hosts reporting yet. Run                    │
│   `wanderer agent --config <yaml>` to onboard one.     │
├────────────────────────────────────────────────────────┤
│ TOP CONCERNS                                           │
│   ...table...                                          │
├────────────────────────────────────────────────────────┤
│ RECENT ACTIVITY                                        │
│   ...table...                                          │
└────────────────────────────────────────────────────────┘
```

The headline (first block) is the pontificaal change: an operator
landing fresh sees *what this is*, *what coverage exists*, and
*when it was last refreshed* without scrolling.

## Data shape

Extend `dashboardView` with one new struct, populated in
`dashboardHandler` from data the store already holds:

```go
type headlineView struct {
    LastScanAt        string  // RFC3339 of most recent scan, "" if none
    TotalScans        int
    PerimeterTargets  int     // unique TargetIDs with Kind=domain
    AgentHostTargets  int     // unique TargetIDs with Kind=host
    Frameworks        []string // sorted, e.g., ["wand", "eucsf"]
}
```

Populating PerimeterTargets vs AgentHostTargets requires one
`store.GetTarget` per unique TargetID. With current target counts
(< 100) this is acceptable; if it ever becomes a hotspot, a
`store.ListTargets` is the obvious upgrade and a sibling change.

The existing `PostureBlocks` becomes per-scope: split into
`ExternalPostureBlocks` (counts limited to scans whose Target is
`Kind=domain`) and `InternalPostureBlocks` (Kind=host). The
existing `summary` aggregator is reused — we just feed it a
filtered snapshot list per scope.

## Template

`dashboard.tmpl` grows three new sections; existing sections
(Top concerns, Recent activity) move below. The empty-state copy
already pinned in unit tests stays — when both External and
Internal posture are empty, the page falls back to today's
`empty-state` paragraph.

A small `nav.tmpl` partial is introduced and included from
dashboard / scan / assessment / drift / index templates so
navigation is consistent across the four layers. The nav has
three entries:

- **Dashboard** → `/ui/`
- **Analysis** → `/ui/targets` (the most common entry into
  per-scan analysis is via the targets list)
- **Reporting** → `/ui/reporting` (omitted from the rendered
  HTML when the route is not registered, so this proposal does
  not depend on `add-reporting-per-check` landing first)

## Empty states are explicit

The "no internal coverage" copy is the most important new
content. An operator running only perimeter scans must see:

> Internal coverage: no agent hosts reporting yet. Run
> `wanderer agent --config <yaml>` to onboard one.

…rather than a missing section. The asymmetry between extern and
intern is the user's named pain point; surfacing it as a
deliberate "we can also do internal, but you haven't wired it"
message is the boring/auditable solution.

## CSS

One small block added to `internal/ui/static/main.css`: the
hero `<header>` gets a `.headline` class with bigger title
typography and a list-of-stats layout. Posture sections grow a
`.scope-label` line. No new colour palette, no new components
beyond what existing badges already provide.

## Test strategy

- `aggregate_test.go` gains a `TestHeadline_Coverage`: feeds in
  a mix of domain-kind and host-kind targets and asserts the
  counts come out right.
- `aggregate_test.go` gains a `TestPostureBy_ExternalInternal`:
  feeds in mixed targets and asserts the split posture maps are
  populated correctly per kind.
- `ui_test.go` gains a render test asserting:
  - Headline contains "External coverage:" and "Internal
    coverage:" labels
  - When no host targets exist, the empty-state copy mentions
    `wanderer agent`
  - Nav links to /ui/, /ui/targets are present (Reporting may or
    may not be present depending on route registration)

No new integration test — the existing UI integration tests
already cover end-to-end render with stub data.

## Migration / rollout

The change is additive. Any existing /ui/ bookmark continues to
work. Operators see the new headline on next page load; nothing
to deploy beyond the new binary.
