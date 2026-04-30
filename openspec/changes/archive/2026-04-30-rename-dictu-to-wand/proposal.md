## Why

The first rule pack in the assessor is named after **DICTU** —
the Dutch government's Dienst ICT Uitvoering, an agency under
the Ministry of Economic Affairs and Climate that publishes the
*Toetsingsinstrument Soevereiniteit Clouddiensten*. Wanderer's
rule pack was inspired by that publicly-available framework, but
Wanderer itself is a Conduction product, not a DICTU one, and we
have no endorsement from DICTU. Branding our output under the
DICTU name implies an affiliation we do not have, and is a real
legal and reputational risk for Conduction. This is the kind of
issue we want to clean up before any external rollout.

The rule semantics are correct and worth keeping. The label is
the problem. We rename the rule pack to **`wand`** — short for
**Wanderer-NL**, our own Conduction brand — and update every
rule ID, persisted Framework value, CLI flag value, code path,
documentation reference, and ADR accordingly. The new docs and
ADR explicitly credit DICTU's framework as the inspiration
without claiming endorsement.

## What Changes

- Rename the Go package `internal/assessor/dictu` to
  `internal/assessor/wand`.
- Rename every rule's `ID` from the `dictu.<dimension>.<short>`
  form to `wand.<dimension>.<short>` (the dimension labels and
  Match logic do not change — only the framework prefix).
- Change the `Framework` value persisted on every Assessment row
  from `"dictu"` to `"wand"`. **BREAKING** for any external
  consumer reading the JSON or the SQLite assessments table.
- Migrate existing assessment rows: a numbered up-only schema
  migration rewrites `framework` and the JSON-encoded
  `dimensions[].rationale[].criterium_id` strings on every row
  whose current Framework is `"dictu"`.
- Change the CLI flag values: `wanderer assess --framework
  dictu|eucsf|both` becomes `wanderer assess --framework
  wand|eucsf|both`. We keep `dictu` as a deprecated alias for
  one release cycle that prints a stderr deprecation notice.
- Update `internal/ui/registry.go::lookupRule` so the dashboard
  Analysis-page and top-concerns rendering resolve `wand.*`
  CriteriumIDs against the renamed registry.
- Update ADR-0009 (dual-framework assessor) with a note on the
  rename and credit the DICTU framework as the inspiration; add
  a fresh ADR-0011 documenting the legal motivation for the
  rename so a future contributor understands why we did this.
- Update CHANGELOG with `### Changed (breaking)` and `### Added`
  (alias) entries.
- Update `docs/assessor.md`, `docs/architecture.md`,
  `docs/operator.md`, `docs/tutorial.md`, README — every place
  the string `dictu` appears outside the deprecation alias.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `assessor`: the rule-pack identifier rolls forward from
  `dictu` to `wand`. Every persisted Assessment with
  `Framework: "dictu"` is migrated to `"wand"`; every rule ID
  inside those rows is rewritten correspondingly.
- `project-hygiene`: schema-migrations gain one new entry that
  performs the rename in a single up-only transaction, per the
  existing requirement that schema changes ship as numbered
  migrations.

## Impact

**Code**:

- `internal/assessor/dictu/` → renamed to
  `internal/assessor/wand/`. Every file inside it stays;
  `package dictu` becomes `package wand`; the public API
  (`DefaultRules`, helpers) keeps the same shape.
- Rule IDs string-replaced from `"dictu."` → `"wand."` (16+
  occurrences across rules.go and rules_test.go).
- `cmd/wanderer/assess.go` — flag parser accepts `wand`,
  `eucsf`, or `both`; legacy `dictu` is accepted with a
  deprecation warning to stderr (one line at startup).
- `internal/ui/registry.go::lookupRule` — switch arm for
  `dictu` becomes `wand`. The lookup keeps the deprecated
  alias for one release so old assessment rows that somehow
  bypassed the migration still render their rationale.

**Persisted state**:

- New schema migration (version 4) updates rows in the
  `assessments` table:
  1. `UPDATE assessments SET framework = 'wand' WHERE framework = 'dictu'`
  2. JSON-rewrite of every `dimensions[*].rationale[*].criterium_id`
     starting with `dictu.` → `wand.` on those same rows.
  Both happen inside one transaction; partial failure rolls back
  cleanly per the existing migration runner.

**APIs**: external consumers reading JSON (`GET /assessments/{id}`,
exporters CSV/JSONL) see the new framework label and rule IDs.
This is the breaking change called out in CHANGELOG.

**Dependencies**: none.

**Read-only contract**: preserved (UI changes are template +
helper only).

**DICTU dimensions informed**: every dimension. The rule
semantics are unchanged — only the label is.

**Passive/active boundary**: N/A — code-level rename.

**Compatibility**: the CLI alias for `dictu` and the UI
registry's deprecated lookup arm both go away in the next
release after this one ships. CHANGELOG carries the
deprecation notice so consumers know what they have one
release to do.
