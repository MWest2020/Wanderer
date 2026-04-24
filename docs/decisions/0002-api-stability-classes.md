# 0002. API stability classes

- Status: accepted
- Date: 2026-04-24

## Context

Wanderer ships both a runnable CLI (`cmd/wanderer`) and importable
Go packages. Without an explicit stability declaration, external
consumers will import whatever compiles — including internal types
that we intend to reshape as the project evolves. That locks us
into accidental public contracts and makes routine refactors
breaking changes after the fact.

Go's standard mechanism for this is the `internal/` directory,
which the tool-chain already enforces at build time. The missing
piece is a documented promise about what lives outside `internal/`.

## Decision

Two stability classes:

1. **Stable public API** — everything under `pkg/`.
   - `pkg/models` is the canonical example today: it defines
     `Finding`, `Target`, `Scan`, and their enums, and those are the
     contract that external tools (exporters, assessors,
     dashboards) can rely on.
   - Breaking changes to `pkg/` SHALL be accompanied by a
     `### Changed (breaking)` entry in `CHANGELOG.md` and a
     corresponding ADR explaining the migration.
   - Deprecations are signalled with a `// Deprecated:` doc comment
     one release before removal.

2. **Private implementation detail** — everything under `internal/`.
   - `internal/probe`, `internal/scanner`, `internal/store`,
     `internal/api`, etc. may change at any time without notice.
   - Third-party packages that import `internal/*` either by
     `replace` directive or by vendoring are explicitly
     unsupported.
   - Internal refactors do not require a CHANGELOG entry unless
     they happen to change observable behaviour.

The `cmd/` binary surface (flags, subcommand names, stdout shape,
HTTP route contracts) is treated as stable public API for
operational purposes, even though Go package stability does not
apply.

## Consequences

- External integrations have a clear target: build against `pkg/`
  and the CLI/HTTP surface, never `internal/`.
- Refactors inside `internal/` stay cheap — no CHANGELOG ceremony,
  no ADR obligation.
- Adding a new public type means thinking first about whether it
  belongs in `pkg/` (contract, long-lived) or in `internal/`
  (convenience, free to change). The default is `internal/`.
- If a type in `internal/` turns out to be genuinely useful to
  external callers, promoting it to `pkg/` is a deliberate
  decision: re-export with a CHANGELOG entry and, if the signature
  changes during promotion, an ADR.
