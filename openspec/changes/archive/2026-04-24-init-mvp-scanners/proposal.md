# Proposal: Initialise MVP Scanner Suite

## Intent

Wanderer starts with nothing. This change lays down the foundational scanner
suite that can take a single input — a domain name operated by a
public-sector organisation — and produce a structured set of findings
across the four observation categories that collectively cover most of the
DICTU Toetsingsinstrument's technically observable signals.

The goal of the MVP is **not** a polished product. It is: given `example.nl`,
does Wanderer produce evidence that a human operator would recognise as
"yes, this tells me something real about our sovereignty position."

## Scope

**In scope:**

- A target model (a domain, optionally with a list of related domains).
- Four probe families, each producing `Finding` records with a shared schema:
  - `probe/dns` — A/AAAA, MX, NS, CNAME, TXT (SPF/DKIM/DMARC policy records),
    CAA.
  - `probe/tls` — certificate chain for HTTPS on the apex and www, issuer
    identity, SANs, validity, matched against Certificate Transparency logs
    (via crt.sh or a local CT mirror).
  - `probe/ip` — resolve A/AAAA to ASN and announced country via a local
    MaxMind GeoLite2-ASN + Country database (bundled, no runtime API).
  - `probe/http` — fetch the apex homepage, record response headers, extract
    third-party resources (scripts, stylesheets, fonts, images by host),
    resolve each third-party host through the same IP probe.
- A `scanner` orchestrator that sequences the probes with a shared context
  and timeout budget, persists findings per scan.
- SQLite persistence for targets, scans, and findings.
- A CLI entry point: `wanderer scan example.nl` prints a human-readable
  summary and exits non-zero if the scan did not complete.
- A minimal HTTP API: `POST /scans` (enqueue), `GET /scans/{id}` (status +
  findings as JSON).

**Out of scope for this change:**

- The assessor (mapping findings → DICTU dimensions/levels) ships as its own
  change after the findings schema has settled under real data.
- Scheduling and diffing between scans. The MVP runs one scan at a time on
  command.
- Web UI. JSON out only.
- Authentication on the HTTP API. The MVP is single-tenant and trusted-
  network; auth is a separate change.
- Supplementary probes: WHOIS, mail-server TLS (STARTTLS), IPv6-only
  verification, reverse-DNS sweeps.

## DICTU dimensions informed

- **Juridisch** — IP→country and TLS issuer identity feed the jurisdictional
  question (does data transit a non-EU provider, who issued the cert).
- **Technologie** — third-party HTTP resources and their hosts feed the
  vendor-lockin question (how many non-EU dependencies does the public-facing
  site load).
- **Data & AI** — CNAME/MX targets feed the data-residency question (where
  does mail actually land).
- **Operationeel** — NS delegation and cert issuance reveal who can
  unilaterally affect continuity (DNS provider, CA).
- **Mens** — out of technical scope; not addressed in this MVP.

## Passive/active boundary

All four probes are **passive** in the sense that:

- DNS queries go to the configured resolver (default: system resolver).
- TLS handshakes terminate at ClientHello + certificate inspection; no
  application data is sent.
- HTTP fetches request only the apex URL with a standard `User-Agent: Wanderer/0.x`
  and honour `robots.txt`. No crawling beyond the single page.
- IP→ASN is a local database lookup, no network call.

There is no scanning of ports, no enumeration of subdomains beyond what DNS
and CT volunteer, no credential-adjacent behaviour.

## Approach

Each probe is a package in `internal/probe/` exposing a single function
with signature

```go
Run(ctx context.Context, target models.Target, cfg Config) ([]models.Finding, error)
```

The scanner composes them; each probe is independently testable with
recorded fixtures. The findings schema is the contract — it is defined once
in `pkg/models/finding.go` and every probe writes into it. Scoring never
knows which probe produced a finding, only what dimension and criterium
the finding addresses.

See [design.md](design.md) for the full technical approach, and
[tasks.md](tasks.md) for the implementation checklist.
