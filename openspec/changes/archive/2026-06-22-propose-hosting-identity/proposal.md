# Proposal: Hosting identity — say who hosts a target's front door, and where

> **Status:** Proposed — open questions pending (2026-06-22). Wave-1 lead
> graduating out of `research-high-signal-observability` (the fourth and
> last cheap "who/where" twin, after `propose-transit-path-probe`,
> `propose-email-routing`, and `propose-dns-hosting`). Same structure as
> DNS hosting, one flow row over: this one speaks for **Hosting**.

## DICTU dimensions

**Juridisch** (the hosting operator's jurisdiction — the apex IP belongs
to an Autonomous System operated by a specific organisation in a specific
country; a non-EEA host serves the organisation's front door under
foreign jurisdiction). The existing `ip.asn` Finding carries
`DimensionHint: Juridisch`; the scoring rule `wand.juridisch.apex_ip_eea`
sits under Juridisch.

## Why

Wanderer already *scores* where a target is hosted: the `dns.a`/`dns.aaaa`
probe resolves the apex, the `ip` probe attaches `ip.asn` (ASN,
organisation, country) to each address, and `wand.juridisch.apex_ip_eea`
correlates them into a verdict like **"apex IPs in US (outside EEA)"**.

That verdict is a *score*, and it is the same **weinigzeggend** output the
field called out for mail and DNS. It names a country, not a who. An
operator reads "apex IPs in DE" and still cannot tell whether that is
Hetzner, AWS Frankfurt, or a municipal data centre — yet the `ip.asn`
organisation already collected (`HETZNER-AS`, `AMAZON-02`,
`CLOUDFLARENET`) answers it:

> **Your front door is hosted at Hetzner (DE).**

That sentence needs no framework. The research direction's design
principle is *lead with the observed fact; the rule pack annotates it* —
and hosting identity is the cheapest remaining place to apply it: the ASN
organisation is already a "who", it is just unspoken and unpolished. The
gap is identical to the one mail and DNS closed: **name the operator**
(the rule today derives the country from the ASN but does not surface a
recognisable host name) and **surface the plain statement as a
first-class, rendered signal** ahead of the score. With this twin, all
four Wave-1 "who/where" rows — Hosting, Mail, DNS, Transit — speak.

## What Changes

- **Name the hosting operator, not just the country.** Resolve the apex
  IP's `ip.asn` organisation to a recognisable host name (Hetzner, AWS,
  Microsoft Azure, Google Cloud, Cloudflare, OVH, Leaseweb, TransIP, …)
  via a small ASN-org normalisation table (strip the `-AS*` / `, Inc.`
  noise, map the common ugly codes to friendly names), keeping the raw
  ASN org as evidence. Same operator-naming move the mail and DNS twins
  added — but the source is the ASN org already on the apex IP, not a
  hostname suffix.
- **Emit one aggregate observed Finding** per domain — `ip.hosting` —
  stating it plainly: *"{domain} is hosted at {operator} ({country})"*,
  listing the apex addresses → operator → country. Observation severity;
  it is a fact, not a verdict.
- **Render it as a first-class who/where line.** The Sovereignty overview
  already carries a **Hosting** flow row mapped to `apex_ip_eea`; once the
  rule verdict is operator-led, that row reads "hosted at Hetzner (DE)"
  for free, alongside the Mail/DNS/Transit flows.
- The existing `wand.juridisch.apex_ip_eea` rule **annotates** the
  observed Finding (the second layer) and leads its verdict with the
  observed operator name (reading the new aggregate), tolerant of the
  JSON-reloaded attribute shape — exactly as the MX and NS rules were
  changed. Scoring is unchanged.

## Intent

Turn the most basic sovereignty fact — *who hosts your service and
where* — into a plain observed statement that leads, with the
EEA-jurisdiction score as the annotation behind it. This completes the
four cheap "who/where" signals (Hosting, Mail, DNS, Transit) that Wave 1
set out to make speak, clearing the way for the Wave-3 org-centric
data-flow map to read them as one picture.

## Scope

- Reuses `dns.a`/`dns.aaaa` + `ip.asn` end-to-end. No new network traffic
  beyond the apex resolution + GeoIP lookup already performed.
- Adds ASN-org operator-naming (a normalisation table + helper) and the
  aggregate Finding; reuses the existing Hosting flow rendering. No schema
  change, no new dependency.
- Apex front door only — the host that serves the apex domain. Related
  hosts (MX, NS, third parties) already have their own twins; per-host
  hosting of every related name is a richer, separate concern.

## Open questions

1. **Where does the aggregate get synthesised?** Same shape as mail and
   DNS: it needs the apex `dns.a`/`dns.aaaa` host and its `ip.asn`, and
   `ip.asn` runs in pass 2 — a post-correlation step. Options: (a) a tiny
   synthesis step in the scanner after enrichment (alongside
   `synthesiseMailRouting` / `synthesiseDNSHosting`), (b) emit it from a
   lightweight observation rule. **Recommendation: (a)** — mirror the two
   prior twins exactly so the observed Finding stays produced by the
   scanner and the wand rule stays a pure annotation.
2. **Operator name source.** Unlike mail/DNS there is no operator
   hostname to map — the apex *is* the domain. The "who" is the `ip.asn`
   organisation, which is already a name, just unpolished
   (`HETZNER-AS`, `AMAZON-02`). **Recommendation: a small ASN-org
   normalisation table** (friendly-name the common hosts, strip the
   `-AS`/`, Inc.` noise) with the raw ASN org retained as the fallback and
   evidence — the analog of the mail/DNS suffix tables, grown in-repo, no
   new dependency. rDNS of the apex IP is a possible secondary hint but
   needs a PTR lookup (new network) — see Q4.
3. **Country vs operator HQ.** The apex IP's *hosting* country (from
   `ip.asn`) can differ from the operator's *legal* HQ (AWS Frankfurt is
   DE-hosted but US-controlled — the CLOUD Act point). **Recommendation:
   surface the hosting country as the observed fact; let the rule carry
   the HQ/jurisdiction nuance** — the same split the three prior twins use.
4. **rDNS / whois enrichment now or later?** The lead names rDNS and whois
   as possible inputs. rDNS needs a new PTR lookup; whois (RDAP) is its
   own probe scoring registrant/registrar country, a different signal
   (who *owns* the domain, not who *hosts* the IP). **Recommendation:
   ship the cheap ASN-org core first (zero new network, exact twin of
   mail/DNS); note rDNS refinement and whois-registrant correlation as a
   follow-up within this capability** rather than widening v1.

## Risks

- **No GeoIP / no `ip.asn`.** Without the mmdb there is no ASN org and no
  country. Mitigation: degrade to "hosting operator undetermined (no
  GeoIP)" — the graceful path the apex rule's `noCountryResult` already
  takes; the scan completes.
- **Operator misattribution.** A stale or wrong normalisation entry
  mislabels the host. Mitigation: the table is the *hint*; always show the
  raw ASN org alongside, so the observed evidence stands even if the
  friendly name is off. Table is unit-tested per entry.
- **Anycast / shared-host apex with no country.** Mitigation: the observed
  line says "operator X, country undetermined (anycast?)", mirroring the
  apex rule's existing no-country handling.
- **No apex A/AAAA record.** A domain whose apex does not resolve.
  Mitigation: the aggregate states there is no resolvable apex host rather
  than failing; the scan completes.

## Not in scope

- rDNS PTR lookups and whois-registrant correlation — noted as
  enrichment follow-ups, not v1 (Q4).
- Per-host hosting identity of every related name (MX, NS, third-party
  hosts) — those names have their own twins; a full per-host hosting map
  is Wave-2/3 territory.
- Changing how `apex_ip_eea` *scores*; this change only adds the observed
  signal it annotates.

## Parallel-safe

Adds ASN-org operator-naming + an aggregate `ip.hosting` Finding (scanner
synthesis step) and leads the existing apex rule's verdict with the
operator; reuses the existing Hosting flow rendering. One new
`hosting-identity` capability spec. Reuses existing probes unchanged. No
schema change, no new dependency, degrades gracefully without GeoIP.
