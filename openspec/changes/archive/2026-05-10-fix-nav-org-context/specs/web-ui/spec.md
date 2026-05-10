# Delta for web-ui

## ADDED Requirements

### Requirement: DAR nav persists the active organisation scope

Every UI page SHALL render its cross-page nav such that the
active organisation scope is preserved across the Dashboard,
Analysis, and Reporting tabs. When the operator is viewing a
per-organisation page (the Dashboard at `/ui/orgs/{slug}`, or
any Analysis or Reporting page with `?org=<slug>`), each nav
link MUST point at the same-org page on the destination layer:

- Dashboard → `/ui/orgs/<slug>`
- Analysis → `/ui/targets?org=<slug>`
- Reporting → `/ui/reporting?org=<slug>`

When no organisation is selected, the nav links MUST point at
the instance-wide views (`/ui/`, `/ui/targets`, `/ui/reporting`).

#### Scenario: Per-org dashboard nav threads the slug

- **Given** the operator is on `/ui/orgs/acme`
- **When** the page renders
- **Then** the Analysis nav link points at `/ui/targets?org=acme`
- **And** the Reporting nav link points at `/ui/reporting?org=acme`

#### Scenario: Reporting with org filter shows scope label

- **Given** the operator opens `/ui/reporting?org=acme`
- **When** the page renders
- **Then** the page header contains the organisation's display
  name in a "Scope" label so the filtered view is visibly distinct
  from the global one

#### Scenario: Analysis pages carry the Reporting nav link

- **Given** the operator opens `/ui/scans/{id}` or
  `/ui/scans/{id}/assessment`
- **When** the page renders
- **Then** the cross-page nav contains the Reporting nav link
  (not omitted as it was when the Reporting route did not exist)

---

### Requirement: Targets page accepts an organisation filter

`/ui/targets` SHALL accept an optional `?org=<slug>` query
parameter that limits the rendered targets list to that
organisation. An unknown slug MUST return HTTP 404. When the
filter is active, the page header MUST display the same "Scope"
label as the Reporting pages.

#### Scenario: Targets list filtered by org

- **Given** organisation `acme` exists and has two perimeter
  targets, while another organisation has different targets
- **When** the operator opens `/ui/targets?org=acme`
- **Then** the rendered table contains exactly acme's two
  targets and no others

#### Scenario: Unknown org slug returns 404

- **Given** no organisation has slug `nope`
- **When** the operator opens `/ui/targets?org=nope`
- **Then** the response status is 404
