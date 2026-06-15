# Delta for web-ui

> Implemented 2026-06-15; ready to merge into the canonical web-ui spec.

## ADDED Requirements

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
