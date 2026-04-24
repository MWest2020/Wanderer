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

