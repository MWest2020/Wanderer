# web-ui — persona restructure (delta)

## ADDED Requirements

### Requirement: Overview leads with the target fleet

The Overview at `/ui/` and `/ui/orgs/{slug}` SHALL render a **target
fleet** table as the first content block under `<main>`, above the
verdict pills and the sovereignty-by-flow rollup. The table MUST carry
one row per target in scope, sorted by domain, with: the domain, a
single headline verdict (the worst score across the target's preferred
assessment — wand if present, else any framework, computed the same way
as the dashboard verdict pills), the last scan time and status, and a
link to that scan's report at `/ui/scans/{id}/assessment`. A target
with a scan but no assessment renders an explicit "not assessed"
placeholder rather than a fake score. Org-scoped Overviews list only
that organisation's targets.

#### Scenario: Overview lists every target with verdict and report link

- **Given** the scope contains targets `a.example` (soeverein) and
  `b.example` (afhankelijk), each with a wand assessment
- **When** the operator opens `/ui/`
- **Then** the fleet table shows a row for `a.example` with a
  soeverein verdict and a row for `b.example` with an afhankelijk
  verdict
- **And** each row links to that target's `/ui/scans/{id}/assessment`
- **And** the fleet table appears before the verdict-pill section

#### Scenario: Unassessed target shows a placeholder, not a score

- **Given** target `c.example` has a completed scan but no assessment
- **When** the operator opens `/ui/`
- **Then** the row for `c.example` shows a "not assessed" placeholder
  in the verdict column

### Requirement: The scan assessment is the report

The per-scan assessment page at `/ui/scans/{id}/assessment` SHALL be
the report surface for one target (the Explorer drill-down). It MUST be
identified by the target **domain** as its heading and HTML title, with
the scan ID demoted to a secondary reference. It MUST be reachable by
clicking a target row on the Overview. The report MUST continue to
render the Sovereignty overview, the flow diagram, and a path to the
raw findings.

#### Scenario: Report is titled by domain and reached from the fleet

- **Given** the operator clicks the `a.example` row on `/ui/`
- **When** the report renders
- **Then** the page heading and `<title>` are `a.example` (not the
  scan ID)
- **And** the scan ID is still present as a secondary reference
- **And** the page links onward to the raw findings for that scan

### Requirement: Trends layer consolidates fleet rules

The UI SHALL expose `/ui/trends` (and `/ui/trends?org=<slug>`) as the
single Farmer layer: rules across the fleet. The page MUST render the
rule catalogue — every registered rule from every framework with
framework, dimension, rule ID, description, and a compact worst-score
status hint per rule in scope — and MUST provide access to the per-rule
cross-target score matrix (soeverein / voldoende / afhankelijk /
onbekend distinct-target counts), either inline or via a drill-in. Each
rule links to the per-rule deep-dive at
`/ui/reporting/{framework}/{ruleID}`. The page MUST honour the
`?org=<slug>` filter and render the active-scope pill when set.

The legacy routes `/ui/analysis` and `/ui/reporting` (with their
`?org=` variants) SHALL redirect to `/ui/trends`, preserving the
organisation scope, so existing deep links survive.

#### Scenario: Trends renders the catalogue and the matrix

- **Given** rule `wand.juridisch.cert_issuer_eea` has fired across the
  scope with a mix of soeverein and afhankelijk rationales
- **When** the operator opens `/ui/trends`
- **Then** the catalogue lists that rule with its description and a
  worst-score "afhankelijk" status hint
- **And** the per-rule score matrix (counts per score) is reachable
  from that row

#### Scenario: Legacy Analysis and Reporting routes redirect to Trends

- **Given** the operator opens `/ui/analysis?org=acme`
- **When** the request is handled
- **Then** the response redirects to `/ui/trends?org=acme`

## MODIFIED Requirements

### Requirement: Dashboard is "is dit goed of niet"

The Overview at `/ui/` and `/ui/orgs/{slug}` SHALL render high-level,
glanceable health information: the **target fleet** table (see "Overview
leads with the target fleet"), the headline-stats strip, one verdict
pill per framework (worst score reached across all targets in scope),
the sovereignty-by-flow rollup, and the organisations list
(instance-wide view only). The page MUST NOT carry the rule × score
matrix, the rule catalogue, a Top concerns table, or the External /
Internal posture distribution blocks — those steering/Farmer questions
live on Trends.

#### Scenario: Overview renders fleet and verdict pills, not the matrix

- **Given** an operator opens `/ui/`
- **When** the page renders
- **Then** the body contains the target fleet table
- **And** the body contains a per-framework verdict pill
- **And** the body does NOT contain a rule × score matrix
- **And** the body does NOT contain a "Top concerns" heading
- **And** the body does NOT contain "External posture" or "Internal
  posture" headings

### Requirement: DAR nav persists the active organisation scope

The UI nav SHALL collapse to two tabs — **Overview** and **Trends** —
and SHALL preserve the active organisation scope across them. The
Report is not a nav tab; it is reached by clicking a target on the
Overview. When the operator is viewing a per-organisation page (the
Overview at `/ui/orgs/{slug}`, or Trends with `?org=<slug>`), each nav
link MUST point at the same-org page on the destination tab:

- Overview → `/ui/orgs/<slug>`
- Trends → `/ui/trends?org=<slug>`

When no organisation is selected, the nav links MUST point at the
instance-wide views (`/ui/`, `/ui/trends`).

#### Scenario: Per-org Overview nav threads the slug

- **Given** the operator is on `/ui/orgs/acme`
- **When** the page renders
- **Then** the Trends nav link points at `/ui/trends?org=acme`
- **And** there is no separate Analysis or Reporting nav tab

## REMOVED Requirements

### Requirement: Analysis layer renders the steering matrix

**Reason:** Consolidated into the single Trends (Farmer) layer; the
steering matrix becomes a section/drill-in of `/ui/trends`. `/ui/analysis`
redirects to `/ui/trends`.

**Migration:** The per-rule cross-target matrix is preserved on Trends;
`/ui/analysis?org=<slug>` → `/ui/trends?org=<slug>`.

### Requirement: Reporting layer is the rule catalogue

**Reason:** Consolidated into the single Trends (Farmer) layer; the rule
catalogue becomes the index of `/ui/trends`. `/ui/reporting` redirects
to `/ui/trends`.

**Migration:** The catalogue and its per-rule status hint are preserved
on Trends; the per-rule deep-dive stays at `/ui/reporting/{fw}/{ruleID}`.
