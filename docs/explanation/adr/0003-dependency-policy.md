---
status: draft
last_reviewed: 2026-07-12
---

# 0003. Dependency policy

- Status: accepted
- Date: 2026-04-24

## Context

Every added dependency is a perpetual cost: transitive CVE exposure,
cross-compilation friction when C code sneaks in, version drift,
and one more thing to audit when the output of Wanderer is itself
used as evidence. A small greenfield project can avoid those costs
cheaply if the policy is explicit from the start; retrofitting a
policy after dozens of dependencies have landed is never effective.

The MVP already made one deliberate, non-obvious choice:
`modernc.org/sqlite` (pure Go) instead of `mattn/go-sqlite3` (CGo).
That choice keeps `go build ./...` trivial on every supported
platform without a C tool-chain. Policy-wise, it should be the
default expectation, not an exception.

## Decision

1. **Prefer the standard library.** If the standard library covers
   a need adequately, do not add a dependency even if a third-party
   package is "nicer".

2. **Pure-Go over CGo.** New dependencies SHALL be pure Go unless a
   strong justification is captured in the proposing change's
   `design.md` and an accompanying ADR. The carrying cost of losing
   `GOOS=linux GOARCH=arm64 go build ./...` without a cross
   tool-chain is higher than most ergonomic wins.

3. **Pinning.** Versions SHALL be pinned in `go.mod` (Go does this
   by default). `go.sum` is committed. No auto-upgrade bots are
   configured by policy — the cost of a spurious update is higher
   than the cost of a deliberate periodic review.

4. **Minimum supported Go.** Documented in `go.mod` and kept in
   lockstep with the CI matrix. Current target: the latest stable
   Go release. Bumping the minimum is a CHANGELOG entry and does
   not require an ADR.

5. **Rationale on introduction.** Every new top-level dependency
   gets a one-line justification in the proposing change's
   `proposal.md` or `design.md`. For decisions that constrain
   future changes (e.g. "we are now committed to chi for routing"),
   write a dedicated ADR.

6. **No vendoring.** The module cache is the source of truth.
   Vendoring is only revisited if a regulatory requirement forces
   it.

## Consequences

- `go build ./...` stays trivial and reproducible on every
  platform the project targets.
- Adding a dependency is a small, visible friction: it needs a
  sentence somewhere. That friction is the point.
- If a future change genuinely needs a CGo dependency (e.g. a
  protocol library with no pure-Go alternative), the policy is
  overridable by explicit ADR — it is not a hard no.
- Dependency review is a *periodic human task*, not a CI gate:
  we accept the risk of a short lag behind upstream security
  releases in exchange for not chasing bot-generated churn.
