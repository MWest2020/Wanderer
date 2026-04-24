# Tasks: Initialise MVP Scanner Suite

## 1. Foundations

- [x] 1.1 `pkg/models/finding.go` — Finding, Severity, DimensionHint enums
- [x] 1.2 `pkg/models/target.go` — Target type with validation
- [x] 1.3 `pkg/models/scan.go` — Scan lifecycle states
- [x] 1.4 `internal/store/sqlite.go` — open DB, migrate schema at startup
- [x] 1.5 `internal/store/store_test.go` — round-trip Finding serialisation

## 2. Probes

- [x] 2.1 `internal/probe/probe.go` — shared `Probe` interface and `Config`
- [x] 2.2 `internal/probe/dns/` — A/AAAA, MX, NS, CNAME, TXT, CAA
- [x] 2.3 `internal/probe/dns` fixtures + unit tests with fake resolver
- [x] 2.4 `internal/probe/tls/` — chain inspection, issuer, SAN, validity
- [x] 2.5 `internal/probe/tls` CT-log lookup via crt.sh JSON endpoint with graceful degradation
- [x] 2.6 `internal/probe/ip/` — GeoLite2 loader, lookup, country + ASN
- [x] 2.7 `internal/probe/ip` — fail-fast on missing/corrupt DB at startup
- [x] 2.8 `internal/probe/http/` — fetch apex, parse HTML, extract third parties
- [x] 2.9 `internal/probe/http` — redirect cap, body cap, robots.txt honour

## 3. Orchestration

- [x] 3.1 `internal/scanner/scanner.go` — sequences probes, shared context
- [x] 3.2 `internal/scanner` — per-probe timeout, global budget, partial-scan result
- [x] 3.3 `internal/scanner/scanner_test.go` — one probe fails, scan still completes

## 4. Interfaces

- [x] 4.1 `cmd/wanderer/main.go` — root command with subcommands
- [x] 4.2 `cmd/wanderer/scan.go` — `wanderer scan <domain>` with human-readable output
- [x] 4.3 `cmd/wanderer/serve.go` — `wanderer serve` starts the HTTP API
- [x] 4.4 `internal/api/api.go` — chi router, `POST /scans`, `GET /scans/{id}`
- [x] 4.5 `internal/api` — structured JSON errors, health endpoint

## 5. Observability

- [x] 5.1 `log/slog` JSON handler wired into scanner + probes, request-id propagated through context
- [x] 5.2 Prometheus counters for probe runs, failures, timeouts
- [x] 5.3 OpenTelemetry traces across scanner → probes (optional for MVP) — deferred; rationale in `docs/observability.md`

## 6. Housekeeping

- [x] 6.1 `Makefile` with `build`, `test`, `lint`, `run`
- [x] 6.2 `.golangci.yml` with staticcheck, errcheck, gofumpt
- [x] 6.3 `.github/workflows/ci.yml` — test + lint on push
- [x] 6.4 `docs/operator.md` — how to run a scan, interpret output
- [x] 6.5 Update `README.md` quickstart once CLI stabilises
