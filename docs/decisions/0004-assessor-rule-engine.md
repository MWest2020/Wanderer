# 0004. Assessor rule engine: Go functions, not a DSL

- Status: accepted
- Date: 2026-04-24

## Context

The assessor maps Findings to DICTU Toetsingsinstrument scores. The
rule set is small today (around ten rules) but will grow as more
probes land, and eventually we will also want to ship rule sets for
other frameworks (NIS2, BIO, ISO 27001).

There is a tempting design: a YAML-or-JSON rule DSL with
jsonpath-like expressions, hot-reloaded rule files, maybe an admin
UI to tune them. This ADR records why we did not do that.

## Decision

Rules are Go functions, compiled into the `wanderer` binary.

Specifically, each rule is a value of type `assessor.Rule` with a
`Match func([]models.Finding) assessor.RuleResult` field. Rules live
in `internal/assessor/<framework>/` (today: `dictu`) and are wired
into a slice by a `DefaultRules()` helper. There is no external rule
configuration file, no runtime rule upload endpoint, and no
expression language.

When a second framework (NIS2 or BIO) is added, it ships as a
sibling package with its own `DefaultRules()` and a CLI/API flag to
select it. Rules that overlap between frameworks are duplicated in
Go; we accept that cost.

## Consequences

- **Compile-time safety.** A rule with a typo or a broken assumption
  fails `go build`, not a production scan. The DICTU rule set is
  the scoring contract; a runtime break would be a visible audit
  failure on a live dashboard.
- **One tool-chain for authoring rules and testing them.** A rule is
  a Go function, so its test is a `go test` with fabricated
  Findings. There is no separate YAML-linter, no DSL unit-test
  harness, no "does this jsonpath match?" debugger.
- **Rule set is versioned with the code.** The rules that scored a
  given Assessment are the rules that were in the binary that ran —
  retrievable by commit, not by a database snapshot of the "active"
  rule set at the time.
- **Extensibility is deliberately slower.** External contributors
  add rules via a pull request that includes a test. This is the
  intended cost: the DICTU scoring output is audit-grade and a
  pull-request review is the boundary we want on new rules.
- **When to revisit.** If we ever have more than one framework and
  genuinely identical rules overlapping across them, or if external
  integrators start maintaining their own frameworks against a
  running Wanderer instance, re-open this question. At that point a
  plugin system (Go plugins, WASM, a DSL) becomes worth the cost.
  Until then, Go functions are the right tool.
