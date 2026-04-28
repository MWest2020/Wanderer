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

### Two-pass scanning

The scanner runs every scan in two passes:

1. **Pass 1** runs DNS, TLS, HTTP and (when configured) the WHOIS
   probe concurrently via `errgroup.WithContext`. Each probe sees the
   original `Target` (apex domain plus any operator-supplied
   `Related` hosts). A per-probe `defer recover()` converts panics
   into `nil`-from-the-group errors so one misbehaving probe cannot
   poison the whole scan.
2. **Pass 2** runs only the IP probe. Between the passes,
   `expandRelatedFromFindings` walks the pass-1 Findings and harvests
   subjects from `dns.mx`, `http.third_party` and `dns.subdomain`
   into a copy of the Target's `Related` slice. The IP probe receives
   that enriched Target so MX hosts and third-party hostnames get
   ASN/country lookups. Other probes still see the original Target;
   the enrichment is local to the IP probe.

`buildProbes` in `cmd/wanderer/scan.go` enforces the ordering by
returning the IP probe last. The whole scan still runs under one
`context.WithTimeout` so the global budget remains a single dial.
`TestIPProbeReceivesDiscoveredHosts` in `internal/scanner` pins the
fan-out invariant. Without this two-pass shape, the rules
`dictu.juridisch.mx_vendor_jurisdiction`,
`dictu.technologie.third_parties_eea` and the third-party half of
`dictu.technologie.no_us_hyperscaler` silently returned Onbekend on
every real scan.

### SSRF guard

`internal/probe/ssrf.go` wraps the dialers used by the HTTP and TLS
probes. A static `*net.IPNet` table covers IPv4 loopback, link-local,
RFC1918, CGNAT, and the cloud-metadata IPs (169.254.169.254,
fd00:ec2::254), plus IPv6 ULA and link-local. Any resolved address
that lands in one of those nets is refused at dial time. Operators
who do need to scan a private host pass `--allow-private-targets`
on `wanderer scan` or `wanderer serve`; default is on (private
blocked). The `POST /scans` handler refuses requests whose domain
resolves only to private addresses unless the flag was set at
server start, so an authenticated API client cannot turn the scanner
into an internal-network probe.

### Framework selection

The assessor supports two rule packs side-by-side:

- **DICTU** — Dutch government sovereignty toets. Maps Findings into
  five dimensions (`juridisch`, `operationeel`, `technologie`,
  `data_ai`, `mens`) and four levels (`onbekend`/`afhankelijk`/
  `gedeeld`/`soeverein`). Lives in `internal/assessor/dictu`.
- **EU CSF (SEAL)** — five SEAL levels (SEAL0–SEAL4) over the same
  Findings. Lives in `internal/assessor/eucsf`. SEAL0 = no evidence,
  SEAL4 = full sovereignty.

Both packs read the same `[]models.Finding` slice and emit
`models.Assessment` records. `wanderer assess --framework
dictu|eucsf|both` selects which pack(s) run; the persistence layer
keeps assessments tagged with `Framework`. The two packs share no
code beyond the Finding shape — adding a third pack means adding a
new package, not modifying either of the existing two.
`docs/assessor.md` is the operator-facing reference.

### Read-only operator UI

`internal/ui` ships a minimal web UI mounted on `wanderer serve` under
`--ui` (default off). Three GET-only routes — `/ui/`, `/ui/scans/{id}`,
`/ui/targets/{id}/drift` — render `html/template` pages backed by
embedded templates and a single CSS file (`go:embed`). The package
contains zero mutating handlers; `ui_test.go` greps the package source
for `r.Post|Patch|Delete|Put` and fails the build if any appear, so
the read-only invariant cannot be bypassed by accident in a future
patch.

Authentication is HTTP Basic against an htpasswd file
(`--ui-htpasswd <path>` or `WANDERER_UI_HTPASSWD`). Only bcrypt
entries (`$2a$`/`$2b$`/`$2y$`) are accepted; `$apr1$` MD5, `{SHA}`
SHA-1, `$5$` SHA-256 crypt and `$6$` SHA-512 crypt are rejected at
startup with an explicit "use bcrypt (`htpasswd -B`)" error. One
algorithm = one battle-tested verification path; an operator who
copies a legacy file gets a hard failure instead of a silent
always-deny. The htpasswd file is re-read on every request so
operators can rotate credentials without restarting.

The UI is intentionally a thin observation surface: no session state,
no JWT, no OAuth. Anything fancier belongs behind a reverse proxy.

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
- Multi-tenant authentication — `wanderer serve` is single-tenant; the
  optional `/ui/` surface uses HTTP Basic via htpasswd, and the JSON
  API itself remains trusted-network.
- Rich web UI — `/ui/` is intentionally a thin read-only HTML surface;
  anything richer (mutations, dashboards, multi-user) belongs behind
  a reverse proxy or in a separate frontend.
- JavaScript rendering for HTTP third-party extraction — static HTML
  only. A headless-browser probe is a plausible future change when the
  complexity/value ratio is right.
