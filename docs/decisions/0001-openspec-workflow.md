# 0001. OpenSpec workflow for non-trivial changes

- Status: accepted
- Date: 2026-04-24

## Context

Wanderer is a small project but its output (findings and, later,
assessments) is meant to be audit-grade: two reviewers running the
same scan should be able to reconstruct *why* the system produced
what it produced. That auditability starts at the source tree. If
changes arrive as free-form commits with only code diffs to read,
the rationale decays the moment the commit scrolls off screen.

Freestyle development also makes parallel work brittle: two
contributors pulling in the same direction will, without a written
proposal, diverge on naming, trust boundaries, and scope.

## Decision

Every non-trivial change — new command, new probe, new data type,
new external interface, or any modification to `pkg/models` —
SHALL start as an [OpenSpec](https://github.com/anthropics/openspec)
change proposal under `openspec/changes/<name>/`. The proposal, its
design notes, spec deltas, and task list are reviewed together with
the eventual implementation.

Exempt: lint fixes, typo fixes, dependency bumps that do not change
behaviour, and refactors that preserve the public API. These go
straight to a pull request.

The `/opsx:*` skill family (`explore`, `propose`, `apply`,
`archive`) is the preferred UX for working with OpenSpec. Skills
that are not the experimental `opsx` variants (`openspec-propose`,
`openspec-apply-change`, etc.) are equivalent and may be used
interchangeably.

## Consequences

- Every non-trivial change carries its own rationale, specs, and
  implementation plan in the same directory — retrievable years
  later without replaying commits.
- Reviews happen in two places: once on the proposal before coding,
  once on the implementation PR. This doubles calendar time on small
  items, which is the intended cost — it is bought for the benefit
  of clear history on the non-small items.
- The `openspec` CLI becomes a hard dependency of the development
  environment. It is not a runtime dependency of Wanderer itself.
- `openspec/changes/archive/YYYY-MM-DD-<name>/` is the permanent
  record; once archived, a change is immutable.
