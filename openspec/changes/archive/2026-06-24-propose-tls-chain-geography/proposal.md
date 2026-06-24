# Proposal: TLS-chain geography — name the CA that controls a target's cert identity, and where it sits

> **Status:** Proposed — open questions pending (2026-06-22). Closes
> **Wave 2** of `research-high-signal-observability` (the sibling split
> off from CDN/front detection, 2.6b). Same design principle as the six
> prior signals — lead with the observed fact, the rule annotates —
> applied to the certificate chain.

## DICTU dimensions

**Juridisch** (the Certificate Authority controls the cryptographic
identity of the site and can revoke or refuse to renew it; a CA under a
foreign jurisdiction is reachable by a sanctions regime or court order an
EEA regulator cannot). The existing `tls.issuer` and `tls.chain` Findings
carry the evidence; the scoring rule `wand.juridisch.cert_issuer_eea` sits
under Juridisch.

## Why

Wanderer already *scores* the certificate issuer: `wand.juridisch.cert_issuer_eea`
reads the leaf certificate's `issuer_country` and returns a verdict like
**"cert issued in US (outside EEA)"**. That verdict names a country, not a
who — the same **weinigzeggend** gap the six prior signals closed. An
operator reads "cert issued in US" and still has to inspect the chain to
learn the self-evident fact:

> **Your certificate is issued by Let's Encrypt (US); the chain runs
> Let's Encrypt R3 → ISRG Root X1.**

Wanderer already collects the pieces — `tls.issuer` carries the issuer
organisation, common name, and country; `tls.chain` carries the
intermediate common names. The gap is the twins' gap: **name the CA**
(issuer org/CN → a recognisable authority — Let's Encrypt, DigiCert,
Sectigo, GlobalSign, Google Trust Services, Amazon, …) and **surface the
chain geography** — the signing CA, its jurisdiction, and the chain to the
root — as a first-class rendered signal ahead of the country score. Unlike
the other six, the certificate has no flow row yet; this lead also gives
it one.

## What Changes

- **Name the issuing CA, not just the country.** Map the `tls.issuer`
  organisation / common name to a recognisable Certificate Authority via a
  small known-CA table (Let's Encrypt, DigiCert, Sectigo, GlobalSign,
  GoDaddy, Entrust, Google Trust Services, Amazon, Cloudflare, …), keeping
  the raw issuer org/CN as evidence.
- **Make the chain geography explicit.** Emit one aggregate observed
  Finding per domain — `tls.chain_geography` — stating it plainly: *"the
  TLS certificate for {domain} is issued by {CA} ({country})"*, listing
  the chain (leaf issuer → intermediates) with the jurisdiction of each
  link that is known. Observation severity; it is a fact, not a verdict.
- **Give the certificate a first-class flow row.** Add a **Certificate**
  row to the Sovereignty overview mapped to `cert_issuer_eea`; once the
  rule verdict leads with the named CA, that row reads "issued by Let's
  Encrypt (US)" alongside the six other flows, completing the picture.
- The existing `wand.juridisch.cert_issuer_eea` rule **annotates** the
  observed Finding and leads its verdict with the named CA (reading the
  new aggregate), tolerant of the JSON-reloaded attribute shape — exactly
  as the prior rules were changed. The EEA scoring is unchanged.

## Intent

Turn the last bare-country signal — who controls the cryptographic
identity of the site, and under whose jurisdiction — into a named,
chain-explicit statement that leads, with the EEA score as the annotation
behind it. With this, all seven who/where signals (Hosting, Mail, DNS,
Transit, page origin map, CDN front, certificate) speak the same way,
ready for the Wave-3 org-centric data-flow map to read them as one
picture.

## Scope

- Reuses `tls.issuer` (issuer org/CN/country) + `tls.chain`
  (intermediates) end-to-end. No new network traffic beyond the TLS
  handshake already performed.
- Adds CA-naming (a small table + helper) and the aggregate Finding; adds
  one flow row. No schema change, no new dependency.
- Optionally enriches the `tls.chain` Finding with each intermediate's
  organisation and country, read from the certificates the TLS probe
  already parsed (passive, no new network) — see open question 4.
- The scanned apex's certificate only. Per-host certificate geography for
  related names, full root-store trust analysis, and CT-log monitoring are
  separate, larger leads.

## Open questions

1. **Where does the aggregate get synthesised?** Same shape as the six
   prior signals: it needs `tls.issuer` and `tls.chain`, both available
   after the TLS probe. Options: (a) a scanner synthesis step alongside
   the existing `synthesise*` helpers, (b) an observation rule.
   **Recommendation: (a)** — mirror the prior signals so the observed
   Finding stays produced by the scanner and the wand rule stays a pure
   annotation.
2. **CA table — how much, where?** A small curated map of issuer
   org/CN substrings → CA brand. **Recommendation: ~10–12 well-known CAs**
   (Let's Encrypt, DigiCert, Sectigo, GlobalSign, GoDaddy, Entrust, Google
   Trust Services, Amazon, Cloudflare, IdenTrust, Buypass, Actalis),
   grown in-repo; raw issuer org/CN retained as fallback and evidence.
3. **Add a Certificate flow row?** The certificate is the one who/where
   signal with no Sovereignty-overview row. **Recommendation: yes** — add a
   **Certificate** row mapped to `cert_issuer_eea`, so the named-CA verdict
   renders alongside the others; it is a one-line `flowRules` addition,
   pure presentation.
4. **Enrich `tls.chain` with per-intermediate jurisdiction?** The leaf
   issuer's country (already in `tls.issuer`) is the signing CA's
   jurisdiction — the headline. Per-link geography needs the intermediate
   certs' org/country, which the probe parses but does not currently
   record. Adding them is passive (no new network). **Recommendation:
   enrich** — it is what "make the chain explicit" means and it is cheap;
   keep `intermediates` (CNs) for backward compatibility and add the
   org/country detail alongside.

## Risks

- **No `tls.issuer` (TLS probe failed / plain HTTP).** Without it there is
  no CA to name. Mitigation: emit no `tls.chain_geography` Finding; the
  rule keeps its existing "no tls.issuer" Onbekend. The scan completes.
- **CA misattribution.** A stale or wrong table entry mislabels the
  authority. Mitigation: the table is the hint; always show the raw issuer
  org/CN, so the evidence stands even if the friendly name is off. Table
  is unit-tested per entry.
- **Issuer country absent.** Many CA certs omit the country field.
  Mitigation: name the CA with "jurisdiction undetermined" — the same
  graceful path the prior signals take; the CA brand itself (e.g. Let's
  Encrypt = US) is the practical jurisdiction tell even when the field is
  blank.
- **Chain to the root incomplete.** TLS handshakes usually omit the root.
  Mitigation: state the chain as observed (leaf → presented
  intermediates); do not infer links that were not served.

## Not in scope

- Per-host certificate geography for related names (MX, NS, third-party
  hosts), full root-store / trust-path analysis, CAA-vs-actual-issuer
  cross-checks, and CT-log monitoring — separate, larger leads.
- Changing how `cert_issuer_eea` *scores* (the EEA issuer-country logic);
  this change only adds the observed CA + chain it annotates, and the flow
  row that renders it.

## Parallel-safe

Adds CA-naming + an aggregate `tls.chain_geography` Finding (scanner
synthesis step), one **Certificate** flow row, and leads the existing cert
rule's verdict with the named CA; optionally enriches the `tls.chain`
Finding (passive). One new `tls-chain-geography` capability spec. Reuses
existing probes (the chain enrichment is additive). No schema change, no
new dependency, degrades gracefully without an issuer country.
