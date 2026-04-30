# 0009. Dual framework assessor: DICTU + EU CSF / SEAL

- Status: accepted
- Date: 2026-04-28

## Context

Wanderer's first assessor pack — DICTU — answers the Dutch
government sovereignty toets in its native vocabulary: five
dimensions (`juridisch`, `operationeel`, `technologie`, `data_ai`,
`mens`) and four levels (`onbekend`, `afhankelijk`, `gedeeld`,
`soeverein`). Reviewers from the EU side ask the same evidence
questions but want the answer in the EU Cybersecurity Framework's
SEAL vocabulary: five levels SEAL0–SEAL4. The two stakeholder groups
overlap on what they look at — leaf certificate jurisdiction, MX
hosts, third-party hosts, hyperscaler dependencies — but disagree on
the answer shape.

Two paths considered:

1. Pick one framework and translate at read time. The translator
   becomes a third source of truth that nobody owns and goes stale
   the first time either framework's vocabulary shifts.
2. Ship two rule packs against the same Findings, persist both, let
   operators select per assess invocation.

## Decision

We ship two assessor packs side-by-side: `internal/assessor/dictu`
and `internal/assessor/eucsf`. They share no rule code; both consume
the shared `[]models.Finding` slice and emit `models.Assessment`
records tagged with `Framework`. `wanderer assess --framework
dictu|eucsf|both` selects which pack(s) run and persists one
record per framework.

Adding a third pack means adding a new package, not modifying
either existing pack. The translator between frameworks does not
exist and SHALL NOT be added — translation, if it ever becomes
necessary, happens at presentation time inside whoever is reading
the JSON, not inside the assessor.

## Consequences

- Two packs to maintain when the Finding catalogue changes — but
  changes are mechanical (a renamed ProbeID surfaces in both packs'
  rule code, caught by both packs' tests).
- The `Assessment.Framework` field is now load-bearing in the store
  schema and the JSON API; renaming it is a breaking change.
- Operators who run `--framework both` get two records per scan;
  the markdown report needs to know which framework it is rendering.
  This is reflected in `docs/assessor.md`.
- The "worst rule wins" collapse is duplicated in both packs by
  design — they apply the same auditability principle (no silent
  inflation of an absent rule's contribution) but each pack remains
  free to evolve its own definition of "worst".
- A future framework (NIS2, ENISA-specific, sector toetsen) lands as
  a sibling package, not a configuration of an existing one.

## Addendum (2026-04-30) — first-pack rename

The first-party rule pack — originally named `dictu` after DICTU's
*Toetsingsinstrument Soevereiniteit Clouddiensten*, which it was
inspired by — was renamed to `wand` (Wanderer-NL) per
[ADR-0011](0011-rename-dictu-to-wand.md). The architectural
decision in this ADR (two parallel rule packs sharing the Finding
contract; per-pack score scale; `Framework` field on every
persisted Assessment) is unchanged; only the identifier of the
first pack moved. ADR-0011 documents the legal and reputational
reasoning.
