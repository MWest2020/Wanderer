# Proposal: Email routing — say where a target's mail actually lands

> **Status:** Accepted — ready to implement (2026-06-17). Q1–Q3 resolved
> as recommended (scanner synthesis / small in-repo operator table /
> hosting-country observed + rule carries jurisdiction). Wave-1 lead
> graduating out of `research-high-signal-observability` (the second
> concrete instance after `propose-transit-path-probe`, the template).

## DICTU dimensions

**Juridisch** (the mail operator's jurisdiction — GDPR / post-Schrems II
adequacy) and **Data & AI** (inbound mail is citizen correspondence —
the content that lands there). The existing `dns.mx` Finding already
carries `DimensionHint: DataAI`; the scoring rule sits under Juridisch.

## Why

Wanderer already *scores* mail routing: the `dns.mx` probe resolves the
MX records, the scanner pushes the MX hosts into `Target.Related`, the
`ip` probe attaches `ip.asn`, and `wand.juridisch.mx_vendor_jurisdiction`
correlates them into a verdict like **"mx hosts in US (outside EEA)"**.

That verdict is a *score*, and it is exactly the kind of output the
field called **weinigzeggend**. It states a country, not a who. An
operator reads "mx hosts in US" and still has to go look up
`aspmx.l.google.com` to learn the self-evident fact:

> **Inbound mail for your domain lands at Google (US).**

That sentence needs no framework to be understood. The research
direction's design principle is *lead with the observed fact; the rule
pack annotates it* — and email routing is the cheapest, highest-signal
place to apply it, because all the data is already collected. The gap is
not correlation; it is **naming the operator** and **surfacing the plain
statement as a first-class, rendered signal** ahead of the score.

## What Changes

- **Name the mail operator, not just the country.** Map each MX host to
  a recognisable operator (Google Workspace, Microsoft 365, Proton,
  Zoho, Mailprotect, self-hosted, …) from its rDNS / `ip.asn`
  organisation plus a small known-MX-suffix table
  (`aspmx.l.google.com` → Google, `*.mail.protection.outlook.com` →
  Microsoft 365, `*.protonmail.ch` → Proton, …). This is the "who" that
  makes "Google (US)" self-evident — the same operator-naming the
  transit probe added via rDNS/ASN-org.
- **Emit one aggregate observed Finding** per domain — `dns.mx_routing`
  — stating it plainly: *"inbound mail for {domain} lands at {operator}
  ({country})"*, listing each MX host → operator → country, ordered by
  preference. Observation severity; it is a fact, not a verdict.
- **Render it as a first-class who/where line** on the report/scan view
  (alongside the transit path), not buried as a rule verdict string.
- The existing `wand.juridisch.mx_vendor_jurisdiction` rule **annotates**
  the observed Finding (the second layer) and is left scoring as today;
  it gains the operator name in its verdict for free if it reads the new
  aggregate.

## Intent

Turn the most relatable sovereignty fact — *where your mail lands and
who runs it* — into a plain observed statement that leads, with the
EEA-jurisdiction score as the annotation behind it. Make Wanderer
*speak* on the surface most operators care about first.

## Scope

- Reuses `dns.mx` + the scanner's `Related`-expansion + `ip.asn`
  end-to-end. No new network traffic beyond the MX/A lookups already
  performed.
- Adds operator-naming (a suffix/ASN-org table + helper) and the
  aggregate Finding; adds the rendering. No schema change.
- Inbound mail only (the MX records). Outbound mail (SPF/DKIM/DMARC
  senders, the actual sending infrastructure) is a richer, separate
  lead — noted, not in scope.

## Open questions

1. **Where does the aggregate get synthesised?** It needs both `dns.mx`
   and `ip.asn`, and `ip.asn` runs *after* the scanner expands `Related`
   — so it is a post-correlation step, not the `dns` probe. Options:
   (a) a tiny synthesis step in the scanner after enrichment, (b) emit
   it from a lightweight observation rule in the assessor.
   **RESOLVED (Mark, 2026-06-17): (a)** — keep it an *observed* Finding
   produced by the scanner so the design principle holds (signal is
   observed, not derived by a rule); the wand rule stays purely an
   annotation.
2. **Operator table — how much, where?** A small curated suffix→operator
   map covering the common public-sector providers (Google, Microsoft
   365, Proton, the big NL mail hosters) plus an ASN-org fallback.
   **RESOLVED (Mark, 2026-06-17): start with ~10 well-known suffixes +
   ASN-org fallback**; grow it the way the egress vendor list grows, not
   a third-party dependency.
3. **Country vs operator HQ.** The MX host's *hosting* country (from
   `ip.asn`) can differ from the operator's *legal* HQ (Google mail in
   an EU region is still US-controlled — the CLOUD Act point).
   **RESOLVED (Mark, 2026-06-17): surface the hosting country as the
   observed fact; let the rule carry the HQ/jurisdiction nuance**, same
   split as the transit destination-vs-control distinction.

## Risks

- **No GeoIP / no `ip.asn`.** Without the mmdb the country is unknown.
  Mitigation: degrade to "lands at {operator}" (operator from the
  suffix table / rDNS) with country omitted — same graceful path as the
  transit probe and the existing MX rule's `noCountryResult`.
- **Operator misattribution.** A stale or wrong suffix entry mislabels
  the operator. Mitigation: the suffix table is the *hint*; always show
  the raw MX host and ASN-org alongside, so the observed evidence stands
  even if the friendly name is off. Table is unit-tested per entry.
- **No MX / null MX (`.`).** A domain with no inbound mail. Mitigation:
  the aggregate states "no inbound mail routing (no MX / null MX)"
  rather than failing — `mxPresent` already covers the scoring side.
- **Anycast MX with no country.** Same undetermined-jurisdiction case
  the MX rule already handles; the observed line says "operator X,
  country undetermined (anycast?)".

## Not in scope

- Outbound mail authentication (SPF/DKIM/DMARC) and sending
  infrastructure — a separate, larger lead.
- Probing or connecting to the mail servers themselves (SMTP banners,
  STARTTLS) — that crosses toward active testing; passive DNS + reuse
  only.
- Changing how `mx_vendor_jurisdiction` *scores*; this change only adds
  the observed signal it annotates.

## Parallel-safe

Adds operator-naming + an aggregate `dns.mx_routing` Finding (scanner
synthesis step) and a report-view who/where line; one new
`email-routing` capability spec. Reuses existing probes and the existing
MX rule unchanged. No schema change, no new dependency, degrades
gracefully without GeoIP.
