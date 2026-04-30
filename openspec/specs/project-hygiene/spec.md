# project-hygiene Specification

## Purpose
TBD - created by archiving change add-maintainability-baseline. Update Purpose after archive.
## Requirements
### Requirement: CHANGELOG is kept current

The project SHALL maintain a `CHANGELOG.md` at the repository root
in Keep-a-Changelog style, and every merged change SHALL add a dated
entry describing what changed, why, and which files were touched.

#### Scenario: Feature change merged

- **Given** a change like `add-assessor` that introduces new user-visible
  behaviour
- **When** its implementation tasks are archived
- **Then** `CHANGELOG.md` contains a dated `### Added` entry naming the
  capability and the OpenSpec change id

#### Scenario: Housekeeping fix merged

- **Given** a lint-only or typo fix
- **When** the fix lands
- **Then** `CHANGELOG.md` contains a dated `### Fixed` entry referencing
  the affected area (even if no OpenSpec change exists)

---

### Requirement: Architecture decisions are recorded

The project SHALL store Architecture Decision Records under
`docs/decisions/` using the filename pattern `NNNN-short-title.md`,
and any decision that constrains future changes (dependency choice,
data-model contract, passive-vs-active boundary) SHALL produce one.

#### Scenario: New cross-cutting dependency

- **Given** a proposed change that introduces a new top-level Go
  dependency
- **When** the change is designed
- **Then** an ADR exists in `docs/decisions/` explaining why the
  dependency was chosen over alternatives
- **And** the change's `design.md` links to the ADR

#### Scenario: Tactical fix with no lasting impact

- **Given** a bug fix that does not constrain future design
- **When** the fix lands
- **Then** no ADR is required

---

### Requirement: Public API stability is declared

The project SHALL treat `pkg/models` as the stable public contract
and SHALL treat everything under `internal/` as free to change
without notice; breaking changes to `pkg/models` SHALL ship with a
CHANGELOG entry under a `### Changed (breaking)` heading and a
corresponding ADR.

#### Scenario: Field renamed in pkg/models

- **Given** a change that renames a field on `pkg/models.Finding`
- **When** the change lands
- **Then** the CHANGELOG contains a `### Changed (breaking)` entry
- **And** an ADR under `docs/decisions/` records the rationale

#### Scenario: Refactor inside internal/scanner

- **Given** a change that reshapes functions in `internal/scanner`
  without touching `pkg/models`
- **When** the change lands
- **Then** no `(breaking)` CHANGELOG entry is required

---

### Requirement: Every new package carries tests and a package comment

The project SHALL require that every new Go package ships with at
least one `_test.go` file and a package-level doc comment of 2–4
sentences describing its intent; `internal/*` SHALL target ≥70% line
coverage and `pkg/models` SHALL target ≥90%.

#### Scenario: New probe package added

- **Given** a change that introduces `internal/probe/egress`
- **When** the change lands
- **Then** the package has a `package egress` doc comment of 2–4
  sentences
- **And** the package ships with a `_test.go` file covering the public
  surface
- **And** coverage for the package is at least 70%

#### Scenario: Refactor of existing package

- **Given** a change that only moves code within `internal/scanner`
- **When** the change lands
- **Then** the existing package doc comment is preserved or updated,
  not removed

---

### Requirement: Non-trivial changes use OpenSpec before implementing

The project SHALL require an OpenSpec change proposal for any work
that adds a new command, probe, data type, or external interface, and
SHALL NOT require one for lint fixes, typo fixes, or internal
refactors that preserve behaviour and public API.

#### Scenario: New probe proposed

- **Given** a plan to add a fifth probe
- **When** work begins
- **Then** a directory `openspec/changes/<name>/` exists with
  proposal, design, tasks, and spec deltas
- **And** `openspec change validate <name> --strict` passes before
  implementation starts

#### Scenario: Typo in an error message

- **Given** a typo found in a log message
- **When** the fix is made directly
- **Then** no OpenSpec change is required
- **And** a CHANGELOG `### Fixed` entry still records the change

---

### Requirement: Schema changes ship as numbered up-only migrations

The store SHALL track schema state in a `schema_migrations` table
holding `(version, name, applied_at)` rows, apply migrations
strictly in numeric order in a single transaction per migration,
and refuse the string-matched `ALTER TABLE ... DEFAULT ...` /
`duplicate column name` tolerance pattern for new schema changes,
so an auditor can read one table to know which schema version is
in production.

#### Scenario: Fresh database applies every migration

- **Given** an empty SQLite file
- **When** `store.Open` runs
- **Then** every entry in the `migrations` slice runs in order
- **And** `schema_migrations` lists every applied version with its
  `applied_at` timestamp

#### Scenario: Database at version N applies only N+1..M

- **Given** a database whose `schema_migrations` table contains
  versions 1, 2, 3
- **And** the source ships migrations 1..5
- **When** `store.Open` runs
- **Then** only migrations 4 and 5 execute
- **And** migrations 1..3 are not re-run

#### Scenario: Failed migration rolls back

- **Given** migration 7 contains an invalid SQL statement
- **When** `store.Open` runs against a database at version 6
- **Then** the transaction for migration 7 rolls back
- **And** `schema_migrations` continues to show version 6 as the
  highest applied version
- **And** `store.Open` returns the underlying SQL error

---

### Requirement: Architecture page covers every shipped modus

The top-level `docs/architecture.md` SHALL describe every
operating modus the binary supports today (perimeter, inventory,
egress) and SHALL link the per-capability documentation page for
each, so a new contributor reading the document gets an accurate
mental model of where each ProbeID prefix originates.

#### Scenario: Modus coverage

- **Given** the current main branch with all capabilities landed
- **When** a contributor reads `docs/architecture.md`
- **Then** the document references the perimeter modus
  (`wanderer scan` / `serve`), the inventory modus
  (`wanderer agent` inspectors), and the egress modus
  (`wanderer agent` egress probe)
- **And** every per-capability doc page (`assessor.md`,
  `agent.md`, `egress.md`, `mcp.md`, `scheduling.md`,
  `drift.md`, `exporters.md`) is linked at least once

#### Scenario: How-to-add-a-probe stays current

- **Given** the architecture page's "How to add a perimeter
  probe" section
- **When** a contributor follows the steps against the current
  `internal/probe` layout
- **Then** every file path the section names exists
- **And** every interface name the section names matches the
  symbol in the codebase

---

### Requirement: Operator documentation explains GeoLite2 setup

`docs/operator.md` SHALL include a "GeoLite2 setup" section
covering MaxMind license-key acquisition, the recommended
periodic-update mechanism (`geoipupdate` via systemd timer or
crontab), the file paths the agent expects, and how to silence
the startup warning when GeoLite2 is intentionally absent.
`docs/architecture.md` and `docs/tutorial.md` SHALL link into
the section.

#### Scenario: Operator opens the docs

- **GIVEN** a contributor reading `docs/operator.md` for the
  first time
- **WHEN** they search for "GeoLite2"
- **THEN** they find a top-level section explaining the setup
- **AND** the section names both `--geoip` / `WANDERER_GEOIP_ASN`
  (required) and `--geoip-country` / `WANDERER_GEOIP_COUNTRY`
  (optional)
- **AND** the section names `WANDERER_GEOIP_OPTIONAL=1` /
  `--no-geoip` as the opt-out for hosts that consciously run
  without ASN annotation

#### Scenario: Architecture doc cross-references the setup

- **GIVEN** the rewritten `docs/architecture.md`
- **WHEN** a reader scrolls to the "External systems and their
  failure modes" table
- **THEN** the GeoLite2 row links to the operator doc's
  "GeoLite2 setup" section

---

### Requirement: CLI warns once at startup when GeoLite2 is missing

`wanderer scan` and `wanderer serve` SHALL emit one
warning-level log line to stderr at startup when no
`--geoip` value is set and the `--no-geoip` /
`WANDERER_GEOIP_OPTIONAL=1` opt-out is not present. The warning
SHALL name the `--geoip` flag, name the operator-doc URL or
path, and state explicitly that the scan continues with reduced
assessment coverage rather than failing.

#### Scenario: Default invocation surfaces the warning

- **GIVEN** an operator runs `wanderer scan example.nl` on a
  host with no `--geoip`, no `WANDERER_GEOIP_ASN`, and no
  `WANDERER_GEOIP_OPTIONAL`
- **WHEN** the binary starts
- **THEN** stderr contains exactly one warning line beginning
  with `warning:`
- **AND** the line names `--geoip` and points at `docs/operator.md`
- **AND** the scan still completes (exit code 0 on success)

#### Scenario: Opt-out silences the warning

- **GIVEN** the same operator with `--no-geoip` set on the
  command line (or `WANDERER_GEOIP_OPTIONAL=1` in the env)
- **WHEN** the binary starts
- **THEN** stderr contains no warning about GeoLite2
- **AND** the scan still completes
- **AND** the per-scan `ip.unavailable` Finding is unchanged

#### Scenario: Configured GeoLite2 produces no warning

- **GIVEN** an operator with `--geoip /var/lib/wanderer/GeoLite2-ASN.mmdb` set and the file is readable
- **WHEN** the binary starts
- **THEN** stderr contains no GeoLite2 warning
- **AND** the IP probe runs in its populated mode

---

### Requirement: Test suite has a stub mmdb path

The test suite SHALL ship a way to exercise the
`internal/probe/ip` populated-but-empty path without a real
MaxMind license. The mechanism SHALL be either a
`scripts/geoip-stub.sh` that produces a deterministic minimal
mmdb file, or an equivalent Go test helper, with a single
command/function call documented in `docs/operator.md`.

#### Scenario: Stub builder produces a valid empty mmdb

- **GIVEN** `scripts/geoip-stub.sh` (or the equivalent test
  helper)
- **WHEN** a test invokes it pointing at a temp directory
- **THEN** the resulting file opens cleanly via the
  `oschwald/maxminddb-golang` reader
- **AND** any IP lookup against it returns "not found" without
  error

---

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

