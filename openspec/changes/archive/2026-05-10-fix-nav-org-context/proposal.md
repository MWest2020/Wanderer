# Proposal: persist org scope across DAR nav tabs

## Intent

The DAR nav (`Dashboard` / `Analysis` / `Reporting`) was wired up
in three separate proposals. Each tab works on its own, but they
do not cooperate when the operator has selected an organisation:

- From `/ui/orgs/acme` (per-org Dashboard) the **Analysis** tab
  links to `/ui/targets` — silently dumping the operator out of
  ACME and back into the instance-wide view.
- The **Reporting** tab supports `?org=<slug>` filtering, but the
  reporting page header does not say which org is active, so a
  filtered view looks identical to the global one until the
  operator inspects the URL.
- The **Reporting** tab is missing entirely from Analysis pages
  (`/ui/targets`, `/ui/scans/{id}`, `/ui/scans/{id}/assessment`,
  `/ui/targets/{id}/drift`) because those handlers pass
  `HasReporting: false` to `nav.tmpl` — left over from when
  Reporting did not yet exist.

Mark named the gap on 2026-05-05: *"ik selecteerd een org en
daar zie ik op dashboarding alles om te sturen. analysis verteld
me in detail en reprt dan idd de rules."* The mental model is
**one continuous experience inside the selected scope**. The
plumbing must reflect that.

## Scope

**In scope:**

- `nav.tmpl` accepts a new `OrgSlug` field. When non-empty, the
  Dashboard / Analysis / Reporting links carry the scope:
  - Dashboard → `/ui/orgs/<slug>`
  - Analysis → `/ui/targets?org=<slug>`
  - Reporting → `/ui/reporting?org=<slug>`
- Every UI page-handler passes `OrgSlug` (resolved from URL path
  param or `?org=` query) and `HasReporting: true` to nav.tmpl.
- `/ui/targets` accepts `?org=<slug>` and filters the targets
  list to that organisation (matches the `/ui/reporting` and
  `/ui/orgs/{slug}` pattern).
- The Reporting index and rule-detail pages render a visible
  "Scope: {orgName}" header line whenever a slug is active.
- The targets page header surfaces the same scope label.

**Out of scope:**

- Per-org sub-routes (`/ui/orgs/{slug}/targets`,
  `/ui/orgs/{slug}/scans/{id}`). The querystring-based filter is
  the boring choice and matches the existing
  `/ui/reporting?org=` shape introduced in
  `add-organisation-pivot`.
- Filtering the `/ui/scans/{id}/assessment` page itself by org
  — a single scan belongs to a single org by definition; the
  scope label on its breadcrumb is enough.
- Persisting org scope across browser sessions (cookie / local
  storage). The URL is the source of truth.

## Wand / EUCSF dimensions informed

None directly — operator-facing UX.

## Passive / active boundary

Template + handler refactor only. No new outbound calls; no new
data ingested.

## Parallel-safe

Touches `internal/ui/`: `nav.tmpl`, `dashboard.tmpl`,
`reporting.tmpl`, `reporting_rule.tmpl`, `index.tmpl` (the
targets page), `scan.tmpl`, `assessment.tmpl`, `drift.tmpl`, and
the matching handler functions in `ui.go`. No schema, no API
shape change, no new package.
