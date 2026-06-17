# Design — UI personas restructure

Locking the non-obvious decisions (per the "boring & auditable" rule).

## The persona → surface mapping

| Persona | Question | Surface | Route |
|---------|----------|---------|-------|
| **Tourist** | "Are my targets OK?" | Overview (fleet + verdicts + flows) | `/ui/`, `/ui/orgs/{slug}` |
| **Explorer** | "What's going on with *this* one?" | Report (assessment drill-down) | `/ui/scans/{id}/assessment` |
| **Farmer** | "Which rule fails across my fleet, over time?" | Trends (matrix + catalogue) | `/ui/trends` |

## Decisions

### D1 — Merge Analysis + Reporting into one "Trends" tab, don't keep both
The steering matrix (Analysis) and the rule catalogue (Reporting) are
two views of the same Farmer question: rules across the fleet. Two
top-level tabs for one persona is exactly the clutter that hid the
Tourist's fleet. They become one page (`/ui/trends`): the catalogue is
the index, the per-score matrix is a section or a drill-in. Keeping the
old routes as redirects costs nothing and preserves deep links.

### D2 — The Report is not a nav tab
You don't navigate to "a report" in the abstract; you navigate to *a
target*. So the Report is reached by clicking a fleet row on the
Overview, never from the top nav. This is what makes the nav collapse
to two tabs without losing anything.

### D3 — Keep the Farmer layer thin
Per the operator: this is primarily a **Tourist + Explorer** tool —
look now, drill in. The Farmer layer (trends over time, drift history,
scheduled-scan dashboards) is a seam, not a build-out. `/ui/trends`
ships as the consolidated matrix+catalogue we already have; over-time
trend lines are explicitly out of scope for this change and graduate
into their own proposal if demand appears.

### D4 — Overview verdict per target = worst score across the wand pack
The fleet row shows one verdict, computed with the existing
`WorstScore` over the preferred (wand) assessment's dimensions, falling
back to whatever framework was assessed. Worst-case is the honest
headline for a Tourist: one red flow makes the target "afhankelijk".
The nuance lives one click deeper in the Report.

### D5 — Don't rename the dashboard route
The Overview stays at `/ui/` (and `/ui/orgs/{slug}`). "Overview" is the
nav label and the mental model; the route is unchanged so bookmarks and
the org-scope threading keep working.

## Out of scope
- Over-time trend charts / drift history UI (Farmer build-out — D3).
- Any probe, assessor, or store change.
- Auth/permission changes.
