# Proposal: Maintainability Baseline

## Intent

Before six parallel changes (assessor, exporters, MCP, inventory probe,
egress probe, scheduling) start landing, lock in the project hygiene
that lets them coexist cleanly and stay auditable. This change does
not add product features — it adds the rails that the feature changes
run on.

Without this baseline, parallel implementation diverges fast: two
probes duplicate a utility because neither knows where it should live,
three changes add entries to `cmd/wanderer/main.go` with conflicting
styles, and there is no CHANGELOG to reconstruct the ordering from.

## Scope

**In scope:**

- `CHANGELOG.md` at the project root, Keep-a-Changelog style.
- `docs/decisions/` folder for Architecture Decision Records, with a
  minimal template (`NNNN-short-title.md`).
- Dependency policy: pure-Go preferred, minimum supported Go version
  documented, no auto-upgrade, versions pinned in `go.mod`.
- API stability classes: `pkg/models` is the stable public contract;
  `internal/*` is free to change without notice.
- Testing baseline: every new package ships with tests, table-driven
  where possible; target ≥70% coverage for `internal/*`, ≥90% for
  `pkg/models`.
- Documentation baseline: every new package has a package comment
  explaining intent in 2–4 sentences; every user-facing feature has a
  matching section or file in `docs/`.
- OpenSpec discipline: any non-trivial change (new command, new probe,
  new data type) must propose before implementing; housekeeping fixes
  are exempt.
- `CODEOWNERS` stub so reviews have a default owner.
- Commit message conventions: short imperative subject, body wraps at
  72, trailer with `Co-Authored-By` when pair-coded or
  agent-assisted. No ticket IDs required (OpenSpec is the tracker).

**Out of scope:**

- Any change to MVP behaviour.
- Release automation / semver tagging — that ships with
  `add-scheduling` when we have something worth releasing.
- Security review / SBOM of Wanderer itself — comes with
  `add-inventory-probe` which introduces the concept anyway.

## DICTU dimensions informed

None directly. This change targets the *project*, not the *subject of
scans*. It is justified by the Operationeel dimension of the Conduction
toolchain itself: an un-auditable codebase cannot produce an auditable
observatory.

## Passive/active boundary

N/A — no new probes.

## Why now, not later

Retrofitting a CHANGELOG after six feature changes have landed is
strictly worse than starting with an empty one. Retrofitting an API
stability policy after consumers have imported `internal/scanner` is
strictly worse than documenting it before they do. The value of this
change is proportional to how many other changes land on top of it, so
it must go first.
