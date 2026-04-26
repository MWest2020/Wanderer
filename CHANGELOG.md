# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once a first release is cut. Until then every entry lives under
`[Unreleased]`.

## [Unreleased]

### Added

- Scheduling + drift: an in-process cron scheduler runs alongside
  `wanderer serve` (via `--schedules <file>`), invoking
  `scanner.Scan` per tick and feeding each new scan to the drift
  engine. New `internal/drift/` engine compares consecutive scans of
  the same target and emits `drift.*` Findings (TLS issuer, days
  left, MX/NS sets, IP country, HTTP third parties). New
  `wanderer diff <scan-a> <scan-b>` CLI prints would-be drift as
  markdown without touching the store. New `GET
  /targets/{id}/drift?since=<RFC3339>` HTTP endpoint. ADR-0006
  records why the scheduler is in-process rather than a Kubernetes
  CronJob.
  (`openspec/changes/add-scheduling`)
- MCP server: `wanderer mcp` subcommand speaks the Model Context
  Protocol over stdio (line-delimited JSON-RPC 2.0). Exposes five
  tools (`scan_domain`, `get_scan`, `list_scans`, `assess_scan`,
  `get_assessment`) and the `wanderer://` resource family for
  reading scans and assessments. Hand-rolled dispatcher in
  `internal/mcp/`; no new dependencies. ADR-0005 records the
  transport choice. `docs/mcp.md` carries the install snippet for
  Claude Desktop / Claude Code.
  (`openspec/changes/add-mcp-server`)
- Exporters: `wanderer export <findings|scans|assessments>` subcommand
  with `--format csv|jsonl`, file or stdout output, and composable
  `--scan`, `--probe`, `--dimension`, `--since`, `--until` selectors
  pushed down to the SQL query. Adds `internal/export/` writers,
  `store.ListFindings` / `ListScans` / `ListAssessments` query
  helpers (with a `Selectors` type), and `docs/exporters.md` for
  recipes (Excel, jq, Grafana, diff).
  (`openspec/changes/add-exporters`)
- Assessor: `pkg/models.Assessment` and its supporting types
  (`Score`, `Completeness`, `DimensionScore`, `Rationale`), the
  `internal/assessor` rule engine, the DICTU MVP rule set under
  `internal/assessor/dictu/` (10 rules across four dimensions),
  markdown/JSON/text report renderers, the `wanderer assess` CLI
  subcommand, and `POST /scans/{id}/assessments` +
  `GET /assessments/{id}` on the HTTP API. Assessments are persisted
  in a new `assessments` table via `store.CreateAssessment`,
  `store.GetAssessment`, and `store.ListAssessmentsForScan`. ADR-0004
  records why rules are Go functions rather than a DSL.
  (`openspec/changes/add-assessor`)
- Maintainability baseline: `CHANGELOG.md`, `CODEOWNERS`, the
  `docs/decisions/` ADR folder with seed records for the OpenSpec
  workflow, API stability classes, and dependency policy, plus
  `docs/maintainability.md` as the single contributor entry point.
  (`openspec/changes/add-maintainability-baseline`)
- Initial MVP scanner suite: DNS (A/AAAA/MX/NS/CNAME/TXT/CAA), TLS
  chain + crt.sh certificate-transparency lookup, IP→ASN→country via
  a local MaxMind GeoLite2 database, and HTTP apex fetch with
  third-party resource extraction. Findings persist to SQLite through
  `modernc.org/sqlite`, the `wanderer` CLI exposes `scan` and `serve`
  subcommands, and a chi-based HTTP API serves `POST /scans` and
  `GET /scans/{id}`. slog (JSON) and Prometheus counters are wired
  into scanner and probes; OpenTelemetry traces were intentionally
  deferred (see `docs/observability.md`).
  (`openspec/changes/archive/2026-04-24-init-mvp-scanners`)

[Unreleased]: https://github.com/MWest2020/wanderer/commits/main
