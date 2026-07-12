---
status: draft
last_reviewed: 2026-07-12
---

# Maintainability

This is the single entry point for contributors. It gathers the
habits that keep Wanderer auditable without adding tooling ceremony.
If you are about to open a pull request, skim this page first.

## CHANGELOG discipline

Every merged change updates [`CHANGELOG.md`](../../CHANGELOG.md)
under `## [Unreleased]`. Use the Keep-a-Changelog sections
(`Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`)
and prefer short, imperative entries that reference the OpenSpec
change they came from. A breaking change to `pkg/*` goes under
`### Changed (breaking)` and SHALL also cite an ADR.

Housekeeping fixes (typos, lint) still get a one-line `### Fixed`
entry. It is cheap and it keeps the log complete.

## ADR discipline

Decisions that constrain future work live in
[`docs/decisions/`](adr/README.md) as Architecture Decision
Records. Write one when a change adds a cross-cutting dependency,
introduces a new data-model contract, moves the passive/active
boundary, or declares the stability class of a new public package.
Do not write one for tactical fixes.

ADRs are additive: do not delete old ones, supersede them with a
new entry and update the `Status` field on the old one.

## OpenSpec discipline

Non-trivial changes — new command, new probe, new data type, any
change to `pkg/models` — start as an OpenSpec change proposal
under [`openspec/changes/<name>/`](../../openspec/). Use the
`/opsx:propose` and `/opsx:apply` skills (or the equivalent
`openspec-*` skills) to drive them. Rationale and details are in
[ADR-0001](adr/0001-openspec-workflow.md).

Small, behaviour-preserving fixes go straight to a pull request
with just a CHANGELOG entry.

## API stability

Two classes, fully described in
[ADR-0002](adr/0002-api-stability-classes.md):

- `pkg/` — stable public contract. Breaking changes require a
  `### Changed (breaking)` CHANGELOG entry and an ADR.
- `internal/` — private. Reshape freely. External imports are
  unsupported.

Default location for new types is `internal/`. Promote to `pkg/`
only when an external consumer needs it.

## Testing baseline

Every new Go package ships with at least one `_test.go` file and a
package-level doc comment of two to four sentences describing
intent. Coverage targets:

- `internal/*` — ≥ 70% line coverage.
- `pkg/models` — ≥ 90% line coverage.

Table-driven tests are the default style. Mocks are acceptable at
package boundaries; integration tests in `internal/api` and
`internal/store` should exercise real HTTP handlers and the real
SQLite backend respectively, not reimplement them.

## Documentation baseline

Every new Go package has a `// Package <name> …` doc comment.
Every user-facing feature has a matching section or file under
[`docs/`](../index.md). If a flag, environment variable, or HTTP
route is added, it appears in [`docs/operator.md`](../how-to/operator.md).

## Dependency policy

Policy summary (full rationale in
[ADR-0003](adr/0003-dependency-policy.md)):

- Prefer the standard library.
- Prefer pure-Go dependencies; CGo needs an ADR.
- Pin versions in `go.mod`; commit `go.sum`; no automated bumps.
- Each new top-level dependency gets a one-line justification in
  the introducing proposal or design document.

## Commit and pull-request style

- Subject line: short imperative, ≤ 72 characters, prefixed with a
  conventional type (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`,
  `test:`). Scope is optional.
- Body: wrap at 72 columns. Explain the *why*, not the *what* —
  the diff already shows what.
- Trailers: `Co-Authored-By: …` when pair-coded or agent-assisted.
  No ticket IDs required; OpenSpec is the tracker.
- One logical change per commit where practical. Squash-merging is
  fine when a branch has noisy intermediate commits.

## What is deliberately *not* here

No commit-message hooks, no CHANGELOG-enforcement CI, no PR
template, no labelling bot, no Dependabot. These are good at scale;
at this project's current size they trade real friction for
speculative quality. Revisit once concrete drift shows up.
