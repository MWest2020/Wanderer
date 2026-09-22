# web-ui Specification

## Purpose
Read-only operator UI for Wanderer. Mounted at `/ui/` on the existing
chi router behind a `--ui` flag, served by `internal/ui` using Go
`html/template` and vanilla CSS. Provides browse access to targets,
scans, findings, drift, and assessments without exposing any mutating
endpoints — public-sector operators can review evidence in a browser
without enabling write paths.

## Requirements

### Requirement: Wanderer ships a read-only operator UI behind a flag

The `wanderer serve` command SHALL mount a read-only HTML interface
at `/ui/` on the existing chi router when `--ui` is set, rendering
its three pages (`/ui/`, `/ui/scans/{id}`,
`/ui/targets/{id}/drift`) with Go `html/template` and serving
vanilla CSS as static assets, so an operator can read scans,
findings, and drift in a browser without enabling any mutating
endpoint.

#### Scenario: UI flag enables the routes

- **Given** `wanderer serve --ui`
- **When** an operator opens `/ui/` in a browser
- **Then** the page lists every persisted Target with its last
  scan status and last Assessment score per framework

#### Scenario: UI flag absent keeps the routes off

- **Given** `wanderer serve` without `--ui`
- **When** any client requests `/ui/`
- **Then** the response status is 404
- **And** no template is rendered

#### Scenario: Scan page groups findings

- **Given** a stored Scan with findings across DNS, TLS, IP, HTTP
- **When** the operator opens `/ui/scans/<id>`
- **Then** the page renders one section per probe prefix
- **And** each finding's severity is colour-coded

---

### Requirement: UI authenticates via HTTP Basic when configured

When `--ui-htpasswd <file>` is set, every `/ui/*` request SHALL
require a matching credential from the htpasswd file, accepting
bcrypt and SHA-512 entries and rejecting MD5, with the htpasswd
file re-read on every request so an operator can rotate
credentials without restarting the binary.

#### Scenario: Bcrypt entry accepts the right password

- **Given** an htpasswd file containing one bcrypt entry for user
  `op` with password `correct horse battery staple`
- **When** a request to `/ui/` arrives with that Basic header
- **Then** the response is the index page with status 200

#### Scenario: Wrong password rejected

- **Given** the same htpasswd file
- **When** a request arrives with `op:wrong`
- **Then** the response status is 401
- **And** the `WWW-Authenticate: Basic` header is set

#### Scenario: MD5 entry rejected at config load

- **Given** an htpasswd file whose first entry uses MD5 (`$apr1$`)
- **When** the server starts
- **Then** the process exits non-zero
- **And** stderr names MD5 as the unsupported algorithm

---

### Requirement: UI surface stays read-only

The UI SHALL remain read-only except for starting a scan, which a
signed-in user MAY do from the entry surface. Every other mutation
(targets, organisations, assessments, exports) SHALL stay with the API
and the CLI. When the instance has no authentication configured at all,
the scan route SHALL be refused rather than served open.

#### Scenario: Scanning is the only mutation

- **WHEN** the UI is served
- **THEN** the only route that changes state is the scan route, and it
  requires a signed-in user

#### Scenario: Mutation handlers fail the build

- **Given** a contributor adds `r.Post("/ui/foo", ...)` to
  `internal/ui/ui.go`, other than the sanctioned scan route
- **When** `go test ./internal/ui/...` runs
- **Then** the package's static-analysis test fails with a clear
  message naming the offending file

#### Scenario: No authentication configured

- **GIVEN** an instance started without OIDC and without htpasswd
- **WHEN** a request posts to the scan route
- **THEN** it is refused, and the startup log says the scan route is
  disabled for lack of authentication

### Requirement: UI renders the persisted Assessment per scan as the Analysis layer

The read-only operator UI SHALL mount a `/ui/scans/{id}/assessment`
GET handler that renders every persisted `models.Assessment` for
the scan as one card per dimension, with the dimension's score
badge, completeness flag, and one row per fired Rationale showing
the rule's `CriteriumID`, `Score`, `Verdict`, the linked Evidence
Finding IDs, and the rule's `Description` plus `Rationale` looked
up from the live rule registry (`dictu.DefaultRules` /
`eucsf.DefaultRules`) at render time.

#### Scenario: Assessment renders for a scored scan

- **GIVEN** a scan with one persisted DICTU Assessment that
  contains rationales for the `cert_issuer_eea` and
  `caa_restricts_issuance` rules
- **WHEN** an operator opens `/ui/scans/{scan-id}/assessment`
- **THEN** the response status is 200
- **AND** the page renders one card per DICTU dimension that has
  at least one Rationale
- **AND** each rationale row shows the rule's score badge, the
  Verdict text, the rule's `Description`, and the rule's
  `Rationale` paragraph in an expandable detail block

#### Scenario: Both frameworks render side by side

- **GIVEN** a scan with two persisted Assessments (one
  `Framework: dictu`, one `Framework: eucsf`)
- **WHEN** the operator opens the assessment page
- **THEN** the page renders one column per framework, each with
  its own dimension cards and rule rows

#### Scenario: Scan without an Assessment shows a hint

- **GIVEN** a scan with zero persisted Assessments
- **WHEN** the operator opens `/ui/scans/{scan-id}/assessment`
- **THEN** the response status is 200
- **AND** the page renders a one-line hint that no Assessment has
  been produced and shows the `wanderer assess <scan-id>` command
  the operator should run

#### Scenario: Retired-rule rationale degrades gracefully

- **GIVEN** an Assessment containing a rationale whose
  `CriteriumID` is no longer present in the live rule registry
- **WHEN** the operator opens the assessment page
- **THEN** the row renders the persisted score, verdict, and
  evidence unchanged
- **AND** the description column reads "rule retired" instead of
  the live rule description

---

### Requirement: Scan-detail page links to the Analysis page when an Assessment exists

The existing `/ui/scans/{id}` template SHALL include a prominent
link to `/ui/scans/{id}/assessment` when at least one persisted
Assessment exists for the scan, and SHALL omit the link
otherwise.

#### Scenario: Link present when scored

- **GIVEN** a scan with at least one persisted Assessment
- **WHEN** the operator opens `/ui/scans/{scan-id}`
- **THEN** the page renders an "Open assessment" link pointing at
  `/ui/scans/{scan-id}/assessment`

#### Scenario: Link absent when not scored

- **GIVEN** a scan with zero persisted Assessments
- **WHEN** the operator opens `/ui/scans/{scan-id}`
- **THEN** the page does not render an "Open assessment" link

---

### Requirement: UI dashboard surfaces aggregate posture at /ui/

The read-only UI's `/ui/` route SHALL render a dashboard view
containing three sections: a posture summary (counts of targets
per worst-dimension score per framework), top concerns (rule
rationales most commonly scored `afhankelijk` across targets,
target-counted), and recent activity (the five most recent scans
across the estate).

#### Scenario: Posture summary counts every target's worst dimension

- **GIVEN** three targets each with one persisted DICTU
  Assessment whose worst populated dimension scores are
  `soeverein`, `afhankelijk`, and `onbekend` respectively
- **WHEN** an operator opens `/ui/`
- **THEN** the response status is 200
- **AND** the DICTU posture summary block reports `1 soeverein`,
  `1 afhankelijk`, `1 onbekend`

#### Scenario: Top concerns counts each rule once per target

- **GIVEN** an estate where the rule
  `dictu.juridisch.cert_issuer_eea` fired `afhankelijk` on 50
  rationales spread across 5 distinct targets
- **WHEN** the operator opens the dashboard
- **THEN** the "top concerns" section lists
  `dictu.juridisch.cert_issuer_eea` with a target count of 5

#### Scenario: Recent activity is chronological across the estate

- **GIVEN** seven persisted scans across multiple targets
- **WHEN** the operator opens the dashboard
- **THEN** the recent activity section lists exactly five entries
- **AND** they are ordered newest first by `started_at`
- **AND** each entry links to the per-scan Analysis page at
  `/ui/scans/{id}/assessment` when an Assessment exists, or to
  the scan-detail page at `/ui/scans/{id}` otherwise

#### Scenario: Empty store renders empty-state copy

- **GIVEN** a fresh store with zero scans and zero assessments
- **WHEN** the operator opens `/ui/`
- **THEN** the dashboard renders without panicking
- **AND** each section displays empty-state copy explaining the
  next action ("run `wanderer scan <domain>` to populate")

---

### Requirement: Worst-dimension score excludes onbekend dimensions

The dashboard's per-target "worst dimension score" SHALL ignore
dimensions whose Score is `onbekend`, dimensions reported as not
applicable (every rationale structural), and dimensions absent from
the Assessment (scans made before the dimension existed), treating
them as not-evaluated rather than worst. The score SHALL be shown
together with the dimensions it was computed over. If no dimension on
a target's most recent Assessment is evaluated, the target's
worst-dimension score SHALL be `onbekend`.

#### Scenario: One onbekend dimension does not drag the rest down

- **GIVEN** a target whose latest DICTU Assessment has
  `juridisch: afhankelijk`, `operationeel: soeverein`, and
  `data_ai: onbekend`
- **WHEN** the dashboard computes the target's worst score
- **THEN** the result is `afhankelijk`
- **AND** `onbekend` is not counted as worse than `afhankelijk`

#### Scenario: All-onbekend target is reported as onbekend

- **GIVEN** a target whose latest DICTU Assessment has every
  dimension at `onbekend` (e.g. only the perimeter probes ran and
  GeoLite2 was unavailable)
- **WHEN** the dashboard computes the worst score
- **THEN** the result is `onbekend`

#### Scenario: Old scan without the new dimension

- **GIVEN** a target whose latest Assessment predates the
  accountability dimension
- **WHEN** the dashboard computes the worst score
- **THEN** the score is computed over the dimensions present
- **AND** the row states which dimensions it covers, and does not
  count accountability as failed or silently average it in

### Requirement: Flat targets table relocates to /ui/targets

The previous `/ui/` flat targets table SHALL be served at
`/ui/targets` after this change, with no behavioural change
beyond the URL.

#### Scenario: Auditor view preserved at new URL

- **GIVEN** a store with three targets and persisted scans /
  assessments per target
- **WHEN** an operator opens `/ui/targets`
- **THEN** the response status is 200
- **AND** the page renders the same per-target table the previous
  `/ui/` index rendered before this change (one row per target,
  last-scan status, framework score badges)

#### Scenario: Dashboard links to the targets table

- **GIVEN** the dashboard at `/ui/`
- **WHEN** the operator looks at the posture summary section
- **THEN** an "All targets" link points at `/ui/targets`

### Requirement: `wanderer serve` MAY load settings from a YAML config file

The `wanderer serve` command SHALL accept a `--config <path>`
flag (and `WANDERER_CONFIG` env var equivalent) that loads a
YAML file covering its operator-tunable settings: `listen`, `db`,
`geoip.{asn,country,optional}`, `ui.{enabled,htpasswd}`,
`schedules`, and `scan.{per_probe_timeout,budget,user_agent,
allow_private_targets}`. The YAML parse MUST be strict — unknown
fields fail the process at startup with an error naming the bad
field, never silently defaulted. When `--config` is unset, the
command SHALL behave byte-identically to the no-YAML form.

#### Scenario: Empty config equals no config

- **Given** a config file containing only `{}`
- **When** `wanderer serve --config empty.yaml` runs
- **Then** every setting resolves to its hard-coded default
- **And** the process behaves identically to `wanderer serve`
  with no `--config` flag

#### Scenario: YAML value applied when no flag or env

- **Given** a config file with `listen: ":9090"`
- **When** `wanderer serve --config x.yaml` runs with
  `WANDERER_LISTEN` unset and no `--addr` flag
- **Then** the HTTP server listens on `:9090`

#### Scenario: Unknown field rejected

- **Given** a config file containing `htpasswrd: /etc/htpasswd`
  (typo for `htpasswd`, under the wrong nesting)
- **When** `wanderer serve --config x.yaml` runs
- **Then** the process exits non-zero before opening any port
- **And** stderr contains an error naming the unknown field

---

### Requirement: Setting precedence is flag, env, YAML, default

`wanderer serve` SHALL resolve every setting by applying the
highest-precedence layer that is present, in the order: (1) CLI
flag explicitly passed on the command line; (2) environment
variable explicitly set in the process env; (3) YAML config
value; (4) hard-coded default. A flag explicitly set to its
default value (e.g. `--ui=false`) MUST still count as
"explicitly set" and override a YAML value.

#### Scenario: Flag overrides YAML

- **Given** a config file with `listen: ":9090"`
- **When** `wanderer serve --config x.yaml --addr :7070` runs
- **Then** the HTTP server listens on `:7070`

#### Scenario: Env var overrides YAML

- **Given** a config file with `db: /var/lib/wanderer/wanderer.db`
- **And** `WANDERER_DB=/tmp/test.db` set in the process env
- **When** `wanderer serve --config x.yaml` runs without `--db`
- **Then** the store is opened against `/tmp/test.db`

#### Scenario: Explicit flag-false overrides YAML true

- **Given** a config file with `ui.enabled: true`
- **When** `wanderer serve --config x.yaml --ui=false` runs
- **Then** the UI is not mounted at `/ui/`

### Requirement: Dashboard headline summarises instance coverage

The `/ui/` dashboard SHALL render a headline section at the top
of the page describing the instance's coverage at a glance: the
timestamp of the most recent scan, the total number of scans
recorded, the number of perimeter targets (Targets with
`Kind=domain`), the number of agent-host targets (Targets with
`Kind=host`), and the list of frameworks for which at least one
Assessment has been persisted. The headline MUST be the first
content below the page navigation; no operator should need to
scroll to see it.

#### Scenario: Headline with mixed coverage

- **Given** a store with 2 perimeter targets, 1 agent host, 5
  scans, and persisted Assessments under `wand` and `eucsf`
- **When** the operator opens `/ui/`
- **Then** the headline shows "5 scans", "2 perimeter targets",
  "1 agent host", and lists `wand` and `eucsf` as scored frameworks
- **And** the headline is rendered before any posture section

#### Scenario: Empty store still renders headline

- **Given** a store with no scans at all
- **When** the operator opens `/ui/`
- **Then** the page still renders a headline section
- **And** it explicitly says "no scans yet" rather than omitting
  the labels

---

### Requirement: Dashboard splits posture into external and internal

The `/ui/` dashboard SHALL render two posture blocks: an
**External posture** block aggregating only Targets with
`Kind=domain`, and an **Internal posture** block aggregating
only Targets with `Kind=host`. Each block MUST surface its empty
state explicitly when the corresponding scope has no Findings,
so an operator can see at a glance whether host-side coverage is
present or missing — never silently absent.

#### Scenario: External-only deployment shows internal empty-state

- **Given** a store with 2 perimeter targets, no agent hosts, and
  Assessments on the perimeter scans
- **When** the operator opens `/ui/`
- **Then** the External posture block shows the framework counts
- **And** the Internal posture block renders an empty-state line
  pointing the operator at `wanderer agent`

#### Scenario: Mixed deployment shows both

- **Given** a store with both perimeter and agent-host targets,
  each with persisted Assessments
- **When** the operator opens `/ui/`
- **Then** both External and Internal posture blocks render
  their respective framework counts
- **And** the framework count totals reflect only the targets in
  each scope (no double-counting)

### Requirement: Reporting index lists rules across targets

The UI SHALL expose `/ui/reporting`, listing every rule that has
produced at least one Rationale across the persisted Assessments,
with one row per rule and per-score columns (`soeverein`,
`voldoende`, `afhankelijk`, `onbekend`) counting **distinct
target IDs** per score. A rule that fires multiple times on the
same target SHALL count once. Rule rows MUST be ordered
deterministically: framework first (`wand` > `eucsf` >
alphabetical), then rule ID alphabetical within each framework.

#### Scenario: Rule firing on two targets with different scores

- **Given** rule `wand.juridisch.cert_issuer_eea` scored
  `soeverein` on target A and `afhankelijk` on target B (each via
  the most recent Assessment)
- **When** the operator opens `/ui/reporting`
- **Then** the row for that rule shows soeverein=1, afhankelijk=1
- **And** other score columns (voldoende, onbekend) show 0

#### Scenario: Rule firing twice on same target counts once

- **Given** target A's most recent Assessment contains two
  Rationale entries for the same rule (multi-host MX dimension,
  for example), both `afhankelijk`
- **When** the operator opens `/ui/reporting`
- **Then** the rule's afhankelijk count is 1, not 2

---

### Requirement: Reporting rule detail page lists targets

The UI SHALL expose
`/ui/reporting/{framework}/{ruleID}`, showing the rule's
description (from the in-process rule registry) plus a row per
target the rule has fired on, ordered by score severity
(`afhankelijk` rows first, then `voldoende`, `soeverein`,
`onbekend`). Each row's target cell SHALL link to the originating
scan's assessment page (`/ui/scans/{scanID}/assessment`). When
a target has more than one Assessment for the rule's framework,
the **most recent** Assessment SHALL drive the row. Unknown
rules (neither registered framework knows the criteriumID) SHALL
return HTTP 404 with no body content beyond the 404 default.

#### Scenario: Detail page renders rule and targets

- **Given** rule `wand.juridisch.cert_issuer_eea` registered in
  the wand pack and fired on two targets
- **When** the operator opens
  `/ui/reporting/wand/wand.juridisch.cert_issuer_eea`
- **Then** the page returns 200
- **And** shows the rule's description text from the registry
- **And** lists both targets with their score and Verdict

#### Scenario: Unknown rule returns 404

- **Given** no registered rule matches
  `wand.juridisch.does_not_exist`
- **When** the operator opens
  `/ui/reporting/wand/wand.juridisch.does_not_exist`
- **Then** the response status is 404

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

### Requirement: Playwright suite runs against deterministic seeded DBs

The Playwright suite SHALL produce its source-of-truth DBs from
a hermetic seeder rather than depend on an operator's hand-rolled
scan history. A developer cloning the repo SHALL be able to run
`make playwright` end-to-end without any prior `wanderer scan`
or `wanderer agent` invocations.

#### Scenario: Fresh clone runs Playwright

- **GIVEN** a clean checkout with no `/tmp/wanderer-demo.db` present
- **WHEN** the developer runs `make playwright`
- **THEN** the suite builds three fixture DBs (baseline /
  agent-host / empty-org), boots `wanderer serve` against each,
  and every spec passes without `test.skip` gating

#### Scenario: Schema migration breaks a fixture cleanly

- **GIVEN** a schema change that the seeder's literal SQL no
  longer satisfies
- **WHEN** `make playwright-fixture` runs
- **THEN** the build fails at fixture compile time, before
  Playwright ever opens a browser

---

### Requirement: Three fixture scenarios cover the spec set

The fixture seeder SHALL produce three independent SQLite files:
`baseline` (two orgs, perimeter scans), `agent-host` (adds a
synthetic agent inventory + Nextcloud surface with one
US-telemetry hit and one US objectstore + IdP), and `empty-org`
(an org with zero targets). Each Playwright project pins to one
scenario via `testMatch`.

#### Scenario: Host rule deep-dive shows the synthetic agent hit

- **GIVEN** the `agent-host` fixture loaded
- **WHEN** an operator opens
  `/ui/reporting/wand/wand.host.no_us_telemetry_packages`
- **THEN** the per-target table contains an `alma` row scored
  `afhankelijk` with `datadog-agent` named in the verdict

#### Scenario: Empty-org placeholder renders

- **GIVEN** the `empty-org` fixture loaded
- **WHEN** an operator opens `/ui/targets` for that org
- **THEN** the page renders an empty-state placeholder rather
  than failing to load

### Requirement: Wanderer can accept Nextcloud as the OIDC provider for `wanderer serve --ui`

Wanderer SHALL, when the `serve.yaml` `oidc:` block is
configured, redirect unauthenticated `/ui/*` requests to the
configured Nextcloud's authorize endpoint, exchange the
returned code on the callback URL, and set a session cookie
keyed against a server-side SQLite session table. A
Nextcloud-side disable of the user SHALL cut off Wanderer UI
access on the next request.

#### Scenario: Authenticated browse honours the Nextcloud session

- **GIVEN** OIDC is configured against `cloud.example.nl` and
  an operator has authenticated successfully
- **WHEN** they request `/ui/orgs/voorbeeld`
- **THEN** the page renders without an additional login
  prompt

#### Scenario: Nextcloud-side disable cuts Wanderer access

- **GIVEN** an authenticated operator's Nextcloud account is
  disabled while their Wanderer session cookie is still
  valid
- **WHEN** they request any `/ui/*` page
- **THEN** the request redirects to `/ui/login` and the
  Nextcloud authorize endpoint refuses re-authentication

#### Scenario: OIDC outage leaves htpasswd fallback usable

- **GIVEN** OIDC is configured AND htpasswd is also
  configured AND the OIDC provider is unreachable
- **WHEN** an operator requests `/ui/` with
  HTTP Basic credentials matching htpasswd
- **THEN** the request renders normally

### Requirement: The assessment view synthesises a Sovereignty overview

The assessment page SHALL render a "Sovereignty overview" that
synthesises the scan's scored rule rationales into an ordered set of
flows — Hosting, Mail, DNS, Transit path, CDN / hyperscaler, Third
parties — each shown with the rule's observed verdict and its score.
The overview SHALL derive solely from the existing assessment data (no
new collection, no jurisdiction logic in the view). When no flow rule
fired, the overview SHALL be omitted rather than shown empty.

#### Scenario: A scored scan shows the synthesis panel

- **GIVEN** a scan whose wand assessment contains flow rules (apex
  hosting, mail, DNS, hyperscaler, third parties)
- **WHEN** an operator opens `/ui/scans/{id}/assessment`
- **THEN** a "Sovereignty overview" section lists each flow with its
  category, score pill, and observed verdict

#### Scenario: No flow rules → no empty panel

- **GIVEN** a scan whose assessment contains none of the flow rules
- **WHEN** the assessment page renders
- **THEN** the Sovereignty overview section is absent

### Requirement: The dashboard rolls up sovereignty flows across targets

The dashboard (instance-wide and per-organisation) SHALL render a
"Sovereignty by flow" section that aggregates the per-target flows
across the scans in scope into one row per flow category — showing the
number of targets assessed for that flow, how many fall outside the
EEA, and the worst score reached. It SHALL derive solely from existing
assessment data and SHALL be omitted when no target has a scored flow.

#### Scenario: An organisation with mixed mail hosting

- **GIVEN** an organisation whose targets' mail routing is scored,
  some outside the EEA
- **WHEN** an operator opens the organisation dashboard
- **THEN** the "Sovereignty by flow" section shows a Mail row with the
  outside-EEA count and the worst score reached

### Requirement: The assessment view renders a no-JS sovereignty flow diagram

The assessment page SHALL render, beside the Sovereignty overview
table, a server-rendered inline SVG hub-and-spoke diagram of the same
flows: the target at the centre and each flow as a spoke whose node is
coloured by the flow's score. The diagram SHALL require no JavaScript
to render and SHALL be derived solely from the existing flow data. It
SHALL be omitted when there are no flows.

#### Scenario: A scored scan shows the flow diagram

- **GIVEN** a scan whose assessment yields one or more sovereignty flows
- **WHEN** an operator opens the assessment page
- **THEN** an `svg.sov-diagram` renders with a central hub and one
  score-coloured node per flow, without any JavaScript

### Requirement: An opt-in dev mode lets the UI trigger a scan

When `wanderer serve --ui-allow-scan` is set, the UI SHALL render a
"Scan a target" form and accept `POST /ui/scan`, which scans the
submitted target, assesses it, and redirects to the target scan's
assessment page. When the flag is not set, the UI SHALL expose neither
the form nor the route and SHALL remain read-only (no mutating
handlers other than the sanctioned scan route exist). The scanner's
private-target guard SHALL continue to apply.

#### Scenario: Dev mode scans from the browser

- **GIVEN** serve is started with `--ui-allow-scan`
- **WHEN** an operator submits a target on the dashboard scan form
- **THEN** a scan runs, an assessment is produced, and the browser is
  redirected to that scan's assessment page

#### Scenario: Read-only by default

- **GIVEN** serve is started without `--ui-allow-scan`
- **WHEN** a client POSTs to `/ui/scan`
- **THEN** the request does not trigger a scan (the route is not mounted)

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

### Requirement: The entry surface asks for a domain and answers it

`/ui/` SHALL lead with a single input that takes a domain (or selects a
host that reports through an agent) and, on submit, starts a scan and
navigates to that target's answer page. Below it SHALL sit the recently
answered targets as one-line verdicts. Scan IDs, rule IDs and the rule
matrix SHALL NOT appear on this surface. Submitting SHALL require a
signed-in user.

#### Scenario: A person types a domain

- **GIVEN** a signed-in operator on `/ui/`
- **WHEN** they enter `voorbeeld.nl` and submit
- **THEN** a scan starts and they land on that target's answer page
- **AND** they never see a scan ID to do it

#### Scenario: Not signed in

- **GIVEN** a request without a session
- **WHEN** it posts a domain to the scan route
- **THEN** it is refused and redirected to the login, and no scan starts

---

### Requirement: The answer is one sentence with its reason

The answer page SHALL open with one sentence in Dutch stating ja, nee
or onbekend for "does this stand under Dutch / European law?", naming
the observation that decided it. `soeverein` and `voldoende` SHALL read
as ja; `afhankelijk` SHALL read as nee; `onbekend` SHALL stay onbekend
and SHALL NEVER be promoted to ja. When questions could not be
answered, the headline SHALL say how many.

#### Scenario: Mail abroad decides the verdict

- **GIVEN** an assessment where every flow is soeverein except mail,
  which is afhankelijk on a US provider
- **WHEN** the answer page renders
- **THEN** the headline reads "Nee" and names the mail provider

#### Scenario: Unknown is not a yes

- **GIVEN** an assessment where hosting is soeverein and DNS is
  onbekend because GeoIP was unavailable
- **WHEN** the answer page renders
- **THEN** the headline does not read a plain "Ja", and states that one
  question could not be answered

---

### Requirement: The answer fills in while the scan runs

The answer page SHALL render from the findings that exist at that
moment and SHALL refresh itself until the scan is complete, without the
operator reloading. Each flow SHALL show its own state (nog bezig /
answered / niet gemeten). The page SHALL work without JavaScript, in
which case it falls back to a meta-refresh.

#### Scenario: Slow probe does not hold the page

- **GIVEN** a running scan where DNS and TLS have landed and the
  traceroute has not
- **WHEN** the operator opens the answer page
- **THEN** the DNS and TLS flows show their verdicts and the transit
  flow shows "nog bezig"

#### Scenario: Scan finishes while the page is open

- **GIVEN** the answer page open on a running scan
- **WHEN** the last probe lands
- **THEN** the page shows the final verdict without a manual reload

---

### Requirement: The reasoning is one click from the answer

From the answer page one link SHALL lead to the reasoning: the seven
sovereignty flows plus accountability, each as a plain-language
question with ja / nee / onbekend / n.v.t., the observed fact in the
verdict, and the evidence collapsed beneath. Rule IDs and protocol
jargon SHALL appear only inside the evidence.

#### Scenario: Explorer opens the reasoning

- **GIVEN** an answer page for a completed scan
- **WHEN** the operator follows the link to the reasoning
- **THEN** each flow is a question with an answer, and no rule ID
  appears outside an expanded evidence block

### Requirement: Domeinen zijn bij te houden als vloot

De UI SHALL een ingelogde gebruiker domeinen laten toevoegen aan en
verwijderen uit de vloot van een organisatie, los van het doen van een
scan. Per domein SHALL zichtbaar zijn wanneer het voor het laatst
gescand is en volgens welk schema. Een domein verwijderen SHALL de
eerdere scans en oordelen niet weggooien.

#### Scenario: Domein toevoegen zonder te scannen

- **GIVEN** een ingelogde gebruiker op het vlootscherm
- **WHEN** die `voorbeeld.nl` toevoegt
- **THEN** staat het domein in de vloot, met "nog niet gescand" als
  laatste scan

#### Scenario: Verwijderen laat de geschiedenis staan

- **GIVEN** een domein met drie eerdere scans
- **WHEN** het uit de vloot wordt gehaald
- **THEN** verdwijnt het uit het overzicht en blijven de scans
  raadpleegbaar

---

### Requirement: Het vlootscherm scoort x van n, niet ja of nee

Het vlootscherm SHALL per domein tonen hoeveel vragen soeverein zijn
beantwoord van het aantal dat beantwoord kón worden (`x/n`), met het
aantal onbeantwoorde vragen apart ernaast. Een onbeantwoorde vraag
SHALL niet meetellen in n en SHALL nooit als geslaagd gelden. Naast de
score SHALL de zwaarste openstaande bevinding staan, zodat een hoge
score geen makkelijke gaten verbergt. Een percentage MAY getoond worden
om op te sorteren; `x/n` SHALL de getoonde waarde zijn.

#### Scenario: Twee domeinen naast elkaar

- **GIVEN** een domein met 5 van 7 soeverein en 2 onbekend, en een
  domein met 6 van 7 en 0 onbekend
- **WHEN** het vlootscherm rendert
- **THEN** staat er "5/7 · 2 onbekend" respectievelijk "6/7", en niet
  twee keer "nee"

#### Scenario: Zwaarste bevinding staat erbij

- **GIVEN** een domein dat 6 van 7 scoort maar waarvan de mail via een
  Amerikaanse aanbieder loopt
- **WHEN** de rij rendert
- **THEN** staat die bevinding naast de score genoemd

---

### Requirement: Een regel legt zichzelf uit, inclusief de drempel

De regelpagina SHALL per regel tonen: wat hij controleert, waarom het
uitmaakt, welke waarneming hij gebruikt, en welke drempel het oordeel
bepaalt — als waarde, in mensentaal ("verloopt binnen 30 dagen",
"meer dan de helft van de nameservers"). De pagina SHALL tonen welke
domeinen nu aan welke kant van die drempel staan. Een drempel SHALL
niet alleen in de vergelijking in de code bestaan.

#### Scenario: Drempel is zichtbaar

- **GIVEN** de regel `wand.operationeel.domain_expiry`
- **WHEN** een gebruiker de regelpagina opent
- **THEN** staat er dat het oordeel omslaat bij 90 en bij 30 dagen

#### Scenario: Wie staat waar

- **GIVEN** vijf domeinen waarvan twee binnen 30 dagen verlopen
- **WHEN** de regelpagina rendert
- **THEN** staan die twee apart van de andere drie

---

### Requirement: Bij elk oordeel "nee" staat wat je eraan doet

Elke regel SHALL één concrete handeling kennen voor het geval hij
afhankelijk scoort, in dezelfde Nederlandse tekstentabel als de rest
van de copy. Die handeling SHALL benoemen wat er moet gebeuren en waar
("vraag je registrar de privacyproxy op voorbeeld.nl te verwijderen"),
niet een doel herhalen ("zorg voor een herkenbare registrant"). De
regelpagina en de onderbouwing SHALL die handeling tonen bij een
falend oordeel. Een regel zonder handeling SHALL de tests laten falen,
net zoals een regel zonder `Rationale` dat nu doet.

#### Scenario: Falende regel toont de handeling

- **GIVEN** een domein waarvan de mail buiten de EER landt
- **WHEN** de gebruiker de onderbouwing of de regelpagina opent
- **THEN** staat er één concrete handeling, met het domein erin genoemd

#### Scenario: Regel zonder handeling

- **GIVEN** een nieuwe regel zonder tekst voor de handeling
- **WHEN** de tests draaien
- **THEN** falen ze met een melding die de regel noemt

### Requirement: De demopagina toont één domein en verder niets

De UI SHALL een publieke route `/demo` kennen die zonder aanmelding de
laatste voltooide scan toont van het domein uit `demo.target`: het
antwoord in één zin, de stromen als vragen met antwoorden, en het
uitklapbare bewijs. De pagina SHALL geen andere domeinen noemen, geen
vloot, geen organisaties en geen regelpagina's met andere targets
tonen, en SHALL geen scan kunnen starten. Zonder `demo.target` SHALL
de route niet bestaan (404), en SHALL dat één keer in het opstartlog
staan.

#### Scenario: Bezoeker zonder account

- **GIVEN** een instantie met `demo.target: westerweel.work` en een
  voltooide scan van dat domein
- **WHEN** iemand zonder aanmelding `/demo` opent
- **THEN** ziet die het antwoord en de onderbouwing van westerweel.work,
  met de datum van de scan

#### Scenario: Geen andere domeinen

- **GIVEN** een instantie die ook andere domeinen volgt
- **WHEN** de demopagina rendert
- **THEN** komt geen enkel ander domein in de pagina voor, ook niet in
  een link of een regeloverzicht

#### Scenario: Demo staat uit

- **GIVEN** een instantie zonder `demo.target`
- **WHEN** iemand `/demo` opent
- **THEN** krijgt die 404, en heeft het opstartlog gezegd dat de demo
  uit staat

#### Scenario: Nog geen scan

- **GIVEN** `demo.target` is gezet maar er is nog geen voltooide scan
- **WHEN** iemand `/demo` opent
- **THEN** zegt de pagina dat er nog geen meting is, in plaats van een
  leeg oordeel te tonen

---

### Requirement: De demo start geen scans

De demoroute SHALL alleen tonen wat op schema is gemeten. Er SHALL geen
manier zijn om vanaf de demopagina een scan te starten, ook niet met een
handmatig verzoek.

#### Scenario: Scanpoging vanaf de demo

- **WHEN** iemand zonder aanmelding een scan probeert te starten via de
  demoroute
- **THEN** wordt dat geweigerd en start er geen scan

### Requirement: Oordelen zijn leesbaar zonder kleur en met genoeg contrast

Elk oordeel dat de UI toont SHALL bij zijn werkelijke tekstgrootte een
contrastverhouding van ten minste 4,5:1 halen. Kleur SHALL nooit het
enige onderscheid zijn tussen twee oordelen: er SHALL ook tekst of vorm
verschillen, zodat het zonder kleurwaarneming leesbaar blijft.

#### Scenario: Oordeelbadge in grijstinten

- **GIVEN** een pagina met de oordelen ja, nee, onbekend en n.v.t.
- **WHEN** die zonder kleur wordt bekeken
- **THEN** is elk oordeel nog te onderscheiden aan zijn tekst

#### Scenario: Contrast gemeten

- **WHEN** axe-core over een scherm draait
- **THEN** meldt het geen contrastfout op een oordeel

---

### Requirement: De brede schermen passen op een telefoon

Een pagina SHALL op 390px breed niet horizontaal scrollen. Een tabel die
breder is dan het scherm SHALL binnen zijn eigen kader scrollen, zodat
de rest van de pagina op zijn plaats blijft.

#### Scenario: Onderbouwing op een telefoon

- **GIVEN** een scherm van 390px
- **WHEN** de onderbouwingspagina rendert
- **THEN** scrollt de pagina niet horizontaal

### Requirement: Accountability renders as an answer sheet, not a rule dump

The assessment report page (`/ui/scans/{id}/assessment`) SHALL render
the accountability dimension as a list of plain-language questions,
one per rule, each answered ja / nee / onbekend / n.v.t. with a
one-sentence verdict in the same register. Rule IDs, RDAP field
names, and RFC numbers SHALL appear only inside the expandable
evidence block, never in the headline. Each failing rule SHALL show
exactly one concrete remediation line. n.v.t., onbekend, and nee
SHALL be visually distinct from one another.

#### Scenario: Tourist reads a failing registrant check

- **GIVEN** a `.com` scan where registrant_identifiable scored
  afhankelijk on a commercial privacy proxy
- **WHEN** an operator opens the assessment report
- **THEN** the accountability section shows the question "Is de
  registrant herkenbaar als uw organisatie?" answered "Nee" with the
  proxy named and a single remediation line
- **AND** `wand.accountability.registrant_identifiable` and RDAP
  jargon appear only after expanding the evidence

#### Scenario: Unknown is presented honestly

- **GIVEN** a scan where soa_rname scored onbekend with reason
  `probe_unavailable` after a timeout
- **WHEN** the operator opens the report
- **THEN** the answer reads "Onbekend — de DNS-SOA-opvraging kreeg
  geen antwoord", visually distinct from a failing "Nee", so absence
  of evidence is never dressed up as a defect

#### Scenario: Not applicable is not a gap

- **GIVEN** a `.nl` scan where registrant_identifiable scored onbekend
  with reason `registry_redacted`
- **WHEN** the operator opens the report
- **THEN** the answer reads "n.v.t. — het register publiceert
  registrantgegevens niet voor .nl", styled differently from
  "Onbekend", and shows no remediation line

---

### Requirement: Answer-sheet copy lives in one Dutch string table

All answer-sheet copy — per rule the question, the verdict text per
outcome, and the remediation line — SHALL ship in Dutch from one
per-rule string table (`internal/assessor/wand/accountability_nl.yaml`).
Templates and rule code SHALL NOT contain answer-sheet copy; rules
supply named parameters only. A test SHALL fail when a rule has no
entry or an entry uses a parameter the rule does not supply. There
SHALL be no i18n machinery in v1: a second language is a second
table.

#### Scenario: Every rule has complete copy

- **WHEN** the string table is loaded in tests
- **THEN** each of the seven rules has a question, a verdict for
  every outcome it can produce, and a remediation line for every
  failing outcome

#### Scenario: No copy in templates

- **WHEN** the report template is rendered for an accountability
  dimension
- **THEN** every visible question, verdict, and remediation string
  originates from the string table

---

### Requirement: Evidence is expandable and self-explaining

Each accountability question SHALL expand in place to the raw
findings that produced the verdict (registrant string, RNAME, fetched
security.txt fields, per-path variant table), each labelled in plain
language with the technical key alongside. The collapsed state SHALL
be the default; expanding one question SHALL NOT navigate away from
the report.

#### Scenario: Explorer drills into variant convergence

- **GIVEN** a variant_convergence verdict of voldoende
- **WHEN** the operator expands the question
- **THEN** an 8-row table shows each path, its status, redirect
  chain, and final origin, with the diverging path highlighted and
  budget-skipped or untested paths marked as such

#### Scenario: Evidence absent

- **GIVEN** an onbekend verdict backed only by an unavailable finding
- **WHEN** the operator expands the question
- **THEN** the block shows the failure (probe, error class, time)
  instead of an empty table

---

### Requirement: Overview surfaces accountability without adding a tab

The Overview (Tourist) page SHALL include the accountability verdict
in the existing per-target headline row (alongside the current
framework score), without introducing a new navigation tab, keeping
the two-tab IA of ADR-0017 intact.

#### Scenario: Fleet row shows the dimension

- **GIVEN** three targets with mixed accountability outcomes
- **WHEN** the operator opens `/ui/`
- **THEN** each target row carries an accountability pill whose
  colour matches the dimension verdict and links to that target's
  report section

#### Scenario: No assessment yet

- **GIVEN** a target scanned before this change (no accountability
  findings)
- **WHEN** the operator opens `/ui/`
- **THEN** the pill renders as "niet beoordeeld", not as a failure

---

### Requirement: Scanner limitations render as operator warnings

A rationale whose reason code has subject `scanner` SHALL NOT be
rendered as a property of the target. The report SHALL show it as an
operator environment warning next to the dimension, and the answer
itself SHALL be based on what was observed.

#### Scenario: Scanner without IPv6

- **GIVEN** a variant_convergence rationale with reason
  `scanner_no_ipv6`
- **WHEN** the operator opens the report
- **THEN** a warning reads "scanner heeft geen IPv6 — v6-paden niet
  gemeten", styled as an environment notice
- **AND** the question's answer reflects only the observed v4 paths

### Requirement: De vloot is de eerste laag

`/ui/` SHALL het vlootoverzicht van de gekozen organisatie tonen: één
score voor het geheel als `x/n` (de som van de soeverein beantwoorde
vragen over de som van de beantwoordbare), het aantal onbeantwoorde
vragen apart, het aantal domeinen dat niet soeverein is, en de
verdeling over de zeven stromen. De lijst SHALL standaard gesorteerd
zijn op de slechtste score eerst. Een invoerveld om een domein te
scannen SHALL aanwezig blijven, maar niet de hoofdzaak van de pagina
zijn.

#### Scenario: CISO opent de tool

- **GIVEN** een organisatie met vijf gescande domeinen
- **WHEN** een ingelogde gebruiker `/ui/` opent
- **THEN** ziet die één score voor de vloot, hoeveel domeinen niet
  soeverein zijn, en de domeinen met de slechtste bovenaan

#### Scenario: Een goede score verbergt geen slecht domein

- **GIVEN** negen domeinen die soeverein scoren en één dat faalt
- **WHEN** de vlootpagina rendert
- **THEN** staat naast de score dat één domein niet soeverein is, en
  staat dat domein bovenaan de lijst

---

### Requirement: Het domein toont zijn score en wat eraan te doen is

De antwoordpagina van één domein SHALL naast de oordeelzin de
`x/n`-score tonen, en SHALL bij elk niet-soeverein punt de handeling
tonen die bij die regel hoort. De zin SHALL blijven zeggen wat het
oordeel bepaalde.

#### Scenario: Domein met twee gebreken

- **GIVEN** een domein dat op mail en certificaat niet soeverein scoort
- **WHEN** de antwoordpagina rendert
- **THEN** staat er een score, de zin die het zwaarste punt noemt, en
  bij beide punten één concrete handeling

---

### Requirement: Eén taal per laag

De oordelen op de vloot-, antwoord- en onderbouwingspagina SHALL in het
Nederlands staan, uit de bestaande tekstentabel. Engelse veldnamen,
RFC-verwijzingen en ruwe probe-uitvoer SHALL alleen binnen het
uitgeklapte bewijs voorkomen.

#### Scenario: Geen half-Engelse regel

- **WHEN** een oordeel op de antwoordpagina rendert
- **THEN** staat er geen Engelstalige zin naast een Nederlandse vraag

#### Scenario: Bewijs blijft technisch

- **WHEN** iemand het bewijs uitklapt
- **THEN** mag daar de ruwe, Engelstalige waarneming staan
