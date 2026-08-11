# Research: high-signal observability — make Wanderer's output speak

> **Status:** Research / direction-finding. This is an umbrella +
> prioritisation doc, not a buildable spec. Each accepted lead spawns
> its own `propose-*` change (the sibling `propose-transit-path-probe`
> is the first, and the template).

## Why

Feedback from the field: Wanderer's output felt **weinigzeggend** — a
lot of own-framework scoring that states a verdict without showing the
observed reality behind it. The fix is not more scores; it is more
**concrete, self-evident, observed signals**. A traceroute that shows
"your Nextcloud lands at CYSO in NL via a JP carrier" needs no
framework to be understood — and *then* the rule pack can annotate it.

The product thesis: **the organisation/host is the spil in the web, and
the operator must see what goes where, and what is used (or misused)
where.** Inbound, outbound, transit — laid out, not abstracted.

## What Changes

Nothing ships from this doc directly. It:
- records the **design principle** that observed signals lead and scores
  annotate (merged into project-hygiene),
- surveys a family of **high-signal observability features**,
- prioritises them into waves,
- defines how each lead graduates into its own concrete proposal.

## Design principle (the antidote to "weinigzeggend")

> Lead with the concrete observed fact; the rule pack annotates it.

Every Finding should read as a plain statement before any score — "your
mail lands at Google (US)", "your DNS is run by Cloudflare (US)", "this
page loads fonts from Google" — and the synthesis surface should be a
**map of what goes where**, not a table of verdicts. Scores are the
second layer, never the first.

## Candidate signals (the survey)

Scored on: **signal** (how self-evident/actionable), **mission** (EU/NL
sovereignty fit), **cost**, **reuse** (existing Wanderer machinery),
**vantage** (robust regardless of where the scanner runs).

| Lead | What it observes | Reuses | Notes |
|------|------------------|--------|-------|
| **Transit path** | the network path + hosting of a target (traceroute) | ip/GeoIP | flagship; already a proposal. Destination robust, transit vantage-flavoured |
| **Destination hosting identity** | rDNS + whois/ASN org → "hosted at CYSO/Hetzner/AWS" | ip, whois | cheap enrichment; turns an IP into a who |
| **Email routing** | MX → MX host → ASN/country → "mail lands at Google (US)" | dns.mx, ip | very high signal, cheap |
| **DNS hosting** | NS → who/where runs your DNS (Cloudflare US, …) | dns.ns, ip | high signal, cheap |
| **Web third-party origin** | what a page loads (fonts, analytics, CDN, scripts) + jurisdiction | http | the "what goes where" on the page itself |
| **CDN / front detection** | is the apex behind Cloudflare/Fastly/Akamai (US fronts) | http, tls, ip | reframes existing data |
| **TLS chain geography** | issuer + intermediate CA jurisdictions | tls.issuer | partly exists; make the chain explicit |
| **Misuse / exposure** | exposed admin panels, open services, leaked metadata | http, ip | the "of misbruikt" angle |
| **BGP route-origin + RPKI** | who originates the prefix; RPKI validity / hijack risk | ip | heavier; external data; later |

Already shipped and on-theme (the model to extend): object-storage
origin (Nextcloud inspector), package vendor jurisdiction
(`eu_package_origin`), container-image sovereignty, egress flows.

## Recommended roadmap

- **Wave 1 — turn endpoints into "who/where" (cheap, high reuse):**
  transit-path (proposed) + email-routing + DNS-hosting +
  destination-hosting-identity. These convert data Wanderer already
  collects into plain who/where statements. Biggest "speak" per unit
  effort.
- **Wave 2 — what a surface pulls in:** web third-party origin map +
  CDN/front detection + TLS-chain geography polish.
- **Wave 3 — the synthesis:** an org-centric **data-flow map** ("spider
  in the web") aggregating inbound / outbound / transit into one
  rendered view. This is where the whole thing becomes the dashboard
  Wanderer should have led with.
- **Parallel track:** misuse/exposure signals.
- **Later / heavier:** BGP route-origin + RPKI.

## How a lead graduates

Each lead, when picked, becomes its own `propose-*` change with a full
design pass (Why / What Changes / Scope / Open questions / Risks),
mirroring `propose-transit-path-probe`. This doc is the backlog + the
prioritisation; it does not itself add probes or requirements beyond
the design principle.

## Open questions

1. **Wave-1 order.** All four are cheap; which first after transit-path?
   Recommendation: **email-routing** (highest "oh, my mail is at Google"
   signal), then DNS-hosting, then hosting-identity enrichment.
2. **The flow map (Wave 3).** Is the org-centric map a Wanderer UI page,
   or an export others render? Recommendation: a UI page first; the
   data is already in the store.
3. **Vantage.** Several leads (transit, egress) are stronger from the
   agent on-host than the perimeter scanner. Recommendation: ship the
   perimeter version first, agent version as the authoritative
   follow-up — same pattern as the transit-path proposal.

## Not in scope

- Implementation of any lead (each gets its own proposal).
- Active offensive testing beyond passive observation + light probing.

## Parallel-safe

Paper-only. The only spec change is the design principle added to
project-hygiene; concrete probes/rules/UI land through their own
proposals.
