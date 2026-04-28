# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once a first release is cut. Until then every entry lives under
`[Unreleased]`.

## [Unreleased]

### Fixed

- DICTU rule `dictu.juridisch.apex_ip_eea` looked for `ProbeID == "dns.A"`
  / `"dns.AAAA"` while the DNS probe emits the lowercase variants
  documented in `docs/findings.md`. The unit test agreed with the rule
  (also using uppercase) so the bug was invisible in CI but caused
  every real scan to return Onbekend for the apex jurisdiction. Rule
  and unit test are now both lowercase, with a comment pinning the
  invariant. (`internal/assessor/dictu/rules.go`,
  `internal/assessor/dictu/rules_test.go`)
- Scanner now feeds DNS- and HTTP-discovered hosts into the IP probe
  before it runs. Previously the IP probe only resolved `target.Domain`
  and the operator-provided `target.Related`; MX hosts (`dns.mx`) and
  third-party hosts (`http.third_party`) found by other probes were
  never looked up, so `dictu.juridisch.mx_vendor_jurisdiction`,
  `dictu.technologie.third_parties_eea`, and the third-party half of
  `dictu.technologie.no_us_hyperscaler` silently returned Onbekend on
  every real scan. New `expandRelatedFromFindings` helper builds an
  enriched target only for the IP probe (other probes still see the
  original target). `buildProbes` now orders the IP probe last so the
  HTTP probe has had a chance to discover third parties.
  (`internal/scanner/scanner.go`, `cmd/wanderer/scan.go`)

### Added

- SSRF guard for outbound probe traffic. `internal/probe/ssrf.go` ships
  a `SafeDialer` plus a static `*net.IPNet` table covering loopback,
  link-local, RFC1918, CGNAT, IPv6 ULA, IPv6 link-local and the cloud
  metadata IPs (169.254.169.254, fd00:ec2::254). The dialer is wired
  into the HTTP probe's `http.Client.Transport` and the TLS probe's
  `tls.Dialer`. `wanderer scan` and `wanderer serve` gain
  `--allow-private-targets` (default off — private blocked); the
  `POST /scans` handler refuses requests whose domain resolves only to
  private addresses unless the flag is set, so an authenticated API
  client cannot turn the scanner into an internal-network probe. Unit
  tests cover one blocked family per address class.
  (`openspec/changes/fill-mvp-gaps` goal #3)
- Passive subdomain discovery. `internal/probe/tls/tls.go` extracts SAN
  names from the apex certificate and emits each as a `dns.subdomain`
  Finding tagged `source: "ct_log"`. `internal/probe/dns/subdomains.go`
  probes 18 common prefixes via the system resolver; resolving names
  emit `dns.subdomain` Findings tagged `source: "prefix_probe"`. When
  every prefix resolves to the same IP set, the probe collapses the
  noise into a single `dns.subdomain.wildcard` Finding rather than 18
  spurious hits. Discovered subdomains feed pass 2 of the scanner via
  `expandRelatedFromFindings` so the IP probe resolves them. Unit tests
  cover hit / miss / wildcard against a fake resolver.
  (`openspec/changes/fill-mvp-gaps` goal #4)
- Amass importer. `internal/scanner/amass.go` parses the JSONL produced
  by `amass enum -json` and returns the FQDN list to `target.Related`.
  `wanderer scan --amass <file>` and `POST /scans` `amass_json` field
  (server-local path only; no inline body) wire it through. Malformed
  JSON aborts at startup so an unenriched scan does not silently look
  fine. `docs/operator.md` documents the
  `amass enum -passive -d … -json out.json && wanderer scan … --amass
  out.json` recipe.
  (`openspec/changes/fill-mvp-gaps` goal #5)
- EU CSF / SEAL rule pack. New `pkg/models.SealLevel` (SEAL0–SEAL4)
  and `models.Framework` enum. `internal/assessor/eucsf/rules.go`
  ships five rules — `eucsf.sov2.cert_issuer_eu`,
  `eucsf.sov2.apex_jurisdiction`, `eucsf.sov3.mx_jurisdiction`,
  `eucsf.sov4.operational_eu`, `eucsf.sov6.no_us_hyperscaler` —
  with at least one EU and one non-EU fixture per rule.
  `wanderer assess --framework dictu|eucsf|both` selects which pack
  runs, plumbed through CLI flags, the HTTP API, and persistence.
  `docs/assessor.md` carries the SEAL section with the score-mapping
  table.
  (`openspec/changes/fill-mvp-gaps` goal #6)
- Read-only operator UI on `wanderer serve` (`--ui`). Mounts at
  `/ui/` with three GET-only routes: `/` (target overview with
  latest scan and worst-dimension framework score per target),
  `/scans/{id}` (findings grouped by probe prefix), and
  `/targets/{id}/drift` (drift findings since `?since=<RFC3339>`).
  Templates and CSS are embedded via `go:embed`; no JavaScript, no
  external assets. Authentication is HTTP Basic against an htpasswd
  file (`--ui-htpasswd <path>` or `WANDERER_UI_HTPASSWD`). Only
  bcrypt entries (`$2a$`/`$2b$`/`$2y$`) are accepted; `$apr1$` MD5,
  `{SHA}` SHA-1, `$5$` and `$6$` crypt entries are rejected at
  startup with an explicit "use bcrypt (`htpasswd -B`)" error — one
  algorithm = one battle-tested verification path. The htpasswd
  file is re-read on every request so credentials can rotate without
  restarting. `ui_test.go` includes a static-analysis check that
  greps the package source for `r.Post|Patch|Delete|Put` and fails
  the build if any mutating handler appears, locking in the
  read-only invariant. Default is off; the JSON API and existing
  flags are unchanged. (`internal/ui/`, `cmd/wanderer/serve.go`,
  `openspec/changes/fill-mvp-gaps` goal #7)
- Concurrent pass-1 execution. The scanner now fans out DNS, TLS,
  HTTP and WHOIS probes via `errgroup.WithContext`, each wrapped in a
  `defer recover()` that converts panics to a nil group error so a
  single misbehaving probe cannot poison the whole scan. Pass 2
  (the IP probe) still runs serially after the pass-1 join so it can
  consume hosts the others discovered. The global budget remains a
  single `context.WithTimeout`. A wall-clock test asserts pass 1
  finishes in roughly the slowest probe's duration, not the sum.
  (`openspec/changes/fill-mvp-gaps` goal #8)
- RDAP / WHOIS probe. `internal/probe/whois/whois.go` calls
  `https://rdap.org/domain/<domain>` (injectable for tests) with a
  5-second timeout and walks the vCard array per RFC 7483. Emits
  `whois.registrant` (country, juridisch dimension, finding
  severity), `whois.registrar` (name, info), and a single
  `whois.unavailable` Finding on any network/parse failure so the
  rest of the scan continues. New rule
  `dictu.juridisch.registrar_jurisdiction` consults the registrant
  country (EEA → soeverein, outside-EEA → afhankelijk, absent →
  onbekend). Stdlib `net/http` only — no WHOIS-43 socket, no
  third-party SDK. Tests cover happy path, HTTP error, malformed
  JSON, empty domain, and no-entities cases.
  (`openspec/changes/fill-mvp-gaps` goal #9)
- Numbered schema migrations. `internal/store/migrations.go`
  introduces a `schema_migrations` table and a versioned migration
  slice. The current schema becomes migration 001 (idempotent
  `CREATE TABLE IF NOT EXISTS` so pre-migration databases adopt it
  cleanly) and the `findings.source_modus` ALTER becomes migration
  002. The previous string-matched ALTER tolerance is removed in
  favour of the migration runner. Tests cover: fresh DB applies all
  migrations, a DB at version N applies only N+1..M, and a failing
  migration rolls back without recording its version.
  (`openspec/changes/fill-mvp-gaps` goal #10)
- End-to-end integration tests in `internal/assessor/dictu/integration_test.go`
  that drive the real DNS probe through the real DICTU rules. Pins the
  probe-ID/assessor-ID contract for `apex_ip_eea`,
  `mx_vendor_jurisdiction`, and `third_parties_eea` so future drift on
  either side breaks the build instead of silently returning Onbekend.
- Scanner unit test `TestIPProbeReceivesDiscoveredHosts` pins the
  fan-out invariant: the IP probe sees discovered hosts in
  `target.Related`, while other probes continue to see the original
  target.
- Egress probe: agent-side observation of where data goes when it
  leaves the host. New `internal/probe/egress/` walks configured
  config files (`.env`, `.yaml`, `.yml`, `.toml`, `.ini`, `.conf`,
  `.json`), `/proc/<pid>/environ`, and systemd unit files, then
  classifies discovered URLs and hosts into nine categories
  (`object_storage`, `database`, `smtp`, `oidc`, `log_shipper`,
  `webhook`, plus `unknown` / `unconfigured` / `error`). Findings
  are tagged with `SourceModus = "egress"` and carry a
  `classifier_rule` attribute. Optional GeoLite2 annotation adds
  ASN/organisation/country per host. The redactor runs in front of
  every value emission path; ADR-0008 documents the contract and
  test discipline. Symlinks pointing outside the configured root
  are not followed. `docs/egress.md` is the operator guide.
  (`openspec/changes/add-egress-probe`)
- Inventory probe + agent: new `wanderer agent` subcommand running
  host-side inspectors (systemd, dpkg, rpm; nextcloud opt-in;
  docker as graceful-unavailable placeholder pending a follow-up).
  New `pkg/models.SourceModus` field tags every Finding with its
  origin (perimeter / inventory / egress / drift) so the assessor's
  completeness calculation can distinguish them; the `findings`
  table gains a `source_modus` column with idempotent migration.
  New `POST /scans/{id}/findings` endpoint authenticates agents via
  HMAC-SHA256 over `<timestamp>\n<body>` with a ±5-minute replay
  window (constant-time compare; single 401 surface). Agent config
  in YAML covers local-mode (writes to a shared SQLite file) and
  remote-mode (HMAC-signed HTTPS to a central core). ADR-0007
  records the trust-model rationale; `docs/agent.md` is the
  operator guide.
  (`openspec/changes/add-inventory-probe`)
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
