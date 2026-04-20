# Design: MVP Scanner Suite

## Component overview

```
              ┌──────────────┐
 CLI ───────► │              │      ┌────────────┐
              │   scanner    │─────►│ probe/dns  │──► system / configured resolver
 API ───────► │ orchestrator │      ├────────────┤
              │              │─────►│ probe/tls  │──► target:443
              │              │      ├────────────┤
              │              │─────►│ probe/ip   │──► local GeoLite2 DB
              │              │      ├────────────┤
              │              │─────►│ probe/http │──► target:80/443 + CT logs
              └──────┬───────┘      └────────────┘
                     │
                     ▼
              ┌─────────────┐
              │    store    │──► SQLite (modernc.org/sqlite, pure Go)
              └─────────────┘
```

## External systems and their failure modes

| System              | Used by     | Failure modes we handle                                                        |
| ------------------- | ----------- | ------------------------------------------------------------------------------ |
| DNS resolver        | `probe/dns` | Timeout, NXDOMAIN, SERVFAIL, truncated responses, resolver returning `127.0.0.1` |
| Target `:443` TLS   | `probe/tls` | Connection refused, handshake timeout, cert expired, hostname mismatch, chain-build failure |
| Target `:80/:443`   | `probe/http`| Redirect loop (cap at 5), non-200, body > 2 MiB, mixed HTTP/HTTPS content     |
| crt.sh (CT mirror)  | `probe/tls` | Rate-limited (HTTP 429), timeout, JSON schema change — all degrade gracefully: the probe records "CT lookup unavailable" and continues |
| MaxMind GeoLite2    | `probe/ip`  | DB file missing/corrupt at startup — fail fast with a clear error, never mid-scan |

## Key decisions

### SQLite for the MVP

`modernc.org/sqlite` is pure-Go, no CGo, embeds cleanly, and the schema we
need (targets, scans, findings) is boring enough that migrating to
PostgreSQL later is a `pg_dump`-equivalent away. The clever valkuil here
would be to start on Postgres because "we'll scale"; we won't, not for a
while, and a SQLite file is trivially auditable on-disk.

### Probes are packages, not plugins

No plugin loader, no registry with `init()` side effects. The scanner
imports each probe package explicitly and calls it. This makes the call
graph static and readable — `go doc` tells you what runs. The valkuil
avoided: a "probe framework" with reflection-based dispatch that lets
anyone ship a probe. We don't need that yet, and we'll wire it in when we
do.

### Findings schema is the contract

Every probe returns `[]models.Finding`. A Finding has:

- `ProbeID` — stable string, e.g. `dns.mx`, `tls.issuer`, `http.third_party`.
- `DimensionHint` — one of the DICTU dimensions, or empty if the finding is
  raw observation with no dimensional meaning on its own.
- `CriteriumHint` — optional DICTU criterium reference.
- `Subject` — the thing being described (a domain, an IP, a host).
- `Attributes` — `map[string]any` for probe-specific structured data.
- `Severity` — `info | observation | concern | finding`.
- `Evidence` — raw data (e.g. the TXT record verbatim, the cert PEM) so the
  assessor and human reviewers can audit without re-scanning.

Scoring will be a separate component that consumes Findings and knows
nothing about how they were produced. This is the single most important
design choice — it is what lets us add probes without touching the
assessor, and vice versa.

### Context and timeouts

The scanner owns a root `context.Context` with a global budget (default 2
min per scan). Each probe gets a derived child context with a per-probe
budget (default 30s). A probe that hits its timeout returns a Finding of
severity `info` with `timeout: true` in Attributes — the scan continues.
This is what makes a partial scan a first-class result rather than an
error.

### HTTP third-party extraction

Parse the fetched HTML with `golang.org/x/net/html`, walk the node tree,
collect `src`/`href` attributes from `<script>`, `<link>`, `<img>`,
`<iframe>`, `<source>`. Resolve each external host through `probe/ip`.
We do not execute JavaScript — a dynamically injected tag tracker is
invisible to us in the MVP, and that is explicitly a known limitation.
Headless browser rendering is a plausible future change but the
complexity/value ratio for the MVP is wrong.

## Data model

```sql
CREATE TABLE targets (
  id         TEXT PRIMARY KEY,      -- ULID
  domain     TEXT NOT NULL UNIQUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE scans (
  id         TEXT PRIMARY KEY,      -- ULID
  target_id  TEXT NOT NULL REFERENCES targets(id),
  started_at DATETIME NOT NULL,
  ended_at   DATETIME,
  status     TEXT NOT NULL,         -- running | complete | partial | failed
  error      TEXT
);

CREATE TABLE findings (
  id             TEXT PRIMARY KEY,  -- ULID
  scan_id        TEXT NOT NULL REFERENCES scans(id),
  probe_id       TEXT NOT NULL,
  dimension_hint TEXT,
  criterium_hint TEXT,
  subject        TEXT NOT NULL,
  severity       TEXT NOT NULL,
  attributes     TEXT NOT NULL,     -- JSON
  evidence       BLOB,
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_findings_scan ON findings(scan_id);
CREATE INDEX idx_findings_probe ON findings(probe_id);
```

## Testing strategy

- Each probe has a `testdata/` directory with recorded fixtures for its
  external calls. DNS fixtures are handled via a fake resolver
  (`net.DefaultResolver` is swappable at test time via `Resolver` interface
  we define). TLS uses `httptest.NewTLSServer` for positive cases and
  deliberately-broken certs in `testdata/` for negative cases.
- Golden files for the `wanderer scan` CLI output so regressions in
  human-readable output are visible in diffs.
- No live network tests in CI. Integration tests against real domains
  live under `//go:build integration` and run opt-in.
