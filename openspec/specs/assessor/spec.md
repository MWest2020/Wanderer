# assessor Specification

## Purpose
TBD - created by archiving change add-assessor. Update Purpose after archive.
## Requirements
### Requirement: Deterministic assessment

The assessor SHALL produce the same Assessment for the same set of
Findings, regardless of when or where it runs.

#### Scenario: Replay produces identical score

- **Given** a Scan `s_abc` with a frozen set of Findings
- **When** the assessor runs against it now
- **And** the assessor runs again an hour later without new Findings
- **Then** the two Assessments have identical per-dimension Scores
- **And** identical Rationale lists (ignoring ID and CreatedAt)

#### Scenario: Same findings, two machines

- **Given** the same Scan JSON exported from one Wanderer instance
- **When** a second Wanderer instance imports it and runs the assessor
- **Then** the resulting Assessment has identical Scores per dimension

---

### Requirement: Completeness transparency

The assessor SHALL record a `Completeness` flag per dimension so a
reader can distinguish "low score due to bad posture" from "low score
due to missing evidence".

#### Scenario: Perimeter-only scan

- **Given** a Scan produced entirely by the MVP probes (no inventory,
  no egress)
- **When** the assessor runs
- **Then** Data & AI dimension has `Completeness: incomplete` for rules
  requiring egress/OIDC data
- **And** the markdown report explicitly states this in the dimension
  section
- **And** the Score for an `incomplete` dimension is `onbekend`
  unless evidence-backed rules gave a lower score

#### Scenario: IP probe unavailable

- **Given** a Scan where `ip.unavailable` is the only IP-related
  finding
- **When** the assessor runs rules depending on IP ASN country
- **Then** those rules do not contribute a score
- **And** the dimension's Completeness is reduced from `complete` to
  `partial` or `incomplete`

---

### Requirement: Evidence citations

Every Rationale entry in an Assessment SHALL reference at least one
`Finding.ID` in its `Evidence` field, or be marked as
"no evidence — rule did not match".

#### Scenario: Verdict cites evidence

- **Given** the `Juridisch` dimension rule "Cert issuer in EU"
- **When** the rule matches a `tls.issuer` finding with
  `issuer_country: ["US"]`
- **Then** the resulting Rationale contains the finding's `ID` in its
  `Evidence` list
- **And** the markdown report renders the citation as
  `Evidence: tls.issuer (finding f_xyz)`

#### Scenario: Missing attribute handled gracefully

- **Given** a `tls.issuer` finding whose `Attributes` map lacks
  `issuer_country`
- **When** the corresponding rule runs
- **Then** the rule does not panic
- **And** the rule either contributes no verdict or contributes
  `Score: onbekend` with a verdict explaining the missing attribute
- **And** a `slog.Warn` is logged

---

### Requirement: CLI output formats

The `wanderer assess <scan-id>` command SHALL support at least three
output formats: human-readable text, markdown, and JSON.

#### Scenario: Markdown format

- **Given** a completed Scan
- **When** `wanderer assess <id> --format markdown` runs
- **Then** stdout contains a markdown document starting with
  `# Wanderer Assessment`
- **And** every dimension section includes a heading, a one-line
  verdict, and an "Evidence:" line per rule

#### Scenario: JSON format

- **Given** a completed Scan
- **When** `wanderer assess <id> --format json` runs
- **Then** stdout is a valid JSON object matching the `Assessment`
  schema in `pkg/models`
- **And** `jq -e '.dimensions | length == 5'` succeeds

#### Scenario: Missing scan

- **Given** a scan ID that does not exist in the store
- **When** `wanderer assess <id>` runs
- **Then** the process exits non-zero
- **And** stderr contains "scan not found"

---

### Requirement: Assessments are persisted

Every run of the assessor against a Scan via the HTTP API SHALL
produce a persisted `Assessment` record retrievable by its ID.

#### Scenario: Assessment survives restart

- **Given** an Assessment `a_abc` produced via `POST /scans/{id}/assessments`
- **When** the server restarts
- **And** the operator requests `GET /assessments/a_abc`
- **Then** the response contains the full Assessment record

#### Scenario: Re-running produces a new record

- **Given** an Assessment `a_1` already persisted for scan `s_abc`
- **When** `POST /scans/s_abc/assessments` is called again
- **Then** a new Assessment `a_2` is persisted
- **And** `a_1` remains retrievable

---

### Requirement: Assessor ships a SEAL rule pack alongside DICTU

The assessor SHALL provide a second rule pack implementing the EU
Cloud Sovereignty Framework (SEAL) under `internal/assessor/eucsf`,
selectable per assessment via `wanderer assess --framework
dictu|eucsf|both`, scoring on a 0–4 SEAL level scale that maps onto
the existing `models.Score` enum so the engine and the persisted
Assessment shape stay unchanged.

#### Scenario: SEAL framework selectable

- **Given** a stored Scan with TLS issuer attributes from a Dutch CA
- **When** the operator runs `wanderer assess <id> --framework eucsf`
- **Then** the resulting Assessment has `Framework: "eucsf"`
- **And** every Rationale's `CriteriumID` starts with `eucsf.`

#### Scenario: Both frameworks persist independently

- **Given** the same scan
- **When** the operator runs `wanderer assess <id> --framework both`
- **Then** two Assessments are persisted — one with
  `Framework: "dictu"` and one with `Framework: "eucsf"`
- **And** both cite the same Findings via their Evidence lists

#### Scenario: SEAL level maps to model Score

- **Given** an `eucsf.sov2.cert_issuer_eu` rule that fires SEAL level 4
- **When** the rule is evaluated
- **Then** the resulting Rationale has `Score: soeverein`
- **And** the SEAL level is recorded in `Attributes.seal_level: "seal_4"`

---

### Requirement: Probe-ID / rule-ID drift is build-breaking

Both the DICTU and SEAL rule packs SHALL each ship at least one
integration test that runs the relevant real probe against a fake
resolver / fake HTTP source through the real assessor, asserting
the actual ProbeID / attribute names the rule pack consumes, so a
casing mismatch or an attribute rename in either side breaks `go
test ./...` rather than silently producing Onbekend on production
scans.

#### Scenario: Casing change breaks the build

- **Given** the DNS probe currently emits `dns.a` and the
  `apex_ip_eea` rule consumes `dns.a`
- **When** a contributor accidentally changes one side to `dns.A`
- **Then** the integration test in
  `internal/assessor/dictu/integration_test.go` fails

#### Scenario: Attribute rename breaks the build

- **Given** the TLS probe currently writes `issuer_country` on
  `tls.issuer` and the `cert_issuer_eu` rule reads
  `issuer_country`
- **When** the probe renames the attribute to `issuerCountry`
- **Then** the eucsf integration test fails before the change can land

---

### Requirement: Rules ignore meta Findings

Every assessor rule SHALL skip Findings whose attributes mark them
as meta (an `error` attribute is present, `no_answer` is true, or
`unavailable` is true) when deciding whether evidence backs a
verdict, so a non-resolvable domain or a missing probe never
produces a positive score.

#### Scenario: NXDOMAIN does not score voldoende on MX presence

- **Given** a Finding set whose only `dns.mx` rows are
  `lookupError` records (each with an `error` attribute set)
- **When** the assessor runs the `dictu.data_ai.mx_present` rule
- **Then** the rule's RuleResult has `Score: onbekend`
- **And** the rule's `Evidence` list is empty

#### Scenario: A `no_answer` row is not evidence

- **Given** a Finding with `ProbeID: dns.caa` and
  `Attributes.no_answer: true`
- **When** the `dictu.operationeel.caa_restricts_issuance` rule
  evaluates the set
- **Then** the rule does not treat the row as a positive CAA
  observation
- **And** the rule's verdict reflects the absence of CAA records,
  not their presence

---

### Requirement: Every assessor Rule carries a plain-language Rationale

The `assessor.Rule` struct SHALL include a `Rationale string`
field that holds a one-paragraph plain-language explanation of
what the rule observes and why it matters for sovereignty
posture, populated alongside the existing `Description`. Every
Rule registered by `dictu.DefaultRules()` and
`eucsf.DefaultRules()` SHALL have a non-empty Rationale; an empty
string is a build-breaking error in the corresponding registry's
test suite.

#### Scenario: Rationale present on every default rule

- **GIVEN** the rule sets returned by
  `internal/assessor/dictu.DefaultRules()` and
  `internal/assessor/eucsf.DefaultRules()`
- **WHEN** a test iterates every Rule and reads `Rationale`
- **THEN** every Rule's `Rationale` is a non-empty string

#### Scenario: Empty Rationale fails CI

- **GIVEN** a contributor adds a new Rule with an empty
  `Rationale` field to either rule pack
- **WHEN** `go test ./internal/assessor/...` runs
- **THEN** the rule pack's `TestEveryRuleHasRationale` test fails
  with a message naming the offending `CriteriumID`

#### Scenario: Rationale is independent of Description

- **GIVEN** a Rule whose `Description` is a single-sentence
  summary ("TLS certificate issued by an authority in the EEA.")
- **WHEN** the renderer reads the Rule
- **THEN** `Rationale` is a separate string carrying the
  consequence of the rule firing
- **AND** `Description` and `Rationale` are not the same value

---

### Requirement: First-party rule pack is named `wand`, not `dictu`

The Conduction-owned rule pack SHALL be identified as `wand`
(Wanderer-NL) in every output Wanderer produces: the persisted
`Assessment.Framework` value, the rule IDs (under the
`wand.<dimension>.<short>` shape), the CLI flag value
(`--framework wand|eucsf|both`), and every documentation or UI
surface that names the framework. The DICTU
*Toetsingsinstrument Soevereiniteit Clouddiensten* SHALL be
credited in the assessor docs and ADR-0011 as the public
framework that inspired the rule set; the implementation,
ownership, and label are Conduction's.

#### Scenario: New assessment carries the wand framework label

- **GIVEN** the renamed rule pack is in production
- **WHEN** an operator runs `wanderer assess <scan-id> --framework wand`
- **THEN** the persisted `Assessment.Framework` is `"wand"`
- **AND** every persisted Rationale's `CriteriumID` starts with
  `wand.`

#### Scenario: Output documents the inspiration without claiming endorsement

- **GIVEN** a contributor reads `docs/assessor.md` after the rename
- **WHEN** they look for the relationship to DICTU
- **THEN** the doc explicitly names DICTU's *Toetsingsinstrument
  Soevereiniteit Clouddiensten* as the inspiration for the rule
  set
- **AND** the doc does not state or imply that DICTU endorses,
  certifies, or otherwise sanctions the Wanderer rule pack

---

### Requirement: Existing assessments are migrated from dictu to wand on store open

The schema migration runner SHALL convert every persisted
assessment row whose `framework = 'dictu'` to `framework =
'wand'` and rewrite every JSON-encoded `criterium_id` string
starting with `dictu.` to start with `wand.` instead. The
update SHALL run inside a single transaction so a partial
failure rolls back cleanly. This migration runs automatically
on `store.Open` once the new binary is in place.

#### Scenario: Pre-rename assessment becomes a wand assessment after open

- **GIVEN** a database containing one assessment row with
  `framework = 'dictu'` and a Rationale whose `criterium_id` is
  `dictu.juridisch.cert_issuer_eea`
- **WHEN** the new binary calls `store.Open` against that
  database
- **THEN** the migration runs to completion
- **AND** the row's `framework` column is `'wand'`
- **AND** the JSON-encoded Rationale's `criterium_id` is
  `wand.juridisch.cert_issuer_eea`

#### Scenario: Already-renamed row is left untouched

- **GIVEN** a database where the migration already ran (the
  assessments table contains rows with `framework = 'wand'` and
  `wand.*` criterium-IDs)
- **WHEN** `store.Open` runs again
- **THEN** the migration is not re-applied (its version is
  already in `schema_migrations`)
- **AND** no rows are modified

---

### Requirement: CLI accepts `dictu` as a deprecated alias for one release

`wanderer assess --framework dictu` SHALL continue to work for
exactly one release after this change ships, scoring the scan
against the `wand` rule pack and emitting one warning to
stderr that names the deprecation and the replacement
(`--framework wand`). The alias is removed in the release after.

#### Scenario: Legacy script still completes

- **GIVEN** an operator script invokes `wanderer assess <id>
  --framework dictu`
- **WHEN** the new binary runs the assessment
- **THEN** the persisted Assessment has `Framework: "wand"`
- **AND** stderr contains exactly one line beginning with
  `warning:` that names `--framework dictu` as deprecated and
  `--framework wand` as the replacement
- **AND** the exit code is 0 on success

#### Scenario: New invocation produces no warning

- **GIVEN** the same operator updates the script to
  `--framework wand`
- **WHEN** the binary runs
- **THEN** stderr contains no deprecation warning

---

### Requirement: Host-side findings produce a non-onbekend verdict

The assessor SHALL score agent-host scans (Targets with
`Kind=host`) on at least one rule per registered rule pack,
so a host scan produces a non-`onbekend` Assessment whenever
the agent's inspectors land their canonical Findings
(`inventory.packages.*`, `inventory.systemd.service`,
`egress.*` from the static scanner). Rules that target
perimeter ProbeIDs MUST continue to return `onbekend` on host
scans — they describe perimeter behaviour, not host
behaviour — but at least one host-shaped rule per pack must
fire on the agent's canonical findings.

#### Scenario: Agent scan produces a host-side verdict

- **GIVEN** an agent scan with `inventory.packages.rpm` and
  `inventory.systemd.service` findings
- **WHEN** the operator runs `wanderer assess <scan-id> --framework both`
- **THEN** the resulting Assessment has at least one
  dimension with a `soeverein`, `voldoende`, or `afhankelijk`
  score (not all `onbekend`)
- **AND** the host scan's verdict pill on `/ui/orgs/{slug}`
  renders that worst score

---

### Requirement: Host-rule soeverein verdicts cite negative evidence

A host-shaped rule SHALL cite at least one inspected Finding
ID in its `Evidence` slice and SHALL include the inspected
count in the Verdict text whenever it concludes soeverein.
This applies to rules reading `inventory.packages.*` or
`inventory.systemd.service`. The assessor engine forces
verdicts with empty Evidence back to `onbekend`, so this
keeps the soeverein call from being silently degraded.

#### Scenario: Clean host scores soeverein with evidence sample

- **GIVEN** an agent scan with 1790 `inventory.packages.rpm`
  Findings, none of which match the US-telemetry vendor list
- **WHEN** the assessor runs
  `wand.host.no_us_telemetry_packages`
- **THEN** the persisted Rationale has `Score: "soeverein"`,
  Verdict text containing `"inspected 1790 packages"`, and
  Evidence with 1..10 Finding IDs sampled from the inspected
  package Findings

### Requirement: Authoritative DNS jurisdiction is scored

The wand rule pack SHALL score the jurisdiction of a target's
authoritative nameservers by correlating the observed `dns.ns` hosts
with their `ip.asn` geo-lookups. When every located nameserver host
resolves to an EEA-registered AS the rule SHALL score soeverein; when
some do the rule SHALL score voldoende; when none do it SHALL score
afhankelijk; and when no nameserver could be located it SHALL score
onbekend. The verdict SHALL name the observed countries.

#### Scenario: EU-hosted nameservers score soeverein

- **GIVEN** a target whose `dns.ns` hosts all resolve to NL-registered AS
- **WHEN** the assessor runs the wand rule pack
- **THEN** `wand.juridisch.ns_vendor_jurisdiction` scores soeverein and
  names the country

#### Scenario: US-managed DNS scores afhankelijk

- **GIVEN** a target whose nameservers resolve to a US-registered AS
- **WHEN** the assessor runs
- **THEN** the rule scores afhankelijk and names the non-EEA jurisdiction

### Requirement: Passive HTTP exposure is scored

The wand rule pack SHALL score a target's passive HTTP exposure from
the observed security-header set and response banner. When HSTS is
absent it SHALL score afhankelijk; when HSTS is present but other
baseline headers are missing it SHALL score voldoende; when all
baseline headers are present it SHALL score soeverein; and when no
security-header observation exists it SHALL score onbekend. A Server /
X-Powered-By stack disclosure SHALL be named in the verdict. The rule
SHALL NOT perform any active or intrusive probing.

#### Scenario: Missing HSTS scores afhankelijk

- **GIVEN** a scan whose http.security_headers finding lists
  Strict-Transport-Security as missing
- **WHEN** the assessor runs
- **THEN** wand.operationeel.http_exposure scores afhankelijk and names
  the missing headers

