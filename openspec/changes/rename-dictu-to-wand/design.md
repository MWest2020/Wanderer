## Context

`internal/assessor/dictu` houses Wanderer's first rule pack: 11
rules across `juridisch`, `operationeel`, `technologie`, and
`data_ai` dimensions, scoring a scan against criteria inspired by
DICTU's published *Toetsingsinstrument Soevereiniteit Clouddiensten*.
The rule semantics are independent work — Wanderer's rules read
the same probe outputs but are written, owned, and maintained by
Conduction. The DICTU label nevertheless implies endorsement.

The companion EU CSF (SEAL) rule pack at `internal/assessor/eucsf`
already establishes the multi-framework pattern: package per
pack, `Framework` field on every persisted Assessment,
`--framework <name>|both` flag selects which packs run.

The persisted shape (`models.Assessment.Framework` and
`models.Rationale.CriteriumID`) is the wire that crosses the
process / DB / consumer boundary. A rename has to migrate every
row and adjust every consumer.

## Goals / Non-Goals

**Goals:**

- Every output of Wanderer (CLI, JSON API, UI, persisted DB,
  exports) refers to the rule pack as `wand` rather than
  `dictu` after this change lands.
- Existing DB content (assessments produced before the rename)
  continues to render correctly; an operator who scored scans
  yesterday does not see "rule retired" placeholders today.
- One release of CLI compatibility for `--framework dictu` so
  scripts in the wild get a deprecation message rather than an
  error.
- The DICTU heritage is acknowledged in the docs as
  *inspiration*, not as endorsement — short paragraph in
  `docs/assessor.md` and ADR-0011.

**Non-Goals:**

- Changing rule semantics. The rules check the same evidence as
  before; only the label changes.
- A configurable framework label per deployment. The framework
  identifier is the audit-trail key; a per-deployment override
  defeats its purpose.
- A long-term `dictu` alias. The alias exists for one release as
  a courtesy to any external script; we do not commit to keeping
  it forever.

## Decisions

### Decision 1: New framework name is `wand`

`wand` is the Wanderer-NL contraction. Short, distinct from the
`eucsf.sov2.*` SEAL prefix already in use, owned by Conduction
because Wanderer is. Alternatives considered:

1. **`nlsoeverein`** — Dutch, descriptive, no agency claim. Passed
   over because four extra characters per rule ID add up across a
   document and provide no extra information.
2. **`sov-nl`** / **`sovnl`** — risk of confusion with
   `eucsf.sov2.*` rule IDs. Re-used prefixes in audit logs hurt
   skim-readability.
3. **`cloudsoev-nl`** — most descriptive. Verbose; the framework
   name appears in every Rationale row and prefix on every CLI
   invocation.

`wand` keeps the brand line clean: Wanderer is the tool, `wand` is
the in-tool rule pack name, `eucsf` is the EU framework's rule
pack name. A reader who sees `wand.juridisch.cert_issuer_eea`
in a log line knows exactly which side of the multi-framework
boundary they are on.

### Decision 2: Schema migration rewrites both columns and rule IDs

Existing assessments are persisted as:

```sql
INSERT INTO assessments (id, scan_id, framework, dimensions, ...)
VALUES (..., 'dictu', '[{"dimension":"juridisch","score":"afhankelijk",...,"rationale":[{"criterium_id":"dictu.juridisch.cert_issuer_eea",...}]}]', ...);
```

The rename touches both the `framework` column **and** the
JSON-encoded `criterium_id` strings inside `dimensions`. The
migration:

```sql
UPDATE assessments
SET framework  = 'wand',
    dimensions = REPLACE(dimensions, '"dictu.', '"wand.')
WHERE framework = 'dictu';
```

The `REPLACE` is safe because:
- `criterium_id` values are the only places `"dictu.` appears
  inside the JSON blob (the prefix-with-quote is the marker;
  `dictu.juridisch.x` cannot appear as a substring of any other
  rule ID).
- SQLite's `REPLACE` is byte-level on TEXT, deterministic, and
  runs inside the same transaction as the column update.

We verify with a small Go test that:
- An assessment row with `framework='dictu'` and rationale-IDs
  starting with `dictu.` round-trips to `framework='wand'` with
  rationale-IDs starting with `wand.`.
- A row that already has `framework='wand'` is left untouched.

### Decision 3: CLI alias `dictu` warns once, keeps working for one release

`wanderer assess --framework dictu` continues to work but emits
one stderr line:

```
warning: --framework dictu is deprecated and will be removed in
the next release; use --framework wand instead. The DICTU
inspiration is documented in docs/assessor.md.
```

This avoids breaking any operator script during the rollout
window. The alias is removed in the next release. ADR-0011
documents the rationale.

### Decision 4: UI lookup keeps a one-release alias arm

`internal/ui/registry.go::lookupRule` accepts both `dictu` and
`wand` as the framework key for one release, both resolving
against `wand.DefaultRules()`. This catches any old DB row that
was somehow not migrated (a copy-pasted SQL dump, a forensic
restore from before this change). Same one-release window as the
CLI alias.

### Decision 5: ADR-0011 documents the rename motivation

`docs/decisions/0011-rename-dictu-to-wand.md` explains the legal
and reputational reasoning so a future contributor reading the
codebase understands "why is the rule pack called `wand` when
the docs talk about DICTU's framework as the inspiration?" The
ADR also names DICTU as the source of the underlying framework
concept and links to the public DICTU Toetsingsinstrument page.

ADR-0009 (the dual-framework decision) gets a one-paragraph
addendum noting that the `dictu` rule-pack name was renamed to
`wand` per ADR-0011; the original architectural decision (two
parallel rule packs, shared Finding contract, `Framework` field
on Assessment) is unchanged.

## Risks / Trade-offs

[Risk] An external consumer (an exporter, a downstream
dashboard, a SIEM rule) reads `framework: "dictu"` and breaks
when it sees `framework: "wand"`. → Mitigation: CHANGELOG
under `### Changed (breaking)` calls out the exact field
rename. The migration is single-pass on `Open()`, so the
moment the new binary runs the data is already converted; no
window where the API returns mixed values.

[Risk] Someone restores an old database from backup after this
change ships and gets `dictu`-prefixed rows back. → Mitigation:
the migration runner is idempotent — every `Open()` checks
`schema_migrations` and re-applies missing migrations in order.
A restore from before migration 4 will land on a binary that
runs migration 4 immediately, converting the rows on first open.

[Risk] The `REPLACE`-based JSON rewrite touches a future field
that happens to contain the literal `"dictu.` substring. →
Mitigation: today the only field carrying that substring is
`criterium_id`. We pin a unit test to that invariant; if a future
schema change adds another field with the prefix, the test will
flag the conflict before the change merges. The migration could
be hardened by JSON-decoding each row, walking the structure, and
re-encoding, at the cost of introducing CGo / heavier code into
the migration path. We accept the byte-level REPLACE for the
current shape and revisit if the shape ever grows.

[Risk] Operators copy-paste the legacy `--framework dictu`
indefinitely and never see the deprecation. → Mitigation: one
release of warning, then removal. The CHANGELOG entry under
`### Removed` in the next release is the second signal.

**Clever valkuil:**

1. **Skipping the migration and letting `lookupRule` translate at
   render time forever.** Tempting because it is the smallest
   diff. Wrong because the database becomes the audit trail of
   what was scored under what framework label; carrying a
   deprecated label in production data forever is exactly the
   kind of debt that triggers the rename in the first place.
2. **Per-deployment configurable framework label.** Lets every
   operator pick their own. Wrong because the framework
   identifier is the cross-installation key — a rule ID has to
   mean the same thing on every Wanderer deployment for findings
   to be comparable.
3. **Renaming the package only and leaving rule IDs as
   `dictu.*`.** Hides the rename from the consumer. Wrong because
   the rule ID is the consumer-facing token (it appears in every
   Rationale, every report, every dashboard concern row).

**External systems & failure modes:**

- SQLite `REPLACE` — pure-Go modernc.org/sqlite, deterministic,
  no CGo. The migration runs inside one transaction so a partial
  failure rolls back. Verified by a unit test against an
  in-memory store seeded with a `dictu`-labelled row.
- The CLI's `--framework` parser — handled by `cmd/wanderer/assess.go`;
  the alias arm is a single switch case.
- The UI registry alias — already covered by
  `lookupRule`'s switch; the alias arm is one extra case for the
  rollout window.
