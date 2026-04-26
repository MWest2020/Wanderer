# scheduling Specification

## Purpose
TBD - created by archiving change add-scheduling. Update Purpose after archive.
## Requirements
### Requirement: Schedules execute at their cron time

A valid cron entry in `wanderer-schedules.yaml` SHALL trigger a scan
of the configured target at each matching cron tick, using the same
scanner pipeline as an ad-hoc scan.

#### Scenario: Simple cron fires

- **Given** a schedule with cron `*/5 * * * *` for target
  `example.nl`, and a running `wanderer serve`
- **When** the clock advances past a 5-minute boundary
- **Then** a new Scan row is persisted for `example.nl`
- **And** the Scan's Findings appear in the store within one minute

#### Scenario: Invalid cron expression rejects at startup

- **Given** a schedules file with `cron: "foo bar"` (invalid)
- **When** `wanderer serve` starts
- **Then** the process exits non-zero
- **And** stderr names the offending schedule entry

#### Scenario: SIGHUP reloads schedules

- **Given** a running `wanderer serve` with one schedule
- **When** the operator edits the schedules file to add a second
  schedule and sends SIGHUP to the process
- **Then** the second schedule's next tick triggers a scan
- **And** the first schedule continues firing on its existing cadence

---

### Requirement: First scan yields a baseline, never spurious drift

The drift engine SHALL NOT emit change Findings when comparing a new
scan against *no* previous scan.

#### Scenario: Baseline

- **Given** a target with no prior scans in the store
- **When** a scheduled scan completes
- **Then** exactly one Finding with
  `ProbeID: drift.baseline_established` is persisted
- **And** no other `drift.*` Findings are persisted

#### Scenario: No changes

- **Given** two consecutive scans for the same target producing
  identical Findings (ignoring IDs + timestamps)
- **When** the drift engine compares them
- **Then** exactly one Finding with `ProbeID: drift.no_changes` is
  persisted
- **And** no other `drift.*` Findings are persisted

---

### Requirement: Drift Findings are first-class

Every drift Finding SHALL have `source_modus: drift`, reference the
two scan IDs it was derived from, and carry a `DimensionHint` copied
from the rule that fired.

#### Scenario: MX set changed

- **Given** scan A with MX hosts `{mail1, mail2}` and scan B with
  `{mail1, mail3}`
- **When** drift runs against B vs A
- **Then** a Finding with `ProbeID: drift.dns.mx_set_changed` is
  persisted
- **And** Attributes contain `added: ["mail3"]`, `removed: ["mail2"]`
- **And** `DimensionHint: data_ai`
- **And** `Attributes.prev_scan_id == "<A.ID>"` and
  `Attributes.curr_scan_id == "<B.ID>"`

#### Scenario: Issuer changed is finding severity

- **Given** a TLS issuer CN change between two scans
- **When** drift runs
- **Then** the resulting Finding has `Severity: finding`

---

### Requirement: `wanderer diff` runs without mutating the store

The `wanderer diff <scan-a> <scan-b>` command SHALL compute and print
the drift Findings it would produce, but SHALL NOT persist them.

#### Scenario: Read-only diff

- **Given** two existing scans
- **When** `wanderer diff <a> <b>` runs
- **Then** stdout contains the drift Findings rendered as markdown
- **And** the `findings` table row count is unchanged afterwards

#### Scenario: Missing scan

- **Given** a scan ID that does not exist
- **When** `wanderer diff <missing> <existing>` runs
- **Then** the command exits non-zero
- **And** stderr names the missing scan ID

