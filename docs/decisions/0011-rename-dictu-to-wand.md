# 0011 — Rename the first-party rule pack from `dictu` to `wand`

**Status:** Accepted, applied 2026-04-30.

**Context.** The first rule pack in Wanderer's assessor was named
`dictu`, after the Dutch government's *Dienst ICT Uitvoering* —
the IT-execution agency under the Ministry of Economic Affairs
and Climate that publishes the *Toetsingsinstrument Soevereiniteit
Clouddiensten*. Wanderer's rule semantics were inspired by that
publicly-available framework, but Wanderer is a Conduction
product and we have no DICTU endorsement. The `dictu` label —
showing up on every persisted Assessment, every CLI invocation,
every rule ID, every UI screen — implied an affiliation we do
not have. That is a real legal and reputational risk, and the
right time to clean it up is before any external rollout.

**Decision.** Rename the rule pack to `wand` (Wanderer-NL).

- Go package `internal/assessor/dictu` → `internal/assessor/wand`.
- Rule IDs `dictu.<dim>.<short>` → `wand.<dim>.<short>` (the
  dimension labels and the rule semantics are unchanged).
- Persisted `Assessment.Framework` value `"dictu"` → `"wand"`.
- CLI flag value `--framework dictu` → `--framework wand` (with
  `dictu` accepted as a deprecated alias for one release that
  prints a stderr warning).
- UI registry's `lookupRule` accepts both `wand` and `dictu` as
  the framework key for one release; both resolve against
  `wand.DefaultRules()`.
- Schema migration version 4 rewrites every persisted assessment
  row whose `framework='dictu'` to `'wand'`, and rewrites every
  JSON-encoded `criterium_id` starting with `dictu.` to start
  with `wand.` instead — both inside one transaction.

**The `wand` name.** Short for "Wanderer-NL". Conduction-owned,
no agency claim, distinct from the `eucsf.sov2.*` SEAL prefix
that already exists. Alternatives considered:

- **`nlsoeverein`** — Dutch, descriptive, but four extra
  characters per rule ID across an entire assessment report.
- **`sov-nl`** / **`sovnl`** — risk of confusion with
  `eucsf.sov2.*` rule IDs.
- **`cloudsoev-nl`** — most descriptive, verbose where every
  rule ID carries the prefix.

`wand` keeps each rule ID compact and unambiguous: a reader who
sees `wand.juridisch.cert_issuer_eea` knows exactly which side
of the multi-framework boundary they are on.

**The DICTU framework as inspiration, credited.** The rules in
the wand pack were originally derived from DICTU's *Toetsingsinstrument
Soevereiniteit Clouddiensten*. We credit the framework as the
public source of inspiration in `docs/assessor.md` and in this
ADR; we do not claim or imply DICTU endorsement of Wanderer or
the wand pack. The independent implementation, the rule
authoring, and the ongoing maintenance are Conduction's.

Reference: <https://www.dictu.nl/>

**Migration safety.** SQLite's `REPLACE(dimensions, '"dictu.',
'"wand.')` is byte-level on TEXT, deterministic, and runs inside
the same transaction as the column update. The substring
`"dictu.` (leading double-quote, trailing dot) only appears as
the prefix of a JSON-encoded `criterium_id` value — verified by
the unit test in `internal/store/store_test.go::TestMigration_RenameDictuToWand`.
A future schema change that adds another field carrying the
literal substring would have to update the migration to be more
selective; the test pins the invariant.

**Deprecation window.** The CLI alias and the UI registry alias
both go away in the next release. The CHANGELOG entry under
`### Removed` in that release is the second signal to operators
who somehow still pass `--framework dictu`.

**Why the wholesale rebrand instead of a translation layer.** A
translation layer at render time keeps the database labelled
`dictu` indefinitely. The database is the audit trail of what
was scored under what framework label; carrying a deprecated
agency-name label in production data forever is exactly the
debt that triggered this rename.

**Trade-offs.**

- External consumers reading the JSON API or the SQLite
  `assessments` table see `framework: "wand"` after this change.
  CHANGELOG calls it out as a breaking change to the persisted
  shape.
- Operator scripts using `--framework dictu` keep working for
  one release with a deprecation warning. Scripts that ignore
  stderr will only break in the next release.
- The DICTU credit lives in the docs and the ADR, not in the
  rule pack identifier. A reader who only looks at the JSON
  output sees `wand` with no jurisdictional hint; the docs make
  the heritage discoverable.

**References.**

- ADR-0009 — Dual-framework assessor (the architectural decision
  to ship two parallel rule packs sharing the Finding contract;
  see addendum at the end of that ADR pointing here).
- Original proposal:
  `openspec/changes/archive/2026-04-30-rename-dictu-to-wand/proposal.md`.
- Design:
  `openspec/changes/archive/2026-04-30-rename-dictu-to-wand/design.md`.
- Spec deltas applied to `openspec/specs/assessor/spec.md` and
  `openspec/specs/project-hygiene/spec.md`.
