## ADDED Requirements

### Requirement: Drift compares perimeter scans, not imports

The drift engine SHALL compare a scan with the previous scan of the same
target that is not an import scan. An import scan holds a different kind of
evidence (Internet.nl's verdicts, not Wanderer's probes); diffing across it
would report every perimeter Finding as added and every imported one as
removed.

#### Scenario: Import between two perimeter scans

- **GIVEN** a perimeter scan, then an import scan, then a perimeter scan with
  identical perimeter Findings, all for the same target
- **WHEN** the drift engine compares the last scan
- **THEN** exactly one Finding with `ProbeID: drift.no_changes` is persisted
- **AND** no other `drift.*` Findings are persisted

#### Scenario: Only an import before the first perimeter scan

- **GIVEN** a target whose only earlier scan is an import scan
- **WHEN** its first perimeter scan completes
- **THEN** exactly one Finding with `ProbeID: drift.baseline_established` is
  persisted
