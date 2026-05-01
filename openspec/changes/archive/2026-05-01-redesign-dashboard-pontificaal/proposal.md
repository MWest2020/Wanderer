# Proposal: pontificaal dashboard headline + intern/extern split

## Intent

The current `/ui/` renders three sections — Posture summary, Top
concerns, Recent activity — that are useful but read more like an
*analysis* surface than a *dashboard*. An operator landing on
`/ui/` for the first time has no immediate answer to: "what is
this, what data does it cover, when was it last updated, and is
there anything I should know?"

Mark named the gap directly: "DASHBOARD moet pontificaal zeggen
hoe en wat" — the dashboard should *grandly* state what it is
showing. And: scans currently land "from outside" (perimeter),
but the agent already supports "from inside out" (inventory +
egress); the dashboard does not surface the asymmetry, so an
operator cannot see at a glance whether host-side coverage exists
or is missing.

This change adds a headline section and an explicit perimeter /
agent split to the dashboard, without schema changes.

## Scope

**In scope:**

- A new **headline** section at the top of `/ui/`: a short
  pontificaal statement of what Wanderer is and what data the
  current instance has. Shows: total scans, perimeter targets
  count, agent-host targets count, latest scan timestamp,
  frameworks scored.
- An explicit **External vs Internal** framing of the existing
  Posture summary. External = perimeter scans (Targets with
  `kind=domain`); Internal = agent-host scans (Targets with
  `kind=host`). When one side has no data, the dashboard says so
  with a one-line empty state — never silently absent.
- A **navigation strip** linking to the three layers: Dashboard
  (current page), Analysis (per-scan assessment), Reporting (the
  per-check cross-target page added in the sibling proposal
  `add-reporting-per-check`). When Reporting is not yet wired,
  the link is omitted; this proposal does not require that
  proposal to land first.
- A small CSS pass for the hero / nav. No JS introduced; the UI
  stays server-rendered HTML.

**Out of scope:**

- Organisation as a first-class concept (separate proposal:
  `propose-organisation-pivot` — schema work, design only first).
  Until that lands, the headline reads "Wanderer instance
  overview" rather than naming an organisation.
- A new Reporting page (separate proposal:
  `add-reporting-per-check`).
- Authentication / multi-tenant separation. The UI remains the
  single-tenant `--ui-htpasswd`-protected surface it is today.
- Any new probe, scan, or assessor logic. The headline reads
  exclusively from data the store already holds.

## Why now

The dashboard has been adding sections incrementally as features
landed (posture, top concerns, activity, framework breakdown).
Without a headline, the page is a *list of widgets* rather than a
narrative — and a sovereignty monitor's value is the narrative,
not the widgets. The cheapest moment to fix this is now, before
the Reporting page lands and adds yet another section to a
header-less surface.

## Wand / EUCSF dimensions informed

None directly — this is operator-facing presentation. Both rule
packs continue to score the same Findings; the dashboard just
frames them more clearly.

## Passive / active boundary

UI rendering only. No new outbound calls; no new data ingested.

## Parallel-safe

Touches `internal/ui/` only: aggregator extension, dashboard
template, one CSS file, tests. No schema, no API, no public-
package surface change.
