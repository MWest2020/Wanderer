# Operator Guide

This document describes how to run Wanderer, interpret its output, and
operate the single-tenant HTTP API. It covers the MVP only —
scheduling, authentication, and the DICTU assessor ship as separate
changes.

## Install

Build from source:

```sh
make build
./bin/wanderer version
```

Or run directly with `go run`:

```sh
go run ./cmd/wanderer scan example.nl
```

## Prerequisites

The IP probe needs a **MaxMind GeoLite2-ASN** database to resolve IPs
to ASN + country. Without it, Wanderer still runs the DNS, TLS, and
HTTP probes; the IP probe records a single "unavailable" finding and
continues.

- Register at <https://www.maxmind.com/en/geolite2/signup> and download
  `GeoLite2-ASN.mmdb` (and optionally `GeoLite2-Country.mmdb`).
- Pass the path via `--geoip` on the CLI or `WANDERER_GEOIP_ASN` env
  var on `wanderer serve`.

If the database file is missing or corrupt, Wanderer fails **fast at
startup**, never mid-scan.

## Run a single scan

```sh
wanderer scan example.nl \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb \
  --db ./wanderer.db
```

Exit codes:

- `0` — scan completed or completed partially (at least one probe
  produced findings)
- `1` — scan failed (domain did not resolve, every probe failed, or
  the store was unreachable)
- `2` — invalid arguments

Output is grouped by probe. Example:

```
Scan s_1729...  status=complete

== dns ==
  [info] dns.a  subject=example.nl
      address: 93.184.216.34
  [observation] dns.mx  subject=example.nl  dim=data_ai
      host: mail.example.nl.
      preference: 10
...
```

Persisted findings live in the SQLite database (`./wanderer.db` by
default). You can inspect them with any SQLite client:

```sh
sqlite3 wanderer.db "SELECT probe_id, severity, subject FROM findings ORDER BY created_at DESC LIMIT 20;"
```

## Interpret the output

Every finding carries:

- `ProbeID` — stable identifier, e.g. `tls.issuer`, `http.third_party`.
- `Severity` — `info`, `observation`, `concern`, `finding`. This is
  deliberately coarse; precise scoring is the assessor's job.
- `DimensionHint` — which DICTU dimension the finding informs, if any.
- `Subject` — the thing being described (a domain, a host, an IP).
- `Attributes` — probe-specific structured data.
- `Evidence` — raw source material (certificate PEM, verbatim DNS
  record) retained so a reviewer can audit without re-scanning.

Interesting starting points when looking at a fresh scan:

| Question                                     | Probe/Finding to check        |
| -------------------------------------------- | ----------------------------- |
| Where does mail land?                        | `dns.mx` hosts + `ip.asn`     |
| Who issued the cert?                         | `tls.issuer` (dim: juridisch) |
| Which non-EU providers serve resources?      | `http.third_party` + `ip.asn` |
| Who controls DNS continuity?                 | `dns.ns`                      |
| Is HTTPS configured correctly?               | `tls.validity`, `tls.verify`  |

## Serve the HTTP API

```sh
wanderer serve \
  --addr :8080 \
  --db /var/lib/wanderer/wanderer.db \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb
```

Endpoints:

- `POST /scans` — body `{"domain":"example.nl","related":["..."]}`.
  Returns the finished Scan record (synchronous in the MVP).
- `GET /scans/{id}` — returns a stored Scan with all findings.
- `GET /healthz` — liveness probe.
- `GET /metrics` — Prometheus counters (see `docs/observability.md`).

All errors use a structured shape:

```json
{"error":{"code":"not_found","message":"scan not found"}}
```

**No authentication**. The MVP is intended for a trusted network (for
example, run inside a Nextcloud stack alongside OpenConnector). Adding
auth is a separate change — do not expose this port to the internet
without a reverse proxy enforcing access control.

## Configuration summary

| Flag                     | Env var                     | Default               |
| ------------------------ | --------------------------- | --------------------- |
| `--addr`                 | `WANDERER_LISTEN`           | `:8080`               |
| `--db`                   | `WANDERER_DB`               | `./wanderer.db`       |
| `--geoip`                | `WANDERER_GEOIP_ASN`        | (empty)               |
| `--geoip-country`        | `WANDERER_GEOIP_COUNTRY`    | (fall back to ASN DB) |
| `--per-probe-timeout`    | —                           | `30s`                 |
| `--budget`               | —                           | `2m`                  |
| `--user-agent`           | —                           | `Wanderer/0.x`        |

## Troubleshooting

- **`ip: asn DB: no such file or directory`** — the GeoLite2 path is
  wrong. Fail-fast is intentional; the probe will not start without a
  readable database file.
- **`tls.ct` shows `unavailable`** — crt.sh rate-limited you or was
  unreachable. The rest of the TLS probe is unaffected.
- **`http.robots_blocked`** — the target's `robots.txt` disallows `/`
  for your User-Agent. Wanderer honours it; use a different User-Agent
  if you have authorisation.
