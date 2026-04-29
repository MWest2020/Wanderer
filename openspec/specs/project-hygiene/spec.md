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

