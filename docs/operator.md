# Operator Guide

This document describes how to run Wanderer, interpret its output, and
operate the single-tenant HTTP API. It covers the MVP only —
scheduling, authentication, and the wand assessor ship as separate
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

## GeoLite2 setup

The IP probe resolves observed IPs to ASN + country via MaxMind's
GeoLite2 database. This is the input that unblocks the wand
**technologie** dimension and most of the **juridisch** dimension —
without it, every rule that depends on `ip.asn` returns `onbekend`,
and an operator looking at the assessment sees a half-blank picture.

GeoLite2 is **free** but the licence forbids redistribution, so
Wanderer cannot ship it; you fetch it once and keep it fresh.

### One-time setup

1. Sign up at <https://www.maxmind.com/en/geolite2/signup>
   (free MaxMind account).
2. From your account portal, generate a **license key**.
3. Download `GeoLite2-ASN.mmdb` (and optionally
   `GeoLite2-Country.mmdb`) — either via the portal's manual
   download, or via `geoipupdate` (recommended for hosts that
   should stay current).

### Paths Wanderer expects

| Env var                     | CLI flag         | Required for     |
| --------------------------- | ---------------- | ---------------- |
| `WANDERER_GEOIP_ASN`        | `--geoip`        | ASN + country    |
| `WANDERER_GEOIP_COUNTRY`    | `--geoip-country`| Country only (optional; defaults to the ASN file) |

The flags apply to `wanderer scan`, `wanderer serve`, and the
agent (where wired into the egress probe). When neither is set
and the opt-out below is also absent, both `wanderer scan` and
`wanderer serve` emit one warning to stderr at startup naming
the missing flag and pointing at this guide. The scan still
completes; the warning is informational.

```
warning: GeoLite2 ASN database is not configured — scan will
continue with reduced assessment coverage. Pass --geoip <path>
(or set WANDERER_GEOIP_ASN), or pass --no-geoip to silence
this warning. See docs/operator.md for setup.
```

If the database file is configured but missing or corrupt at
runtime, Wanderer fails **fast at startup**, never mid-scan.

### Opt-out for offline labs and CI

Hosts that consciously run without ASN annotation (offline audit
labs, CI smoke tests, demos) silence the warning with either:

- the `--no-geoip` CLI flag, or
- the `WANDERER_GEOIP_OPTIONAL=1` environment variable.

Neither changes the runtime behaviour — the IP probe still emits
its single `ip.unavailable` info Finding and the rest of the scan
continues — the opt-out only suppresses the startup warning.

### Recommended: keep GeoLite2 fresh with `geoipupdate`

MaxMind ships a small daemon
([`geoipupdate`](https://github.com/maxmind/geoipupdate)) that
re-downloads the database on a schedule. Wanderer reads whatever
file is at the configured path, so a periodic `geoipupdate` run
followed by a `wanderer serve` SIGHUP (or process restart) is the
operational shape we recommend.

Minimal `/etc/GeoIP.conf` (the daemon's config, not Wanderer's):

```conf
AccountID YOUR_ACCOUNT_ID
LicenseKey YOUR_LICENSE_KEY
EditionIDs GeoLite2-ASN GeoLite2-Country
```

Systemd timer (preferred over crontab on systemd hosts):

```ini
# /etc/systemd/system/geoipupdate.service
[Unit]
Description=Update MaxMind GeoIP databases
[Service]
Type=oneshot
ExecStart=/usr/bin/geoipupdate
```

```ini
# /etc/systemd/system/geoipupdate.timer
[Unit]
Description=Daily geoipupdate
[Timer]
OnCalendar=daily
Persistent=true
[Install]
WantedBy=timers.target
```

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now geoipupdate.timer
```

Crontab alternative (for non-systemd hosts):

```cron
# 03:00 every day, append output to a log
0 3 * * * /usr/bin/geoipupdate >> /var/log/geoipupdate.log 2>&1
```

### Test stub for offline runs

`scripts/geoip-stub.sh /tmp/stub.mmdb` produces a deterministic
empty-but-valid GeoLite2-shaped mmdb so the test suite (and an
operator running smoke tests) can exercise the populated-but-empty
branch of the IP probe without a real MaxMind license. The stub
opens cleanly via the same reader Wanderer uses; every IP lookup
returns "not found" rather than erroring out.

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

### Pair with Amass for richer subdomain coverage

Wanderer's built-in subdomain discovery is intentionally light: SAN
mining from the apex certificate plus a fixed prefix sweep. For a
broader picture without bolting an active enumerator into the scanner
itself, run [`amass`](https://github.com/owasp-amass/amass) once and
feed the result in:

```sh
amass enum -passive -d example.nl -json amass.json
wanderer scan example.nl \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb \
  --amass amass.json \
  --db ./wanderer.db
```

`internal/scanner/amass.go` parses the JSONL produced by `amass enum
-json` and merges the FQDNs into `target.Related` so the IP probe
resolves them in pass 2 and the assessor sees them as third parties.
The same mechanism is available over the API: `POST /scans` accepts an
`amass_json` field carrying a server-local file path (the serve
endpoint refuses inline file bodies — keep the file on the box).

CLI failures during Amass parsing are fatal at startup, not silent.
A malformed JSON file aborts before the scan begins; a missing path
errors immediately. This is deliberate — silent fallthrough on a
flag that names a file is the kind of thing that makes a scan look
fine while being effectively unenriched.

## Interpret the output

Every finding carries:

- `ProbeID` — stable identifier, e.g. `tls.issuer`, `http.third_party`.
- `Severity` — `info`, `observation`, `concern`, `finding`. This is
  deliberately coarse; precise scoring is the assessor's job.
- `DimensionHint` — which sovereignty dimension the finding informs, if any.
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
