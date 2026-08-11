# Tasks: high-signal observability (research)

This is a research / prioritisation umbrella. "Done" = the principle is
recorded and each lead has a clear next step; the leads themselves are
tracked as their own proposals.

## 1. Frame

- [x] 1.1 Record the design principle (observed signal leads, score
  annotates) in project-hygiene. (gemerged bij archivering van deze change)
- [x] 1.2 Confirm the candidate-signal survey + the wave ordering with
  Mark (Q1–Q3 in proposal.md). (bevestigd door Mark 2026-08-11: archiveren als af)

## 2. Graduate the leads (each → its own `propose-*` change)

- [x] 2.1 Transit path → `propose-transit-path-probe` (done — the
  template).
- [x] 2.2 Email routing (MX → host → jurisdiction) →
  `propose-email-routing` (proposed 2026-06-17; second instance after
  transit-path).
- [x] 2.3 DNS hosting (NS → who/where) → `propose-dns-hosting` (proposed
  2026-06-18; third instance after transit-path and email-routing).
- [x] 2.4 Destination hosting identity (ASN-org → who hosts the apex) →
  `propose-hosting-identity` (proposed + shipped 2026-06-22; fourth twin,
  completing the cheap Hosting/Mail/DNS/Transit who-where set). rDNS +
  whois enrichment deferred to a follow-up.
- [x] 2.5 Web third-party origin map (vendor-grouped: what a page loads →
  who → where) → `propose-third-party-origin` (proposed + shipped
  2026-06-22; first Wave-2 surface signal). Per-subpage crawl + CSP +
  runtime capture deferred.
- [x] 2.6 CDN / front detection (is the apex behind Cloudflare/Fastly/
  Akamai) → `propose-cdn-front` (proposed + shipped 2026-06-22; second
  Wave-2 signal). Reframes a fronted apex the hosting signal reads as
  "hosted at".
- [x] 2.6b TLS-chain geography polish (name the CA + map intermediate-CA
  jurisdictions; new Certificate flow row) → `propose-tls-chain-geography`
  (proposed + shipped 2026-06-24). **Wave 2 complete** — seven who/where
  signals now lead with the observed fact.
- [x] 2.7 Org-centric data-flow map ("spider in the web") UI. → graduates to its
  own Wave 3+ `propose-*` change (buiten scope van deze umbrella).
- [x] 2.8 Misuse / exposure signals (parallel track). → eigen Wave 3+ proposal.
- [x] 2.9 BGP route-origin + RPKI (later / heavier). → eigen Wave 3+ proposal.

## 3. Wrap-up

- [x] 3.1 Keep this doc as the living backlog; archive only when the
  principle is merged and the wave-1 leads each have a proposal. (voldaan:
  Wave 2 compleet, wave-1 leads 2.1–2.4 elk een proposal; principe merget hier)
