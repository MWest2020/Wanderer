# Delta for web-ui

> Implemented 2026-06-15; ready to merge into the canonical web-ui spec.

## ADDED Requirements

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
