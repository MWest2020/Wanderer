# 0008. Egress redaction: hard contract, tested first

- Status: accepted
- Date: 2026-04-27

## Context

The egress probe reads application configuration to discover where
data goes when it leaves the host. Every place it looks — config
files, process environments, systemd unit files — is also a place
where secrets live: API keys, database passwords, signing keys,
session tokens. If the probe ever leaks one of those into a
`Finding`, an `Evidence` blob, or a log line, the leak is
catastrophic — the secret has now propagated through the store,
the exporters, the MCP server, and any operator dashboards.

This is a different risk class from "the probe might emit a
spurious finding". A noisy classifier produces friction; a leaking
redactor produces an incident.

## Decision

The redactor is the security contract of the egress probe and is
treated accordingly.

1. **Redaction is mandatory and lives at one place.** Every value
   the probe handles passes through
   `internal/probe/egress/redact.Apply(key, value)` before it is
   stored in `Attributes`, written to `Evidence`, or emitted to
   slog. There is exactly one redaction call site per output path;
   adding a second emission path requires routing through `Apply`
   again.

2. **Redact on key OR value.** A key name matching the secret-name
   pattern (case-insensitive: `KEY`, `SECRET`, `TOKEN`, `PASSWORD`,
   `PASSWD`, `PW`, `PWD`, `ACCESS_KEY`, `PRIVATE_KEY`,
   `CLIENT_SECRET`, `AUTH_TOKEN`, `BEARER`) replaces the entire
   value with `«redacted»` regardless of shape. Independently, a
   value matching a known token shape (AWS, Slack, GitHub, Google,
   PEM block, JWT) is also replaced. URLs with inline credentials
   have only the password portion replaced.

3. **Redaction is tested before scanners are wired.** The redact
   package shipped in this change carries golden-file tests for
   ≥30 realistic snippets *before* the scanners or classifier
   integrate it. Coverage target is 95% — a passing rule that
   never matched is a security regression risk.

4. **Failures fail closed.** A panic in any sub-rule is recovered
   to a "redact" outcome, not a "passthrough" outcome. We would
   rather over-redact than under-redact.

5. **The placeholder is a single literal string.** `«redacted»` is
   used everywhere (tests, docs, runtime). Operators learn one
   marker.

## Consequences

- **Adding a new secret pattern is a one-file change with tests.**
  Append to either `secretKeyNameRE` or `secretValueREs` in
  `redact.go`, add a snippet to `redact_test.go` covering both the
  positive and the false-positive boundary, ship.
- **False negatives are the danger we explicitly own.** Every time
  a security review (formal or ad-hoc) surfaces a leaking shape
  Wanderer did not redact, the fix is a regex-and-test PR. We
  document this in `docs/egress.md` so operators understand the
  guarantee is "best effort against known secret shapes, not
  cryptographic certainty".
- **Performance is irrelevant at this scale.** The redactor runs
  once per candidate; a config file has hundreds of candidates,
  not millions. We do not optimise; we keep it boring and correct.
- **Cross-cutting policy enforcement is reviewable.** A reviewer
  can grep `redact.Apply(` in the egress package and see every
  location secrets pass through. There is no clever wrapper
  obscuring the call site.

When to revisit: when an external security review surfaces a class
of secret the value-shape rules cannot catch, or when the egress
probe gains a flow-observation mode (eBPF) where the data shape
itself is different. Either case is a follow-up change with its
own ADR.
