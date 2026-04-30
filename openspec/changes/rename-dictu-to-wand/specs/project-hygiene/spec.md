## ADDED Requirements

### Requirement: ADR-0011 records the dictu→wand rename motivation

The numbered ADR set under `docs/decisions/` SHALL include
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

- **GIVEN** a contributor opens `docs/decisions/`
- **WHEN** they look for the source of the `wand` name
- **THEN** ADR-0011 documents the rename's motivation,
  references the DICTU framework as the inspiration source, and
  links to the migration that handled existing data

#### Scenario: ADR-0009 acknowledges the rename

- **GIVEN** the same contributor reads ADR-0009
- **WHEN** they reach the section that named the first rule pack
- **THEN** an addendum at the end of ADR-0009 points at ADR-0011
  for the rename and notes the new identifier `wand`
