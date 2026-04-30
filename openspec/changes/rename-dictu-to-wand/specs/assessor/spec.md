## ADDED Requirements

### Requirement: First-party rule pack is named `wand`, not `dictu`

The Conduction-owned rule pack SHALL be identified as `wand`
(Wanderer-NL) in every output Wanderer produces: the persisted
`Assessment.Framework` value, the rule IDs (under the
`wand.<dimension>.<short>` shape), the CLI flag value
(`--framework wand|eucsf|both`), and every documentation or UI
surface that names the framework. The DICTU
*Toetsingsinstrument Soevereiniteit Clouddiensten* SHALL be
credited in the assessor docs and ADR-0011 as the public
framework that inspired the rule set; the implementation,
ownership, and label are Conduction's.

#### Scenario: New assessment carries the wand framework label

- **GIVEN** the renamed rule pack is in production
- **WHEN** an operator runs `wanderer assess <scan-id> --framework wand`
- **THEN** the persisted `Assessment.Framework` is `"wand"`
- **AND** every persisted Rationale's `CriteriumID` starts with
  `wand.`

#### Scenario: Output documents the inspiration without claiming endorsement

- **GIVEN** a contributor reads `docs/assessor.md` after the rename
- **WHEN** they look for the relationship to DICTU
- **THEN** the doc explicitly names DICTU's *Toetsingsinstrument
  Soevereiniteit Clouddiensten* as the inspiration for the rule
  set
- **AND** the doc does not state or imply that DICTU endorses,
  certifies, or otherwise sanctions the Wanderer rule pack

---

### Requirement: Existing assessments are migrated from dictu to wand on store open

The schema migration runner SHALL convert every persisted
assessment row whose `framework = 'dictu'` to `framework =
'wand'` and rewrite every JSON-encoded `criterium_id` string
starting with `dictu.` to start with `wand.` instead. The
update SHALL run inside a single transaction so a partial
failure rolls back cleanly. This migration runs automatically
on `store.Open` once the new binary is in place.

#### Scenario: Pre-rename assessment becomes a wand assessment after open

- **GIVEN** a database containing one assessment row with
  `framework = 'dictu'` and a Rationale whose `criterium_id` is
  `dictu.juridisch.cert_issuer_eea`
- **WHEN** the new binary calls `store.Open` against that
  database
- **THEN** the migration runs to completion
- **AND** the row's `framework` column is `'wand'`
- **AND** the JSON-encoded Rationale's `criterium_id` is
  `wand.juridisch.cert_issuer_eea`

#### Scenario: Already-renamed row is left untouched

- **GIVEN** a database where the migration already ran (the
  assessments table contains rows with `framework = 'wand'` and
  `wand.*` criterium-IDs)
- **WHEN** `store.Open` runs again
- **THEN** the migration is not re-applied (its version is
  already in `schema_migrations`)
- **AND** no rows are modified

---

### Requirement: CLI accepts `dictu` as a deprecated alias for one release

`wanderer assess --framework dictu` SHALL continue to work for
exactly one release after this change ships, scoring the scan
against the `wand` rule pack and emitting one warning to
stderr that names the deprecation and the replacement
(`--framework wand`). The alias is removed in the release after.

#### Scenario: Legacy script still completes

- **GIVEN** an operator script invokes `wanderer assess <id>
  --framework dictu`
- **WHEN** the new binary runs the assessment
- **THEN** the persisted Assessment has `Framework: "wand"`
- **AND** stderr contains exactly one line beginning with
  `warning:` that names `--framework dictu` as deprecated and
  `--framework wand` as the replacement
- **AND** the exit code is 0 on success

#### Scenario: New invocation produces no warning

- **GIVEN** the same operator updates the script to
  `--framework wand`
- **WHEN** the binary runs
- **THEN** stderr contains no deprecation warning
