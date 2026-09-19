# Delta for scanner

> Draft. One new import path following the Amass precedent
> (offline artifact in, findings out; malformed input degrades, never
> blocks).

## ADDED Requirements

### Requirement: Wanderer imports netnl findings files

`wanderer import internetnl <file>` SHALL parse a
`netnl-findings/v1` document, match each domain to a known target,
and persist `internetnl.web.*` / `internetnl.mail.*` findings
(verdict, category, measured_at, report URL, request ID) under a
scan of kind `import`. Unknown domains SHALL be logged at WARN and
skipped; malformed entries SHALL be logged at WARN and skipped; a
schema version other than v1 SHALL abort the import naming the
expected version. Re-importing an identical file SHALL be idempotent.

#### Scenario: Batch file for the fleet

- **GIVEN** a v1 findings file carrying web+mail results for three
  known targets and one unknown domain
- **WHEN** the operator runs `wanderer import internetnl batch.json`
- **THEN** findings are persisted for the three targets under an
  import scan
- **AND** the unknown domain is warned about and skipped
- **AND** the command exits 0

#### Scenario: Schema version mismatch

- **GIVEN** a file declaring `netnl-findings/v2`
- **WHEN** the operator runs the import
- **THEN** the import aborts before persisting anything and the error
  names the supported version

#### Scenario: Truncated file

- **GIVEN** a v1 file whose final entry is malformed JSON
- **WHEN** the operator runs the import
- **THEN** the valid entries are persisted, the malformed entry is
  warned about, and the partial import is recorded as such

---

### Requirement: Imports and perimeter scans do not erase each other

Imported `internetnl.*` findings SHALL persist independently of
perimeter scans: a later perimeter scan SHALL NOT remove or supersede
imported findings, and a later import SHALL NOT touch perimeter
findings. The assessor reads the newest of each.

#### Scenario: Perimeter scan after import

- **GIVEN** a target with imported standards findings from yesterday
- **WHEN** a perimeter scan completes today
- **THEN** the assessment combines today's perimeter findings with
  yesterday's imported standards findings

#### Scenario: Fresh import after perimeter scan

- **GIVEN** the same target
- **WHEN** a new findings file is imported
- **THEN** the standards rules score from the new import and the
  perimeter findings are unchanged
