---
status: draft
last_reviewed: 2026-07-12
---

# Wanderer documentation

Wanderer is a passive sovereignty-posture scanner: it observes what a
public-facing deployment exposes — TLS chains, egress destinations,
host configuration — and turns that into findings and DICTU scores you
can act on. This is the documentation set; the project-level
[`README`](../README.md) covers what Wanderer is and why it exists, and
spec-driven change proposals live in [`openspec/`](../openspec/).

Status: **draft**. These pages were reorganised into the handbook docs
contract and carry `status: draft` until each is reviewed on its own
merits.

## Start here

- [**Tutorial**](how-to/tutorial.md) — hands-on walkthrough. Run your
  first scan, read the output, understand what each finding means.

## How-to

- [**Operator guide**](how-to/operator.md) — install, flags, env vars,
  troubleshooting. The full configuration surface.
- [**Scheduling**](how-to/scheduling.md) — cron-driven scans inside
  `wanderer serve`, schedules file format, SIGHUP semantics.
- [**Releasing**](how-to/releasing.md) — cut a release and keep the
  downstream ExApp in sync.

## Reference

- [**Findings reference**](reference/findings.md) — every ProbeID,
  severity, and attribute shape. Use this when interpreting output or
  when writing code that consumes findings.
- [**Assessor**](reference/assessor.md) — how DICTU scores are produced
  from findings, how to read an Assessment, and how to extend the rule
  set.
- [**Exporters**](reference/exporters.md) — CSV and JSONL export from
  the local store, with composable selectors.
- [**MCP server**](reference/mcp.md) — drive scans and read findings
  from Claude Code or Claude Desktop via the Model Context Protocol.
- [**Drift**](reference/drift.md) — what counts as posture drift between
  two scans, the rule set, the `wanderer diff` CLI.
- [**Egress**](reference/egress.md) — what the egress probe catches and
  misses, the redaction guarantee, classifier rules.
- [**Observability**](reference/observability.md) — logs, Prometheus
  metrics, OpenTelemetry (deferred).

## Explanation

- [**Architecture**](explanation/architecture.md) — how the components
  fit together, key design decisions, how to add a probe.
- [**Agent**](explanation/agent.md) — `wanderer agent` host-side
  inspectors, config, least-privilege user setup, HMAC remote transport.
- [**Maintainability**](explanation/maintainability.md) — single entry
  point for contributors: CHANGELOG, ADRs, API stability, testing
  baseline, dependency policy, commit style.
- [**Architecture Decision Records**](explanation/adr/README.md) — the
  decisions that shape the project, append-only.
