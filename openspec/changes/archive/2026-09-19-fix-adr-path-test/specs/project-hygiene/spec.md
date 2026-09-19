## MODIFIED Requirements

### Requirement: Architecture decisions are recorded

The project SHALL store Architecture Decision Records under
`docs/explanation/adr/` using the filename pattern `NNNN-short-title.md`,
and any decision that constrains future changes (dependency choice,
data-model contract, passive-vs-active boundary) SHALL produce one.

#### Scenario: New cross-cutting dependency

- **Given** a proposed change that introduces a new top-level Go
  dependency
- **When** the change is designed
- **Then** an ADR exists in `docs/explanation/adr/` explaining why the
  dependency was chosen over alternatives
- **And** the change's `design.md` links to the ADR

#### Scenario: Tactical fix with no lasting impact

- **Given** a bug fix that does not constrain future design
- **When** the fix lands
- **Then** no ADR is required

### Requirement: ADR-0011 records the dictu→wand rename motivation

The numbered ADR set under `docs/explanation/adr/` SHALL include
ADR-0011 covering the rename of the first-party rule pack from
`dictu` to `wand`. The ADR SHALL explain the legal /
reputational concern (DICTU is a Dutch government agency, not a
Conduction product), credit the DICTU *Toetsingsinstrument
Soevereiniteit Clouddiensten* as the public source of
inspiration, and document the migration path (one-release CLI
alias plus schema migration). ADR-0009 (dual-framework
assessor) SHALL be updated with a one-paragraph addendum noting
the rename.

#### Scenario: Future contributor reads the rename rationale

- **GIVEN** a contributor opens `docs/explanation/adr/`
- **WHEN** they look for the source of the `wand` name
- **THEN** ADR-0011 documents the rename's motivation,
  references the DICTU framework as the inspiration source, and
  links to the migration that handled existing data

#### Scenario: ADR-0009 acknowledges the rename

- **GIVEN** the same contributor reads ADR-0009
- **WHEN** they reach the section that named the first rule pack
- **THEN** an addendum at the end of ADR-0009 points at ADR-0011
  for the rename and notes the new identifier `wand`

### Requirement: Every UI-touching ADR has a Playwright spec

The project SHALL maintain one Playwright spec file per ADR
that describes UI behaviour. A doc-lint check SHALL run as part
of `go test ./...` to grep `docs/explanation/adr/` for `## UI
surface` sections and confirm a matching spec exists at
`tests/playwright/specs/<adr-slug>.spec.ts`. ADRs without UI
claims need no spec.

#### Scenario: ADR without spec fails the lint

- **Given** an ADR `docs/explanation/adr/0042-some-ui-change.md`
  contains a `## UI surface` section
- **And** no file exists at
  `tests/playwright/specs/0042-some-ui-change.spec.ts`
- **When** the doc-lint test runs
- **Then** the test fails and names the missing spec

#### Scenario: ADR with spec passes the lint

- **Given** the ADR's spec file exists
- **When** the doc-lint test runs
- **Then** the test passes

#### Scenario: Lint reads the contract location

- **Given** the ADRs live under `docs/explanation/adr/` and
  `docs/decisions/` does not exist
- **When** the doc-lint test runs
- **Then** it reads `docs/explanation/adr/` and does not fail on
  the missing old directory
