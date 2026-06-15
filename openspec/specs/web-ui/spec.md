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

The `internal/ui` package SHALL register only HTTP GET handlers
and SHALL NOT register any handler that mutates store state
(POST, PUT, PATCH, DELETE), enforced by a static-analysis test
that greps the package for those method names and fails the build
if any are present.

#### Scenario: Mutation handlers fail the build

- **Given** a contributor adds `r.Post("/ui/foo", ...)` to
  `internal/ui/ui.go`
- **When** `go test ./internal/ui/...` runs
- **Then** the package's static-analysis test fails with a clear
  message naming the offending file

---

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
dimensions whose Score is `onbekend`, treating them as
not-evaluated rather than worst. If every dimension on a
target's most recent Assessment is `onbekend`, the target's
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

---

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

### Requirement: Analysis layer renders the steering matrix

The UI SHALL expose `/ui/analysis` (and
`/ui/analysis?org=<slug>`) as the central Analysis page. The
view MUST render a per-rule cross-target table with one row
per rule that has produced at least one Rationale, and columns
for distinct-target counts per `soeverein`, `voldoende`,
`afhankelijk`, and `onbekend` score. Stable row order is
framework first (`wand` > `eucsf` > alphabetical), then ruleID
alphabetical within a framework. The page MUST honour the
`?org=<slug>` filter and render a "Scope: {orgName}" pill in
the header when active.

#### Scenario: Analysis matrix renders rules and counts

- **Given** rule `wand.juridisch.cert_issuer_eea` has fired on
  three targets — one soeverein, two afhankelijk — across the
  current organisation's scans
- **When** the operator opens `/ui/analysis`
- **Then** the row for that rule shows soeverein=1,
  afhankelijk=2, voldoende=0, onbekend=0
- **And** the row's rule cell links to
  `/ui/reporting/wand/wand.juridisch.cert_issuer_eea`

---

### Requirement: Reporting layer is the rule catalogue

The UI SHALL render `/ui/reporting` as a rule catalogue:
**every** registered rule from every framework, with the
framework, dimension, rule ID, human-readable description, and
rationale text. The catalogue MAY carry a compact "current
state" column per row — the worst score reached across the
snapshots in scope plus a triage hint of how many targets sit
at that score — but MUST NOT render the full per-score matrix
(that stays on Analysis). Each row links to the existing
per-rule deep-dive page at
`/ui/reporting/{framework}/{ruleID}`.

#### Scenario: Catalogue lists rules with descriptions

- **Given** the wand pack registers
  `wand.juridisch.cert_issuer_eea` and the eucsf pack registers
  `eucsf.sov2.cert_issuer_eu`
- **When** the operator opens `/ui/reporting`
- **Then** both rules appear in the catalogue with their
  description text

#### Scenario: Catalogue carries a per-rule status hint

- **Given** rule `wand.juridisch.cert_issuer_eea` has fired
  across the scope with a mix of soeverein + afhankelijk
  rationales
- **When** the operator opens `/ui/reporting`
- **Then** the row for that rule shows a compact "afhankelijk"
  status pill (the worst score reached) with a "X of Y
  targets" hint
- **And** the row does NOT show the full soeverein / voldoende
  / afhankelijk / onbekend score breakdown (that lives on
  Analysis)

#### Scenario: Rules without rationale render an explicit placeholder

- **Given** rule `wand.juridisch.cert_issuer_eea` has not yet
  produced a rationale (no scans assessed under wand)
- **When** the operator opens `/ui/reporting`
- **Then** the row's status column reads "no rationale yet"
  rather than a fake score

### Requirement: Dashboard is "is dit goed of niet"

The Dashboard at `/ui/` and `/ui/orgs/{slug}` SHALL render
high-level health information only: the headline-stats strip,
one verdict pill per framework (worst score reached across all
targets in scope), and the organisations list (instance-wide
view only). The page MUST NOT carry the rule × score-counts
matrix, the Top concerns table, or the External / Internal
posture distribution blocks — those answer steering questions
that belong on Analysis.

#### Scenario: Dashboard renders verdict pills, not the matrix

- **Given** an operator opens `/ui/`
- **When** the page renders
- **Then** the body contains a per-framework verdict pill
  (e.g. "wand · afhankelijk · 3 of 4 targets")
- **And** the body does NOT contain a "Top concerns" heading
- **And** the body does NOT contain a "Recent activity" heading
- **And** the body does NOT contain "External posture" or
  "Internal posture" headings

---

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
- **WHEN** they request `/ui/orgs/conduction`
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

