# Proposal: CDN / front detection — say when a target's apex is behind an edge, and whose

> **Status:** Proposed — open questions pending (2026-06-22). Second
> Wave-2 lead of `research-high-signal-observability`, after the web
> third-party origin map. Same design principle as the five prior signals
> — lead with the observed fact, the rule annotates — applied to the one
> place the existing signals quietly mislead: a CDN-fronted apex.

## DICTU dimensions

**Technologie** (a CDN/edge in front of the apex proxies every request and
terminates the site's TLS — a runtime dependency on the edge operator) and
**Juridisch** (the edge operator's jurisdiction; the major CDNs —
Cloudflare, Fastly, Akamai, CloudFront — are US-headquartered and
CLOUD-Act-reachable). The existing `ip.asn`, `http.response` (`server`
header), and `tls.issuer` Findings already carry the evidence; the scoring
rule `wand.technologie.no_us_hyperscaler` sits under Technologie.

## Why

The hosting-identity signal Wanderer just shipped reads the apex IP's ASN
organisation and says **"example.com is hosted at Cloudflare (US)"**. For
a CDN-fronted site that sentence is quietly wrong in a way that matters:
Cloudflare is not where the site is *hosted* — it is the **edge in front
of it**. The apex IP belongs to the CDN, the origin is masked behind it,
and every request (and the TLS handshake) terminates at the edge operator
before reaching wherever the site actually runs.

Wanderer already collects everything needed to tell the two apart:

- the apex `ip.asn` organisation (a known CDN/edge org — `CLOUDFLARENET`,
  `FASTLY`, `AKAMAI` — is the primary tell);
- the `http.response` **`server`** header (`cloudflare`, `AkamaiGHost`,
  `Vercel`, `Netlify`, … corroborate it);
- the `tls.issuer` (an edge-managed certificate is a secondary tell).

The `wand.technologie.no_us_hyperscaler` rule already *scores* this — its
verdict reads **"US hyperscaler in path: CLOUDFLARENET"** — but that is a
raw ASN org, not a who, and it never says the word that makes it click:
*front*. The design principle is *lead with the observed fact* — and the
self-evident fact here is:

> **Your apex is fronted by Cloudflare (US) — every request and your TLS
> terminate at its edge; the real origin is hidden behind it.**

That sentence reframes the hosting picture for the (very common) fronted
case and needs no framework.

## What Changes

- **Detect the apex front and name the edge.** Correlate the apex
  `ip.asn` organisation with the `http.response` `server` header against a
  small curated CDN-signature table (Cloudflare, Fastly, Akamai, Amazon
  CloudFront, Vercel, Netlify, Sucuri, BunnyCDN, Imperva/Incapsula, …),
  naming the edge operator and the **hosting country** of the edge IP.
  The raw ASN org, server header, and apex address are retained as
  evidence.
- **Emit one aggregate observed Finding** per domain — `http.cdn_front` —
  stating it plainly: *"{domain}'s apex is fronted by {edge} ({country})"*
  when an edge is detected, or *"no CDN/edge front detected — apex served
  directly"* when it is not. Observation severity; it is a fact, not a
  verdict.
- **Render it as a first-class who/where line.** The Sovereignty overview
  already carries a **CDN / hyperscaler** flow row mapped to
  `no_us_hyperscaler`; once the rule verdict leads with the named front,
  that row reads "apex fronted by Cloudflare (US)" for free, alongside the
  five other flows.
- The existing `wand.technologie.no_us_hyperscaler` rule **annotates** the
  observed Finding and leads its verdict with the named front when the
  apex is fronted (reading the new aggregate), tolerant of the
  JSON-reloaded attribute shape — exactly as the prior signals' rules were
  changed. Its US-hyperscaler-in-path scoring is unchanged.

## Intent

Turn the most misleading gap in the existing output — a CDN-fronted apex
that reads as "hosted at Cloudflare" — into a plain observed statement
that names the edge and says it is a *front*, with the
US-hyperscaler-reach score as the annotation behind it. This sharpens the
"spider in the web" picture before Wave 3 draws it: an edge in front of
the apex is a different kind of dependency than an origin, and the map
should say so.

## Scope

- Reuses the apex `ip.asn` + `http.response` (`server` header) + the
  scanned apex's `tls.issuer` end-to-end. No new network traffic beyond
  the lookups already performed.
- Adds CDN-signature detection (a small org/server table + helper) and the
  aggregate Finding; reuses the existing CDN/hyperscaler flow rendering.
  No schema change, no new dependency.
- Apex front only — the edge in front of the scanned apex. Per-host front
  detection for related names, and origin-IP de-masking (finding the IP
  behind the edge), are richer, separate concerns.

## Open questions

1. **Where does the aggregate get synthesised?** Same shape as the five
   prior signals: it needs the apex `ip.asn`, `http.response`, and
   `tls.issuer`, all available after pass 2. Options: (a) a tiny synthesis
   step in the scanner alongside the existing `synthesise*` helpers, (b)
   an observation rule. **Recommendation: (a)** — mirror the prior signals
   so the observed Finding stays produced by the scanner and the wand rule
   stays a pure annotation.
2. **CDN signature table — how much, where?** A small curated table
   keyed on ASN-org substring *and* `server`-header substring (a header
   match raises confidence over org alone). **Recommendation: ~10–12
   well-known edges**, grown in-repo like the mail/DNS/hosting tables; no
   new dependency. Conservative entries only — where the server header or
   a distinctive ASN org is a strong tell — to avoid false "fronted by"
   claims.
3. **What counts as "fronted"?** The apex ASN org being a CDN org is the
   primary signal; the server header corroborates. **Recommendation: flag
   fronted when the apex ASN org matches a CDN signature, or the server
   header does**; record which signal(s) fired as evidence so a
   header-only or org-only match is transparent.
4. **TLS-chain geography polish — now or split?** The backlog bundles
   "CDN/front detection; TLS-chain geography polish". They touch different
   rules (front → `no_us_hyperscaler`; CA chain → `cert_issuer_eea`) and
   are independently shippable. **Recommendation: ship CDN/front detection
   alone (the higher-signal half that reframes the apex); split TLS-chain
   geography into its own follow-up lead** rather than widening this
   change. The cert issuer is used here only as corroborating evidence,
   not scored.

## Risks

- **No GeoIP / no `ip.asn`.** Without the apex org the primary signal is
  gone; the server header alone can still name the edge. Mitigation:
  detect on the server header when present; degrade to "front undetermined
  (no GeoIP, no edge header)" otherwise — the scan completes.
- **Front misattribution.** A stale or wrong signature mislabels the edge,
  or a self-hosted reverse proxy emits a CDN-like header. Mitigation: the
  table is conservative and the Finding records which signal(s) fired
  (org, header) plus the raw values, so the evidence stands even if the
  friendly name is off. Table is unit-tested per entry.
- **Anycast edge with no country.** The dominant case for big CDNs.
  Mitigation: name the edge with "country undetermined (anycast?)",
  mirroring the existing no-country handling.
- **False negative (origin served directly).** A site not behind a known
  edge reads "no front detected" — correct, but the table cannot know
  every edge. Mitigation: stated as a known limit; the observed apex
  hosting identity still stands for the direct case.

## Not in scope

- TLS-chain geography (issuer + intermediate-CA jurisdictions made
  explicit) — the sibling half of the backlog item, split to its own lead
  (Q4).
- Origin de-masking (discovering the real IP behind the edge), per-host
  front detection for related names, and WAF/bot-management detection —
  separate, larger leads.
- Changing how `no_us_hyperscaler` *scores* (US-hyperscaler reach); this
  change only adds the observed front it annotates.

## Parallel-safe

Adds CDN-signature detection + an aggregate `http.cdn_front` Finding
(scanner synthesis step) and leads the existing CDN/hyperscaler rule's
verdict with the named front; reuses the existing flow rendering. One new
`cdn-front` capability spec. Reuses existing probes unchanged. No schema
change, no new dependency, degrades gracefully without GeoIP.
