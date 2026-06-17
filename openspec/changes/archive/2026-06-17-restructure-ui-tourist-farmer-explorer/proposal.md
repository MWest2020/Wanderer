# Restructure the UI around BI consumption personas (Tourist / Explorer / Farmer)

## Why

The UI grew a three-tab nav — **D**ashboard / **A**nalysis /
**R**eporting — where each tab answers a different question but the
labels don't say which, and the most valuable surface is buried:

- **Dashboard** shows aggregate posture (verdict pills, flow rollup)
  but never led with the one thing a returning operator looks for:
  *their targets and how each scored*. That list sat a click away on
  `/ui/targets`.
- **Analysis** is a rule×score steering matrix.
- **Reporting** is a rule catalogue.

Field feedback (the same operator who drove the "weinigzeggend" work):
*"on analysis I see Wanderer targets and last scans — that's what I
expect on my dashboard. The scan **is** the reporting. Reporting as a
catalogue is confusing."*

That maps cleanly onto the well-known BI consumption personas:

- **Data Tourist** — infrequent, wants a curated, glanceable answer.
  *"Show me the headline, don't make me build anything."*
- **Data Explorer** — ad-hoc, drills down, follows a thread, asks new
  questions.
- **Data Farmer** — returns to the same set regularly, cultivates and
  harvests over time (trends, drift, "which rule fails where").

Today's nav serves none of them well: the Tourist's fleet view is
hidden, the Explorer's report is labelled by an opaque scan ID, and
the Farmer's cross-fleet tools (matrix + catalogue) occupy the
top-billing slots as if they were the headline.

## What Changes

Re-cast the three surfaces onto the three personas:

1. **Overview = Tourist.** The dashboard leads with the **target
   fleet**: one row per target, its headline verdict, last scan, and a
   link to that scan's report — alongside the existing verdict pills
   and sovereignty-by-flow rollup. (The fleet table already shipped as
   the first step; this change formalises it in the spec.)

2. **Report = Explorer.** The per-scan assessment **is** the report.
   It is reached by clicking a target on the Overview, is identified by
   its **domain** (not the scan ID — already shipped), and leans into
   drill-down: the sovereignty overview + diagram, follow a flow to its
   ASN/country, down to the raw findings.

3. **Trends = Farmer.** The rule×score matrix (old Analysis) and the
   rule catalogue (old Reporting) **consolidate into one demoted
   "Trends" layer** — cross-fleet, "which rule fails on which targets",
   and the seam for over-time/drift later. It stays **thin** on
   purpose: this is a kijk-nu + drill-in tool, not a reporting suite.

4. **Nav collapses** from Dashboard/Analysis/Reporting to **Overview /
   Trends** (two tabs). The Report is not a nav tab — you arrive there
   by clicking a target.

Nothing about the probes, assessor, or store changes. This is purely
the web-ui information architecture.

## Impact

- **Affected capability:** `web-ui`.
- **Affected requirements (MODIFIED):** "Dashboard is 'is dit goed of
  niet'", "DAR nav persists the active organisation scope", "Analysis
  layer renders the steering matrix".
- **Affected requirements (REMOVED, folded into Trends):** "Reporting
  layer is the rule catalogue".
- **Affected requirements (ADDED):** "Overview leads with the target
  fleet", "The scan assessment is the report (Explorer drill-down)".
- **Routes:** `/ui/analysis` + `/ui/reporting` consolidate under
  `/ui/trends`; the old paths 301/302-redirect so deep links survive.
- **Docs:** an ADR (`docs/decisions/`) records the persona model with a
  `## UI surface` section, which requires a Playwright spec per the
  doc-lint contract.
- **No data migration.** Existing scans/assessments render unchanged.
