# Architecture

Wanderer is an observation engine. The MVP takes one input — a domain
name — and turns it into a collection of structured `Finding` records
covering four probe families. A future assessor turns those findings
into a DICTU sovereignty score; the MVP deliberately stops short of
scoring.

## Components

```
              ┌──────────────┐
 CLI ───────► │              │      ┌────────────┐
              │   scanner    │─────►│ probe/dns  │──► system / configured resolver
 API ───────► │ orchestrator │      ├────────────┤
              │              │─────►│ probe/tls  │──► target:443 + crt.sh (best-effort)
              │              │      ├────────────┤
              │              │─────►│ probe/ip   │──► local GeoLite2 DB
              │              │      ├────────────┤
              │              │─────►│ probe/http │──► target:80/443
              └──────┬───────┘      └────────────┘
                     │
                     ▼
              ┌─────────────┐
              │    store    │──► SQLite (modernc.org/sqlite, pure Go)
              └─────────────┘
```

Every probe implements the same interface (`internal/probe.Probe`) and
returns `[]models.Finding`. The scanner knows nothing about what any
specific probe does; the probes know nothing about each other.

## Data flow

1. Caller (CLI or HTTP API) hands the scanner a `models.Target`.
2. Scanner upserts the Target, creates a `running` `Scan` row.
3. For each configured probe, the scanner derives a child context with
   a per-probe timeout (default 30s), runs the probe, and persists its
   findings. Panics are recovered, timeouts are recorded as info
   findings — a failed probe never stops the scan.
4. Scanner finalises the `Scan` with terminal status:
   `complete` (all probes succeeded), `partial` (one or more produced
   errors but others produced findings), or `failed` (nothing usable).
5. Caller reads the final scan + findings back from the store.

## Key design decisions

### SQLite for the MVP

`modernc.org/sqlite` is pure Go (no CGo), so builds are trivial on any
platform and the DB is a single auditable file on disk. Migrating to
PostgreSQL later is a `pg_dump`-shaped problem. The alternative —
starting on Postgres "because we'll scale" — was rejected as premature.

### Probes are packages, not plugins

No plugin loader, no registry with `init()` side effects. `scanner` imports
each probe package and calls it. The call graph is static and readable
via `go doc`. Adding a probe means writing a package and wiring it into
`cmd/wanderer/scan.go#buildProbes`. Reflection-based dispatch can come
later when we actually need it.

### The Finding schema is the contract

Every probe returns `[]models.Finding` and nothing richer. The assessor
(future) reads `Finding.ProbeID`, `Finding.DimensionHint`, and
`Finding.Attributes` without ever importing a probe package. This
decoupling is the single most important choice: new probes do not
touch the assessor, and vice versa. See `pkg/models/finding.go` for
the shape, and `docs/findings.md` for the catalog.

### Partial scans are first-class

A probe that errors, panics, or times out is not a scan failure. The
scan records the failure as a `<probe>.error` / `.panic` / `.timeout`
finding and continues. An operator gets imperfect output rather than
nothing. A scan is `failed` only when **every** probe produced no
usable findings.

### Passive observation boundary

- DNS: lookups via the configured resolver.
- TLS: ClientHello + certificate inspection; no application data.
- HTTP: one GET of the apex URL, with a `Wanderer/0.x` User-Agent,
  honouring `robots.txt`, capping redirects at 5 and body at 2 MiB.
- IP: local database lookup, no network call.

No port scanning, no subdomain enumeration beyond what DNS and CT logs
volunteer, nothing credential-adjacent. This is not a pentest tool.

### External systems and their failure modes

| System              | Used by     | Failure handling |
| ------------------- | ----------- | ---------------- |
| DNS resolver        | `probe/dns` | NXDOMAIN / timeout / SERVFAIL → info Finding with `kind` attribute |
| Target `:443` TLS   | `probe/tls` | Handshake failure → retry with verification off and record verification failure as Finding |
| crt.sh              | `probe/tls` | Any failure → single "unavailable" Finding, rest of TLS probe unaffected |
| MaxMind GeoLite2    | `probe/ip`  | Missing/corrupt DB → **fail fast at startup**, never mid-scan |
| Target `:80/:443`   | `probe/http`| HTTP fallback if HTTPS fails (recorded as `http.scheme_downgrade`); body > 2 MiB is truncated |

## Extending the probe suite

To add a new probe:

1. Create `internal/probe/<name>/` with a type implementing
   `probe.Probe` (`ID() string` + `Run(ctx, target, cfg) ([]Finding, error)`).
2. Return findings with stable `ProbeID` strings (`<name>.<what>`).
   Put probe-specific structured data in `Attributes`. Raw source
   material (certificates, DNS record text) goes in `Evidence`.
3. Wire the probe into `cmd/wanderer/scan.go#buildProbes` (and into
   the serve path, which shares the same builder).
4. Document the new ProbeIDs in `docs/findings.md`.

That's the whole integration surface. No framework, no registry.

## Surfaces that are out of scope for the MVP

- Assessor (Finding → DICTU dimension/level) — ships as its own change.
- Scheduling + diffing between scans — one scan at a time, on command.
- Authentication — MVP is single-tenant, trusted-network.
- Web UI — JSON out only.
- JavaScript rendering for HTTP third-party extraction — static HTML
  only. A headless-browser probe is a plausible future change when the
  complexity/value ratio is right.
