# Proposal: DNS hosting — say who runs a target's authoritative DNS, and where

> **Status:** Proposed — open questions pending (2026-06-18). Wave-1 lead
> graduating out of `research-high-signal-observability` (the third
> concrete instance after `propose-transit-path-probe` and
> `propose-email-routing`, which is the direct template — DNS hosting is
> the structural twin of email routing, one rung further along).

## DICTU dimensions

**Juridisch** (the DNS operator's jurisdiction — a non-EEA managed-DNS
provider resolves and can withhold or redirect every name the
organisation publishes, under foreign jurisdiction) and **Operationeel**
(authoritative DNS is the control plane for the org's whole namespace —
an availability/control concern). The existing `dns.ns` Finding carries
`DimensionHint: Operationeel`; the scoring rule
`wand.juridisch.ns_vendor_jurisdiction` sits under Juridisch.

## Why

Wanderer already *scores* DNS hosting end-to-end: the `dns.ns` probe
resolves the authoritative nameservers, the scanner expands those NS
hosts into `Target.Related`, the `ip` probe attaches `ip.asn`, and
`wand.juridisch.ns_vendor_jurisdiction` correlates them into a verdict
like **"authoritative DNS in US (outside EEA)"**.

That verdict is a *score*, and it is the same kind of **weinigzeggend**
output the field called out for mail. It names a country, not a who. An
operator reads "authoritative DNS in US" and still has to go look up
`ns1.cloudflare.com` to learn the self-evident fact:

> **Your DNS is run by Cloudflare (US).**

That sentence needs no framework. The research direction's design
principle is *lead with the observed fact; the rule pack annotates it* —
and DNS hosting is, with email routing, the cheapest, highest-signal
place to apply it: all the data is already collected and the NS hosts are
already correlated with `ip.asn`. The gap is identical to the one
email-routing closed: **name the operator** (the rule today derives the
country from the ASN org but does not name a recognisable provider) and
**surface the plain statement as a first-class, rendered signal** ahead
of the score.

## What Changes

- **Name the DNS operator, not just the country.** Map each NS host to a
  recognisable managed-DNS operator (Cloudflare, AWS Route 53, Azure DNS,
  Google Cloud DNS, NS1, TransIP, self-hosted, …) from a small
  known-NS-suffix table (`*.cloudflare.com` → Cloudflare,
  `*.awsdns-*.net/org/co.uk` → AWS Route 53, `*.azure-dns.*` → Azure,
  `*.googledomains.com` → Google, `*.transip.net/nl` → TransIP, …) with
  an `ip.asn` organisation fallback — the same operator-naming the
  transit and mail probes added via rDNS/ASN-org.
- **Emit one aggregate observed Finding** per domain — `dns.ns_hosting` —
  stating it plainly: *"DNS for {domain} is run by {operator}
  ({country})"*, listing each NS host → operator → country. Observation
  severity; it is a fact, not a verdict.
- **Render it as a first-class who/where line.** The Sovereignty overview
  already carries a **DNS** flow row mapped to
  `ns_vendor_jurisdiction`; once the rule verdict is operator-led, that
  row reads "DNS run by Cloudflare (US)" for free, alongside the
  Hosting/Mail flows.
- The existing `wand.juridisch.ns_vendor_jurisdiction` rule **annotates**
  the observed Finding (the second layer) and leads its verdict with the
  observed operator name (reading the new aggregate), tolerant of the
  JSON-reloaded attribute shape — exactly as the MX rule was changed.
  Scoring is unchanged.

## Intent

Turn the second-most-relatable sovereignty fact — *who runs your DNS and
where* — into a plain observed statement that leads, with the
EEA-jurisdiction score as the annotation behind it. With mail routing
already speaking, this completes the two cheapest "who/where" signals on
the control plane.

## Scope

- Reuses `dns.ns` + the scanner's `Related`-expansion (NS hosts are
  already expanded — `scanner.go` `case "dns.ns"`) + `ip.asn`
  end-to-end. No new network traffic beyond the NS/A lookups already
  performed.
- Adds operator-naming (a suffix/ASN-org table + helper) and the
  aggregate Finding; reuses the existing DNS flow rendering. No schema
  change.
- Authoritative NS only. Recursive-resolver choice, DNSSEC signing
  posture, and registrar jurisdiction are richer, separate leads —
  noted, not in scope.

## Open questions

1. **Where does the aggregate get synthesised?** Same shape as mail: it
   needs both `dns.ns` and `ip.asn`, and `ip.asn` runs *after* the
   scanner expands `Related` — a post-correlation step. Options: (a) a
   tiny synthesis step in the scanner after enrichment (alongside
   `synthesiseMailRouting`), (b) emit it from a lightweight observation
   rule. **Recommendation: (a)** — mirror email-routing exactly so the
   observed Finding stays produced by the scanner and the wand rule
   stays a pure annotation.
2. **Operator table — how much, where?** A small curated suffix→operator
   map covering the common managed-DNS providers (Cloudflare, AWS,
   Azure, Google, NS1, the big NL hosters like TransIP/Antagonist) plus
   an ASN-org fallback. **Recommendation: start with ~10 well-known
   suffixes + ASN-org fallback**, grown in-repo like the egress vendor
   and mail-operator lists; no new dependency. Consider sharing the
   operator-suffix machinery with `mailrouting.go` rather than
   duplicating it.
3. **Country vs operator HQ.** The NS host's *hosting* country (from
   `ip.asn`) can differ from the operator's *legal* HQ (Cloudflare
   anycast resolving in an EU PoP is still US-controlled — the CLOUD Act
   point). **Recommendation: surface the hosting country as the observed
   fact; let the rule carry the HQ/jurisdiction nuance** — the same split
   email-routing and the transit probe use.

## Risks

- **No GeoIP / no `ip.asn`.** Without the mmdb the country is unknown.
  Mitigation: degrade to "DNS run by {operator}" (operator from the
  suffix table / rDNS) with country omitted — the graceful path the
  transit and mail probes and the NS rule's `noCountryResult` already
  take.
- **Operator misattribution.** A stale or wrong suffix entry mislabels
  the operator. Mitigation: the suffix table is the *hint*; always show
  the raw NS host and ASN-org alongside, so the observed evidence stands
  even if the friendly name is off. Table is unit-tested per entry.
- **Anycast NS with no country.** The dominant case for big DNS
  providers. Mitigation: the observed line says "operator X, country
  undetermined (anycast?)", mirroring the NS rule's existing no-country
  handling.
- **No NS records.** A domain whose authoritative NS cannot be resolved.
  Mitigation: the aggregate states there is no resolvable authoritative
  DNS rather than failing; the scan completes.

## Not in scope

- Recursive-resolver / DNS-resolver-choice analysis, DNSSEC signing
  posture, and registrar jurisdiction — separate, larger leads.
- Probing or connecting to the nameservers beyond the DNS lookups
  already performed (no zone transfers, no version.bind) — that crosses
  toward active testing; passive DNS + reuse only.
- Changing how `ns_vendor_jurisdiction` *scores*; this change only adds
  the observed signal it annotates.

## Parallel-safe

Adds operator-naming + an aggregate `dns.ns_hosting` Finding (scanner
synthesis step) and leads the existing NS rule's verdict with the
operator; reuses the existing DNS flow rendering. One new `dns-hosting`
capability spec. Reuses existing probes and the `Related` expansion
unchanged. No schema change, no new dependency, degrades gracefully
without GeoIP.
