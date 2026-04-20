# Wanderer

> Sovereignty checker for larger organisations that lost their way.

**Wanderer** is an automated digital sovereignty monitor for public-sector
organisations. It continuously maps an organisation's actual digital footprint
— DNS, MX, TLS, IP/ASN, HTTP third parties — and scores those findings against
the [DICTU Toetsingsinstrument Soevereiniteit Clouddiensten][dictu] so that
"how sovereign are we?" becomes a question you can answer with evidence instead
of a form.

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
- An assessor. Findings are mapped to the DICTU dimensions and levels to
  produce an evidence-backed sovereignty profile.
- A living picture. Scans run on a schedule; the profile changes when reality
  does.

**Isn't:**

- A pentest tool. Wanderer only looks at externally observable signals, it does
  not probe internal systems or attempt exploitation.
- A replacement for the DICTU toets. The toets asks the policy questions;
  Wanderer supplies the technical evidence that informs the answers.
- A commercial SaaS replacement for Shodan or SecurityTrails. The intent is
  deployable inside the organisation, auditable, and free of vendor lock-in.

## Status

Greenfield. The OpenSpec change [`init-mvp-scanners`](openspec/changes/init-mvp-scanners/)
defines the MVP: DNS/MX scanners, TLS + certificate transparency, IP→ASN→
jurisdiction resolution, and HTTP header / third-party resource discovery.

## Layout

```
cmd/wanderer/            # CLI + server entry point
internal/
  scanner/              # Orchestration — takes a target, runs probes
  probe/                # Individual probes (dns, tls, http, ip)
  assessor/             # Maps findings to DICTU dimensions/levels
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
