# Documentation

Start here:

- [**Tutorial**](tutorial.md) — hands-on walkthrough. Run your first
  scan, read the output, understand what each finding means.

Reference:

- [**Operator guide**](operator.md) — install, flags, env vars,
  troubleshooting. The full configuration surface.
- [**Findings reference**](findings.md) — every ProbeID, severity, and
  attribute shape. Use this when interpreting output or when writing
  code that consumes findings.
- [**Assessor**](assessor.md) — how DICTU scores are produced from
  findings, how to read an Assessment, and how to extend the rule set.
- [**Exporters**](exporters.md) — CSV and JSONL export from the local
  store, with composable selectors.
- [**MCP server**](mcp.md) — drive scans and read findings from
  Claude Code or Claude Desktop via the Model Context Protocol.
- [**Architecture**](architecture.md) — how the components fit
  together, key design decisions, how to add a probe.
- [**Observability**](observability.md) — logs, Prometheus metrics,
  OpenTelemetry (deferred).

Contributing:

- [**Maintainability**](maintainability.md) — single entry point for
  contributors: CHANGELOG, ADRs, API stability, testing baseline,
  dependency policy, commit style.
- [**Architecture Decision Records**](decisions/README.md) — the
  decisions that shape the project, append-only.

The project-level [`README`](../README.md) covers what Wanderer is and
why it exists. Spec-driven change proposals live in
[`openspec/`](../openspec/).
