# Delta for web-ui

## ADDED Requirements

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
