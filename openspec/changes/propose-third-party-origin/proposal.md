# Proposal: Web third-party origin map — name what a page pulls in, and from where

> **Status:** Proposed — open questions pending (2026-06-22). First lead
> of **Wave 2** ("what a surface pulls in") in
> `research-high-signal-observability`, after Wave 1's four who/where
> twins (transit, mail, DNS, hosting) shipped. Same design principle as
> the twins — lead with the observed fact, the rule annotates — but the
> observed fact is a *map* of many vendors, not one endpoint.

## DICTU dimensions

**Technologie** (each third-party host a page loads is a runtime
dependency that sees the visitor's IP, browsing context, and often a
session cookie; a non-EEA vendor exports that on every page load) and
**Juridisch** (the vendor's jurisdiction). The existing
`http.third_party` Finding carries `DimensionHint: Technologie`; the
scoring rule `wand.technologie.third_parties_eea` sits under Technologie.

## Why

Wanderer already *scores* a target's third-party surface: the `http`
probe records one `http.third_party` Finding per external host the apex
page loads (with the `kinds` of resource — script, style, font, img,
iframe), the scanner expands those hosts into `Target.Related`, the `ip`
probe attaches `ip.asn`, and `wand.technologie.third_parties_eea`
correlates them into a verdict like **"3 of 5 third-party hosts resolve
in the EEA"**.

That verdict is a *count*, and it is the same **weinigzeggend** output the
field called out for the Wave-1 flows. It names a number, not a who. An
operator reads "2 of 5 outside the EEA" and still has to open dev-tools
and reverse the hostnames to learn the self-evident fact:

> **Your pages load fonts from Google (US) and a script bundle from
> jsDelivr — analytics stays in the EU.**

That sentence needs no framework. The research direction's design
principle is *lead with the observed fact; the rule pack annotates it* —
and the third-party origin map is the cheapest Wave-2 place to apply it:
all the data is already collected, the hosts are already correlated with
`ip.asn`, and the `kinds` already say *what* each host serves. The gap is
the twins' gap, one step richer: **name the vendor** (host → recognisable
vendor — Google Fonts, Google Analytics, jsDelivr, Cloudflare, …) and
**surface the plain origin map** — vendor → what it serves → where — as a
first-class rendered signal ahead of the count.

## What Changes

- **Name the vendor, not just the host.** Map each third-party host to a
  recognisable vendor via a small known-suffix table
  (`fonts.googleapis.com`/`fonts.gstatic.com` → Google Fonts,
  `google-analytics.com`/`googletagmanager.com` → Google Analytics,
  `cdn.jsdelivr.net` → jsDelivr, `cdnjs.cloudflare.com` → cdnjs,
  `connect.facebook.net` → Meta, …) with an `ip.asn`-organisation
  fallback — reusing the `operatorBySuffix` machinery the mail/DNS twins
  share. The raw host and ASN-org are retained as evidence.
- **Emit one aggregate observed Finding** per domain — `http.origin_map` —
  stating it plainly: a vendor-grouped map, each entry *vendor — what it
  serves (kinds) — country*, with the non-EEA vendors called out as the
  export surface. Observation severity; it is a fact, not a verdict.
- **Render it as a first-class origin-map line.** The Sovereignty overview
  already carries a **Third parties** flow row mapped to
  `third_parties_eea`; once the rule verdict leads with the (non-EEA)
  vendor names, that row reads "loads from Google (US), Cloudflare (US) —
  3 of 5 hosts in the EEA" for free, alongside the four Wave-1 flows.
- The existing `wand.technologie.third_parties_eea` rule **annotates** the
  observed map (the second layer) and leads its verdict with the observed
  vendor names (reading the new aggregate), tolerant of the JSON-reloaded
  attribute shape — exactly as the four Wave-1 rules were changed. Scoring
  (the in/out-EEA count) is unchanged.

## Intent

Turn "what every page load pulls in, and from whose jurisdiction" — the
on-page data-export surface — from a bare count into a named origin map
that leads, with the EEA count as the annotation behind it. This is the
first Wave-2 surface signal and a direct input to the Wave-3 org-centric
data-flow map, which will read the four endpoint flows *and* this on-page
map as one "spider in the web" picture.

## Scope

- Reuses `http.third_party` (+ its `kinds`) + the scanner's
  `Related`-expansion + `ip.asn` end-to-end. No new network traffic beyond
  the page fetch + host lookups already performed.
- Adds vendor-naming (a suffix/ASN-org table, reusing `operatorBySuffix`)
  and the aggregate Finding; reuses the existing Third-parties flow
  rendering. No schema change, no new dependency.
- Apex page only — the third parties the `http` probe already records for
  the scanned page. Per-subpage crawling, CSP/connect-src analysis, and
  actual request capture (vs static HTML parsing) are richer, separate
  leads.

## Open questions

1. **Where does the aggregate get synthesised?** Same shape as the four
   twins: it needs `http.third_party` and `ip.asn`, and `ip.asn` runs in
   pass 2 — a post-correlation step. Options: (a) a tiny synthesis step in
   the scanner after enrichment (alongside the four `synthesise*`
   helpers), (b) emit it from a lightweight observation rule.
   **Recommendation: (a)** — mirror the twins exactly so the observed
   Finding stays produced by the scanner and the wand rule stays a pure
   annotation.
2. **Vendor table — how much, where?** A small curated suffix→vendor map
   covering the common page third parties (Google Fonts/Analytics/Tag
   Manager, jsDelivr, cdnjs, unpkg, jQuery CDN, Meta, Hotjar, …) plus an
   ASN-org fallback. **Recommendation: start with ~15 well-known suffixes
   + ASN-org fallback**, reusing `operatorBySuffix`, grown in-repo like
   the mail/DNS tables; no new dependency.
3. **Grouping — by vendor or by host?** A page often loads several hosts
   from one vendor (`fonts.googleapis.com` + `fonts.gstatic.com`).
   **Recommendation: group by vendor** in the map (dedupe hosts under one
   vendor, union their `kinds` and countries), so the line reads "fonts
   from Google" once, not twice; retain the per-host detail in evidence.
4. **What does the rule lead with — all vendors or the non-EEA ones?**
   The actionable signal is the *export surface*. **Recommendation: lead
   the verdict with the non-EEA vendor names** (the ones that move data
   out), keeping the existing in/out-EEA count as the detail — "loads from
   Google (US) — 3 of 5 hosts in the EEA". All-EEA pages keep a clean
   "all N hosts in the EEA" with no scary lead.

## Risks

- **No GeoIP / no `ip.asn`.** Without the mmdb the per-vendor country is
  unknown. Mitigation: degrade to a vendor list with country omitted — the
  graceful path the rule's `noCountryResult` already takes.
- **Vendor misattribution.** A stale or wrong suffix entry mislabels a
  vendor. Mitigation: the suffix table is the *hint*; always show the raw
  host and ASN-org alongside, so the observed evidence stands even if the
  friendly name is off. Table is unit-tested per entry.
- **Anycast vendor with no country.** Common for CDN-fronted assets.
  Mitigation: the map names the vendor with "country undetermined
  (anycast?)", mirroring the rule's existing no-country handling.
- **No third parties found.** A page that loads nothing external (or the
  HTTP probe did not run). Mitigation: the aggregate states the page loads
  no third parties rather than failing; the scan completes.
- **Static-HTML blind spots.** The probe parses served HTML, so
  script-injected third parties are missed. Mitigation: out of scope and
  stated as such — this maps what the served page declares, a real signal
  even if not exhaustive.

## Not in scope

- Per-subpage crawling, CSP/`connect-src` policy analysis, and runtime
  request capture (headless browser) — separate, larger leads.
- CDN / front detection for the *apex itself* (is the site behind
  Cloudflare/Fastly) — that is the next Wave-2 lead (2.6), which reframes
  apex `ip`/`tls` data, not the page's third parties.
- Changing how `third_parties_eea` *scores* (the in/out-EEA count); this
  change only adds the observed map it annotates.

## Parallel-safe

Adds vendor-naming + an aggregate `http.origin_map` Finding (scanner
synthesis step) and leads the existing third-party rule's verdict with the
vendors; reuses the existing Third-parties flow rendering. One new
`third-party-origin` capability spec. Reuses existing probes and the
`Related` expansion unchanged. No schema change, no new dependency,
degrades gracefully without GeoIP.
