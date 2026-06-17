# 0017 — UI information architecture: Tourist / Explorer / Farmer

**Status:** Accepted, 2026-06-17. Implements
`restructure-ui-tourist-farmer-explorer`.

**Context.** The UI grew a three-tab nav — Dashboard / Analysis /
Reporting — where each tab answered a different question but the
labels didn't say which, and the most valuable surface was buried. The
Dashboard showed aggregate posture (verdict pills, flow rollup) but
never led with the one thing a returning operator looks for: *their
targets and how each scored* — that list sat a click away on
`/ui/targets`. Analysis was a rule×score matrix; Reporting a rule
catalogue. Field feedback: *"on analysis I see Wanderer targets and
last scans — that's what I expect on my dashboard. The scan **is** the
reporting; a catalogue as the headline is confusing."*

That maps onto the BI consumption personas: the **Tourist** wants a
curated, glanceable answer; the **Explorer** drills into one thing; the
**Farmer** returns to the same cross-fleet set over time. The old nav
served none of them well.

**Decision.** Re-cast the three surfaces onto the personas:

- **Overview = Tourist** (`/ui/`, `/ui/orgs/{slug}`): leads with the
  target fleet (one row per target, headline verdict, last scan, link
  to its report), then the verdict pills and sovereignty-by-flow
  rollup.
- **Report = Explorer** (`/ui/scans/{id}/assessment`): the per-scan
  assessment *is* the report. Identified by the target domain (not the
  scan ID), reached by clicking a target on the Overview. Not a nav
  tab.
- **Trends = Farmer** (`/ui/trends`): the rule catalogue (index) and
  the rule×score matrix consolidate onto one demoted tab. Kept thin —
  over-time trend lines are out of scope and graduate into their own
  proposal if demand appears.

The nav collapses to two tabs: **Overview / Trends**. `/ui/analysis`
and `/ui/reporting` 302-redirect to `/ui/trends`, preserving the org
scope; the per-rule deep-dive stays at `/ui/reporting/{fw}/{ruleID}`.

**Consequences.** No probe/assessor/store change; pure web-ui IA. The
Farmer layer stays a seam, matching how the tool is actually used
(look-now + drill-in). Deep links to the retired routes survive via
the redirects.

## UI surface

- **Overview** renders a `table.targets` fleet block as the first
  section under `<main>`: each row carries the domain, a `score-*`
  verdict badge (worst across the wand pack, "not assessed" when
  none), the last scan time/status, and a `report →` link to
  `/ui/scans/{id}/assessment`. The verdict pills and flow rollup follow.
- **Report** (`/ui/scans/{id}/assessment`) is titled by the domain;
  the scan ID is a secondary reference.
- **Trends** (`/ui/trends`) renders the rule catalogue (with the
  "Current state" status column) and the per-rule score matrix in two
  sections; rule cells link to the per-rule deep-dive.
- **Nav** shows exactly two tabs, **Overview** and **Trends**; the
  retired Analysis and Reporting tabs are gone, and their routes
  redirect to `/ui/trends` (scope-preserving).

The Playwright spec `tests/playwright/specs/ui-personas.spec.ts`
asserts the fleet table on the Overview, the two-tab nav, the Trends
consolidation, and the legacy-route redirects.
