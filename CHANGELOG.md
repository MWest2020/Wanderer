# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once a first release is cut. Until then every entry lives under
`[Unreleased]`.

## [Unreleased]

### Added

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
