# Proposal: Accountability dimension — score who is answerable for the footprint, and whether you can reach them

> **Status:** Design gate passed (2026-09-19, see design.md "Design
> gate outcome"). Implementation runs through habitat — one task
> cluster per run, this change as the contract. No code outside
> habitat runs.

## Why

Wanderer currently answers **where** the footprint lives (Juridisch:
registrar/NS/MX/cert-issuer jurisdiction) and **whether it keeps
working** (Operationeel: CAA, cert validity, DNS redundancy). It does
not answer the third sovereignty question: **who is answerable for
each piece, and can you actually reach them?** A domain whose
registrant hides behind a privacy proxy, whose SOA RNAME bounces,
whose registrar relationship runs through an anonymous reseller, and
which publishes no security.txt is unaccountable infrastructure — for
a public-sector organisation that is a sovereignty defect just as real
as a US nameserver, because incident response, transfer, and legal
process all depend on an identifiable, reachable counterparty.

The observation surface for this already half-exists: the whois probe
pulls registrant country and registrar name out of RDAP but discards
registrant identity, reseller, status codes, and event dates; the HTTP
probe reads headers but never `/.well-known/security.txt`; nothing
reads SOA RNAME; NS host jurisdiction is scored but NS *holder*
transparency is not.

**Attribution.** None. The rules rest on public standards — RDAP
(RFC 9083), RFC 2142 role mailboxes, RFC 9116 `security.txt` — and the
implementation is Wanderer's own. Decision: Mark, 2026-09-22.

## DICTU dimension(s)

None map cleanly, and that is the point: accountability is a gap in
the DICTU instrument itself. wand is Conduction's own pack (ADR-0011), so it
may extend beyond the DICTU dimensions. This change adds
**`accountability`** as a wand-native dimension alongside
juridisch/operationeel/technologie/data_ai/mens. Two rules
(`domain_expiry`, `variant_convergence`) land under the existing
Operationeel dimension because they are continuity properties, not
accountability ones.

## Passive / active boundary

All new observation is passive or indistinguishable from an ordinary
visitor:

- RDAP lookups (already performed; this change parses more of the
  same response, plus one extra RDAP lookup per unique NS registrable
  domain).
- One DNS SOA query per target; one forward+MX resolution of the
  RNAME domain. **No SMTP probing** — mailbox verification would
  cross the abuse boundary and is explicitly out of scope.
- One HTTPS GET to `/.well-known/security.txt` (RFC 9116 — published
  precisely to be fetched).
- The variants probe covers 8 paths (apex/www × v4/v6 × http/https),
  redirect depth ≤ 5 per path, and a hard budget of **24 connections
  per target** across all paths. Budget exhausted → remaining paths
  are recorded as "not followed (budget)" and scoring uses what was
  observed. No retries; every hop passes the SSRF guard.

### The passive ceiling for .nl (stated honestly)

SIDN redacts registrant data for every `.nl` domain and publishes no
expiration event. Matching the registrant against the organisation's
declared name is therefore **impossible via RDAP for .nl** — not a
bug, the passive ceiling. Wanderer does not work around it (no
registrar APIs, no scraping). For `.nl` fleets the registrant and
expiry questions render as "n.v.t." with that reason; the dimension's
story is carried by `no_reseller`, `soa_rname`, `securitytxt`, and
`ns_holder_transparent`.

## What Changes

- **Reason codes (generic, not accountability-specific)**: a rule
  result may carry a machine-readable `reason`. Each code belongs to
  one class: `structural` (not applicable — renders "n.v.t.",
  excluded from dimension aggregation) or `gap` (a measurement hole —
  renders "onbekend", counts as incomplete). The four-value scale is
  unchanged. The standards change reuses this mechanism.
- **whois probe** parses, in addition to the current fields, recursing
  into nested entities: registrant name/kind, privacy-proxy signal,
  reseller entity, domain status codes, and expiry event → new
  findings `whois.registrant_identity`, `whois.reseller`,
  `whois.status`, `whois.expiry`.
- **New soa probe**: `dns.soa` finding with MNAME + RNAME; resolves
  the RNAME mailbox domain (exists? has MX?).
- **HTTP probe** additionally fetches `/.well-known/security.txt` →
  `http.securitytxt` finding (present, parseable, Contact, Expires).
- **New variants probe** → `http.variants` finding (per path:
  reachable / refused / not followed, redirect chain, final origin).
- **Scanner**: RDAP lookup for each unique NS registrable domain →
  `whois.ns_holder` finding; records the organisation's expected
  registrant names as a `config.expected_registrant` finding at scan
  time.
- **Organisation**: `expected_registrant` is a list of names on the
  organisation (store migration on `organisations`, set via
  `wanderer org`). No per-target override in v1.
- **Assessor**: new `accountability` dimension with five rules
  (`registrant_identifiable`, `no_reseller`, `soa_rname`,
  `securitytxt`, `ns_holder_transparent`), plus
  `wand.operationeel.domain_expiry` and
  `wand.operationeel.variant_convergence`. Two YAML lists next to the
  rules: `privacy_proxies.yaml` (commercial proxy services) and
  `registry_redaction.yaml` (TLDs whose registry redacts registrant
  data; seed: `.nl` → SIDN).
- **Copy**: all answer-sheet text (question, verdict per outcome,
  remediation line) in Dutch, in **one** per-rule string table. No
  i18n machinery; a second language is a second table.
- **web-ui**: accountability rendered as an answer sheet, not a rule
  dump — see the web-ui delta. **The interface requirement is
  first-class in this proposal, not a follow-up**: accountability is
  the most human-facing dimension wand has (its findings name people,
  mailboxes, and organisations), and it is only useful if a
  non-specialist can read the verdict. Implementers should study
  Wordsworth for tone and interaction patterns: plain-language
  headline first, evidence expandable underneath, one concrete fix
  per failing rule.

## Schema and contract changes

This change is **not** schema-free. Explicitly:

1. `models.DimensionHint`: new value `accountability`, accepted by
   `Valid()`.
2. `assessor.DICTUDimensions` → **`WandDimensions`** (rename; the list
   now extends beyond DICTU). Consumers iterate the list instead of
   assuming a length.
3. `assessor/report_test.go` and the assessor spec's JSON scenario
   stop asserting exactly five dimensions (`length == 5` → length of
   `WandDimensions`).
4. `models.Rationale` gains `reason` (string, `omitempty`); `RuleResult`
   gains `Reason`. **Additive**: old assessment JSON without `reason`
   and without an `accountability` entry still loads unchanged.
5. Store migration: `organisations.expected_registrant` (JSON array of
   names, default `[]`).

The standards change (`2026-09-19-propose-internetnl-standards`)
builds on items 1, 2 and 4 and adds its own dimension value; the two
changes otherwise stay disjoint.

## Scope / Not in scope

**In:** the five accountability rules, the two operationeel rules,
the probe extensions above, the reason-code mechanism, the
organisation-level expected registrant, the NL string table, UI
answer-sheet rendering.

**Out (explicitly, to keep this bounded):**

- RDAP *protocol* extensions (actor/escalation relationships,
  ISO2-prefixed subject identifiers). That is IETF/ICANN work, not
  Wanderer code. At most a docs/explanation note on which RDAP fields
  we wish existed.
- Any work-around for registry redaction (registrar APIs, scraping,
  live KVK / trade-register lookups). Phase-2 at the earliest.
- SMTP-level verification of RNAME or security.txt mailboxes
  (abuse boundary).
- Email alerting on expiring HTTPS/security.txt/domains. Its own
  proposal.
- Any DNSSEC/RPKI/SPF/DKIM/DMARC/STARTTLS/DANE/TLS-config/IPv6
  *compliance* probes — permanently delegated to Internet.nl via the
  standards change. (The variants probe observes reachability and
  redirects only; it judges no IPv6 compliance.)
- i18n machinery beyond the one string table.

## Build process

Implementation runs as habitat runs (builder, then reviewer), one task
cluster per run, with this change as contract input
(`HABITAT_TASK_REF` → `runs/<nn>-*.md` in this change). No code lands
outside a habitat run; a fix discovered mid-way goes back into this
spec or into its own run.
