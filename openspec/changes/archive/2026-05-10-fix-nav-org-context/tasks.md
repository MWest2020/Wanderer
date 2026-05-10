# Tasks: persist org scope across DAR nav tabs

## 1. Nav partial

- [x] 1.1 `nav.tmpl` accepts `OrgSlug` (string) alongside `Active`
  and `HasReporting`
- [x] 1.2 Dashboard link → `/ui/orgs/<slug>` when slug set,
  else `/ui/`
- [x] 1.3 Analysis link → `/ui/targets?org=<slug>` when slug set
- [x] 1.4 Reporting link → `/ui/reporting?org=<slug>` when slug set

## 2. Handler wiring

- [x] 2.1 `dashboardHandler` (instance-wide) passes
  `OrgSlug=""`, `HasReporting=true`
- [x] 2.2 `dashboardOrgHandler` passes `OrgSlug=<slug>`,
  `HasReporting=true`
- [x] 2.3 `targetsHandler` reads `?org=`, filters, passes scope
  through to template
- [x] 2.4 `scanHandler`, `assessmentHandler`, `driftHandler` set
  `HasReporting=true` (and pass through any `?org=` from query
  for nav consistency)
- [x] 2.5 `reportingIndexHandler` and `reportingRuleHandler`
  populate a `ScopedOrganisation` on their views and pass
  `OrgSlug` to nav.tmpl

## 3. Templates

- [x] 3.1 `reporting.tmpl` renders scope label in the header
  ("Scope: <Name>") when active
- [x] 3.2 `reporting_rule.tmpl` ditto
- [x] 3.3 `index.tmpl` (targets) ditto
- [x] 3.4 Every page-template invocation of nav.tmpl threads the
  parent's OrgSlug

## 4. Tests

- [x] 4.1 `/ui/orgs/acme` HTML contains `href="/ui/targets?org=acme"`
- [x] 4.2 `/ui/orgs/acme` HTML contains `href="/ui/reporting?org=acme"`
- [x] 4.3 `/ui/reporting?org=acme` page body contains the org's
  display name in a Scope label
- [x] 4.4 `/ui/targets?org=acme` filters to acme's targets only
- [x] 4.5 `/ui/scans/{id}` HTML contains a Reporting nav link
  (regression test for the missing tab)

## 5. Docs

- [x] 5.1 `docs/architecture.md` "Read-only operator UI" — short
  note that scope persists across the nav via querystring
