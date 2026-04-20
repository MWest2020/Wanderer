# Tasks: Initialise MVP Scanner Suite

## 1. Foundations

- [ ] 1.1 `pkg/models/finding.go` — Finding, Severity, DimensionHint enums
- [ ] 1.2 `pkg/models/target.go` — Target type with validation
- [ ] 1.3 `pkg/models/scan.go` — Scan lifecycle states
- [ ] 1.4 `internal/store/sqlite.go` — open DB, migrate schema at startup
- [ ] 1.5 `internal/store/store_test.go` — round-trip Finding serialisation

## 2. Probes

- [ ] 2.1 `internal/probe/probe.go` — shared `Probe` interface and `Config`
- [ ] 2.2 `internal/probe/dns/` — A/AAAA, MX, NS, CNAME, TXT, CAA
- [ ] 2.3 `internal/probe/dns` fixtures + unit tests with fake resolver
- [ ] 2.4 `internal/probe/tls/` — chain inspection, issuer, SAN, validity
- [ ] 2.5 `internal/probe/tls` CT-log lookup via crt.sh JSON endpoint with graceful degradation
- [ ] 2.6 `internal/probe/ip/` — GeoLite2 loader, lookup, country + ASN
- [ ] 2.7 `internal/probe/ip` — fail-fast on missing/corrupt DB at startup
- [ ] 2.8 `internal/probe/http/` — fetch apex, parse HTML, extract third parties
- [ ] 2.9 `internal/probe/http` — redirect cap, body cap, robots.txt honour

## 3. Orchestration

- [ ] 3.1 `internal/scanner/scanner.go` — sequences probes, shared context
- [ ] 3.2 `internal/scanner` — per-probe timeout, global budget, partial-scan result
- [ ] 3.3 `internal/scanner/scanner_test.go` — one probe fails, scan still completes

## 4. Interfaces

- [ ] 4.1 `cmd/wanderer/main.go` — root command with subcommands
- [ ] 4.2 `cmd/wanderer/scan.go` — `wanderer scan <domain>` with human-readable output
- [ ] 4.3 `cmd/wanderer/serve.go` — `wanderer serve` starts the HTTP API
- [ ] 4.4 `internal/api/api.go` — chi router, `POST /scans`, `GET /scans/{id}`
- [ ] 4.5 `internal/api` — structured JSON errors, health endpoint

## 5. Observability

- [ ] 5.1 `log/slog` JSON handler wired into scanner + probes, request-id propagated through context
- [ ] 5.2 Prometheus counters for probe runs, failures, timeouts
- [ ] 5.3 OpenTelemetry traces across scanner → probes (optional for MVP)

## 6. Housekeeping

- [ ] 6.1 `Makefile` with `build`, `test`, `lint`, `run`
- [ ] 6.2 `.golangci.yml` with staticcheck, errcheck, gofumpt
- [ ] 6.3 `.github/workflows/ci.yml` — test + lint on push
- [ ] 6.4 `docs/operator.md` — how to run a scan, interpret output
- [ ] 6.5 Update `README.md` quickstart once CLI stabilises
