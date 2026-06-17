# Tasks

## 1. Overview (Tourist)
- [x] 1.1 Render the target fleet table on `/ui/` and `/ui/orgs/{slug}`
  (domain, verdict, last scan, report link). *(shipped: commit on main)*
- [x] 1.2 Confirm the fleet is the first block under `<main>`, above the
  verdict pills and flow rollup.
- [x] 1.3 Org-scoped dashboards show only that org's targets.

## 2. Report (Explorer)
- [x] 2.1 Assessment page is identified by the domain, not the scan ID.
  *(shipped)*
- [x] 2.2 Each fleet row links to `/ui/scans/{id}/assessment`.
- [x] 2.3 Verify the report's flows drill to ASN/country and to raw
  findings (no new work expected; assert the links exist).

## 3. Trends (Farmer)
- [x] 3.1 Add `/ui/trends` consolidating the rule catalogue (index) and
  the rule×score matrix (section / drill-in).
- [x] 3.2 Redirect `/ui/analysis` and `/ui/reporting` (and their
  `?org=` variants) to `/ui/trends`, preserving the org scope.
- [x] 3.3 Keep the per-rule deep-dive at `/ui/reporting/{fw}/{ruleID}`
  (or move under `/ui/trends/...`); update inbound links.

## 4. Nav
- [x] 4.1 Collapse the nav to **Overview** + **Trends**; drop the
  separate Analysis and Reporting tabs.
- [x] 4.2 Thread the active org scope across both tabs (Overview →
  `/ui/orgs/{slug}`, Trends → `/ui/trends?org={slug}`).

## 5. Docs & tests
- [x] 5.1 ADR in `docs/decisions/` with a `## UI surface` section
  (persona model + nav).
- [x] 5.2 Playwright spec for the ADR (doc-lint contract): fleet table
  on Overview, target → report click-through, Trends consolidation,
  two-tab nav.
- [x] 5.3 Update Go UI tests: fleet table, nav items, redirects.
- [x] 5.4 `openspec validate restructure-ui-tourist-farmer-explorer
  --strict` passes; archive on apply.
