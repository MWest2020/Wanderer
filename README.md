# Wanderer

> Sovereignty checker for larger organisations that lost their way.

**Wanderer** is an automated digital sovereignty monitor for public-sector
organisations. It continuously maps an organisation's actual digital footprint
— DNS, MX, TLS, IP/ASN, HTTP third parties — and scores those findings against
its own **wand** rule pack (Wanderer-NL — inspired by the
[DICTU Toetsingsinstrument Soevereiniteit Clouddiensten][dictu], independent
implementation) so that "how sovereign are we?" becomes a question you can
answer with evidence instead of a form. The DICTU framework is credited as
the public source of inspiration; Conduction owns and maintains the wand
implementation. See [ADR-0011](docs/decisions/0011-rename-dictu-to-wand.md).

[dictu]: https://www.dictu.nl/documenten/publicaties/2025/toetsingsinstrument-soevereiniteit-clouddiensten

## Why

The DICTU toetsingsinstrument and derivatives like
[soevereiniteitstoets.nl][toets] work on **declared** cloud services: someone
picks a vendor, walks through five dimensions and fifteen criteria, produces a
score. That's useful, but it has two limits: it's a snapshot, and it assumes
the operator already knows which services are in play. In practice,
organisations don't. Every municipality has SaaS that crept in via a pilot, an
MX record pointing somewhere nobody documented, a CDN chosen by a supplier, a
login form that quietly federates to a US identity provider.

[toets]: https://soevereiniteitstoets.nl

Wanderer closes that gap from the other direction. Start with what an
organisation **actually operates** — domains, websites, mail flows, certificate
chains — and walk the graph outward until the picture is complete. Then score.

## What it is (and isn't)

**Is:**

- An observation engine. Scanners probe public signals (DNS, TLS, HTTP, WHOIS,
  IP/ASN) and record raw findings.
- An assessor. Findings are mapped to the wand dimensions and levels to
  produce an evidence-backed sovereignty profile.
- A living picture. Scans run on a schedule; the profile changes when reality
  does.

**Isn't:**

- A pentest tool. Wanderer only looks at externally observable signals, it does
  not probe internal systems or attempt exploitation.
- A replacement for the DICTU toets. The toets asks the policy questions;
  Wanderer's wand pack supplies the technical evidence that informs the
  answers.
- A commercial SaaS replacement for Shodan or SecurityTrails. The intent is
  deployable inside the organisation, auditable, and free of vendor lock-in.

## Status

Wanderer covers three observation modi end-to-end:

- **Perimeter** — DNS (A/AAAA/MX/NS/CNAME/TXT/CAA), TLS + Certificate
  Transparency via crt.sh, passive subdomain discovery (CT-log SANs +
  prefix sweep + optional Amass import), IP→ASN→country via a local
  GeoLite2 database, HTTP header / third-party resource discovery,
  and RDAP/WHOIS for registrant + registrar jurisdiction. The
  scanner runs in two passes (concurrent fan-out for pass 1, IP
  probe in pass 2 with hosts discovered by the others) under a
  single global timeout, with an SSRF guard that refuses private and
  cloud-metadata addresses unless explicitly allowed.
- **Inventory** — `wanderer agent` reports host-side findings
  (systemd, dpkg/rpm, Nextcloud opt-in, Docker placeholder) to a
  central core via HMAC-signed HTTPS, or writes them straight into a
  shared SQLite file in local mode.
- **Egress** — agent-side classification of where data leaves the
  host: configured config files, `/proc/<pid>/environ`, and systemd
  unit files are scanned for object-storage / database / SMTP /
  OIDC / log-shipper / webhook destinations, with optional GeoLite2
  jurisdiction annotation and a redactor in front of every value
  emission.

Findings persist to SQLite and are retrievable via a JSON HTTP API
or a read-only HTML UI (`wanderer serve --ui`, htpasswd-protected).

Two assessor packs ship in the same binary:

- **wand** (Wanderer-NL) — Dutch sovereignty rule pack, four levels
  over five dimensions, inspired by the DICTU
  *Toetsingsinstrument Soevereiniteit Clouddiensten*. See
  [`docs/assessor.md`](docs/assessor.md) and
  [ADR-0011](docs/decisions/0011-rename-dictu-to-wand.md).
- **EU CSF / SEAL** — five SEAL levels (SEAL0–SEAL4) over the same
  Findings; selected via `wanderer assess --framework eucsf|both`.

## Quickstart

```sh
make build
./bin/wanderer scan example.nl --geoip /path/to/GeoLite2-ASN.mmdb
```

The database is an ordinary SQLite file (`./wanderer.db` by default).
See [`docs/operator.md`](docs/operator.md) for the full operator guide.

To run the HTTP API:

```sh
./bin/wanderer serve --addr :8080 --geoip /path/to/GeoLite2-ASN.mmdb
curl -X POST http://localhost:8080/scans -d '{"domain":"example.nl"}'
```

## Layout

```
cmd/wanderer/            # CLI + server entry point
internal/
  scanner/              # Orchestration — takes a target, runs probes
  probe/                # Individual probes (dns, tls, http, ip)
  assessor/             # Maps findings to sovereignty dimensions/levels (wand + SEAL)
  store/                # Persistence (findings, scans, targets)
  api/                  # HTTP API
pkg/models/             # Shared types, exportable
openspec/               # Spec-driven development artefacts
docs/                   # Design notes, operator guide
```

## License

[EUPL-1.2](LICENSE) — compatible with Conduction's other open-source components.

## Name

Large organisations rarely lose data all at once. They lose *track* of it —
piece by piece, supplier by supplier, pilot by pilot. Wanderer is for the
organisations that need to find their way back.
