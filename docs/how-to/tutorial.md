---
status: draft
last_reviewed: 2026-07-12
---

# Tutorial: your first Wanderer scan

This is a hands-on walkthrough. By the end you will have installed
Wanderer, run a scan against a domain you control, and understood what
each block of the output means. Budget: about 20 minutes.

## Who this is for

You are probably one of:

- a **compliance or information-security analyst** at a public-sector
  organisation, asked to produce sovereignty-posture evidence (the
  Dutch *Toetsingsinstrument Soevereiniteit Clouddiensten* is one
  example of the kind of artefact this evidence supports),
- a **platform operator** wanting to see what your org's public-facing
  systems look like from the outside,
- a **developer** extending Wanderer with a new probe or integrating it
  into a broader audit pipeline.

You do not need to be a DNS or TLS expert. You do need to be comfortable
on a Linux shell.

## 1. Install

Clone and build:

```sh
git clone https://github.com/MWest2020/wanderer.git
cd wanderer
make build
./bin/wanderer version
```

You should see a version string like `0.0.0-dev` or a git tag.

If you do not have Go 1.25 installed, either grab it from
<https://go.dev/dl/> or run Wanderer via your distro's Go package.

## 2. (Strongly recommended) get a GeoLite2 database

The IP probe resolves each IP address to an ASN and a country, which is
one of the most useful signals for answering "where does our data
live". That lookup is a local database query — Wanderer ships no IP
data itself.

For the short version: register at
<https://www.maxmind.com/en/geolite2/signup>, download
`GeoLite2-ASN.mmdb`, and put it somewhere stable
(`/var/lib/wanderer/GeoLite2-ASN.mmdb` or `~/wanderer-data/`).

For the long version — including the recommended `geoipupdate`
setup that keeps the database fresh, the optional country-only
file, and the opt-out for offline labs — see the
[GeoLite2 setup section in `docs/operator.md`](operator.md#geolite2-setup).

Without this database Wanderer still runs — the IP probe records a
single `ip.unavailable` finding and the rest of the scan proceeds —
but you will be missing the jurisdictional picture and the
**technologie** dimension of the assessment will stay `onbekend`.
The CLI prints a one-line warning to stderr at startup until you
either configure a database or pass `--no-geoip` to silence it.

## 3. Run your first scan

Pick a domain your organisation operates. Avoid scanning domains you do
not own or have permission to audit — Wanderer's traffic is benign, but
the courtesy rule applies.

```sh
./bin/wanderer scan example.nl \
  --geoip ~/wanderer-data/GeoLite2-ASN.mmdb
```

What happens:

1. Wanderer opens a SQLite database at `./wanderer.db` (created on
   first run).
2. It upserts your target into the `targets` table and creates a
   `scans` row in status `running`.
3. It runs the four probes in sequence (DNS, TLS, IP, HTTP). Each has
   a 30-second budget; the whole scan caps at 2 minutes.
4. It prints a human-readable summary and exits.

Exit codes: `0` = complete or partial (you got something useful),
`1` = failed (domain didn't resolve, or every probe failed),
`2` = bad arguments.

## 4. Read the output

The output is grouped by probe family. A typical block looks like:

```
== dns ==
  [info] dns.a  subject=example.nl
      address: 93.184.216.34
  [observation] dns.mx  subject=example.nl  dim=data_ai
      host: aspmx.l.google.com.
      preference: 10
  [observation] dns.ns  subject=example.nl  dim=operationeel
      host: ns1.cloud-supplier.example.
  [observation] dns.txt.spf  subject=example.nl  dim=data_ai
      record: v=spf1 include:_spf.google.com ~all
      kind: spf
```

Each finding has:

- **Severity** in square brackets: `info`, `observation`, `concern`,
  `finding`.
- **ProbeID** — a stable identifier. See
  [`docs/findings.md`](../reference/findings.md) for the full catalogue.
- **Subject** — what the finding is about. Usually the domain; for
  `http.third_party` findings it is the external host.
- **Dimension hint** (optional) — which sovereignty dimension this fact
  informs: `juridisch`, `technologie`, `data_ai`, `operationeel`,
  `mens`.
- **Attributes** — probe-specific structured data indented below.

### What to look at first

- **`tls.issuer`** — who signed the certificate? `issuer_country` and
  `issuer_o` are your jurisdictional signal. A US-based CA for a Dutch
  municipality's apex site is a fact the toets will care about.
- **`dns.mx` + `ip.asn`** — where does mail physically land? The MX
  host's ASN and country tell you whether outbound mail leaves the EU.
- **`http.third_party`** — which external hosts does the homepage
  load resources from? Combined with the `ip.asn` findings for those
  hosts, this is your vendor-dependency map.
- **`dns.ns`** — who runs your DNS? A single supplier with registrar
  lock can unilaterally affect continuity.

### Example narrative

A concrete reading of a scan might sound like:

> "Our apex domain resolves to one IPv4 and one IPv6 address, both in
> ASN 15169 (Google LLC, US). The cert was issued by a US CA
> (`issuer_country: ["US"]`). MX points at Microsoft 365 in ASN 8075
> (US). DNS is delegated to a Dutch provider (ASN 1136, NL). The
> homepage loads scripts from `cdn.cloudflare.com` (US) and
> `stats.vendor.eu` (NL)."

That paragraph is the start of a sovereignty story backed by evidence.

## 5. Inspect the database

Everything the CLI prints is also persisted. Any SQLite client works:

```sh
sqlite3 wanderer.db "SELECT probe_id, severity, subject FROM findings WHERE scan_id = (SELECT id FROM scans ORDER BY started_at DESC LIMIT 1);"
```

`Evidence` (cert PEM, verbatim DNS records) is in the `evidence` BLOB
column so you can audit later without re-scanning.

## 6. Try the HTTP API

For longer-running use or integration with other tooling, run the
server:

```sh
./bin/wanderer serve --addr :8080 \
  --geoip ~/wanderer-data/GeoLite2-ASN.mmdb
```

In another terminal:

```sh
# Kick off a scan (synchronous in the MVP — the response is the
# finished scan).
curl -sX POST http://localhost:8080/scans \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.nl"}' | jq .

# Fetch a stored scan by ID later.
curl -s http://localhost:8080/scans/s_1729...abc | jq .

# Liveness + metrics.
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/metrics | head
```

Errors come back as a structured JSON shape:

```json
{"error": {"code": "not_found", "message": "scan not found"}}
```

The MVP has no authentication. Do not expose port 8080 to the public
internet without a reverse proxy that enforces access control.

## 7. Scan a second domain

Run the scan again against a different domain you control, or the same
domain a day later. Each run produces a new Scan row; findings are
never overwritten. Diffing between runs is not yet built in — that
ships as a separate change.

## 8. What to do with the findings

At MVP stage, Wanderer produces **evidence**, not a score. A typical
workflow:

1. Run scans across your full domain portfolio, one at a time.
2. Export findings (the SQLite file, or the API) into whatever tool
   you use for sovereignty toetsing.
3. Use the catalogue in [`docs/findings.md`](../reference/findings.md) as the
   mapping key: each ProbeID already carries a `DimensionHint` for
   the sovereignty dimension it informs.
4. The wand and SEAL assessor packs already do steps 2–3 for you;
   `wanderer assess <scan-id> --framework both` produces an
   audit-ready Markdown / JSON report.

## Where to go next

- [`docs/operator.md`](operator.md) — full flag/env reference,
  troubleshooting.
- [`docs/findings.md`](../reference/findings.md) — every ProbeID with attribute
  shapes.
- [`docs/architecture.md`](../explanation/architecture.md) — how the components fit
  together, how to add a probe.
- [`docs/observability.md`](../reference/observability.md) — logs, metrics, traces.

## Common gotchas

- **`ip: asn DB: no such file or directory`** — the GeoLite2 path is
  wrong. Wanderer refuses to start rather than silently running
  blind; fix the path.
- **`tls.ct` shows `unavailable`** — crt.sh rate-limited you or
  returned an error. The rest of the TLS probe is unaffected; retry
  later or point `--user-agent` at something more polite.
- **Only one `http.response` finding even on a weird site** — Wanderer
  does not execute JavaScript. Dynamically injected trackers are
  invisible in the MVP. Headless-browser rendering is a plausible
  future change.
- **No findings at all** — the domain might not have resolved. Check
  for `dns.a` with `kind: "nxdomain"` and the scan's terminal status.
