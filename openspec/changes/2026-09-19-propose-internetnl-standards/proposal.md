# Proposal: Standards dimension — ingest Internet.nl measurements instead of reimplementing them

> **Status:** Draft (2026-09-19). Companion to
> `propose-accountability-dimension`. v1 is import-only (Amass
> pattern, zero new network dependencies); v2 (facade probe via the
> scheduler) is a named follow-up, not part of this change.

## Why

Wanderer's wand pack has no DNSSEC, no SPF/DKIM/DMARC, no STARTTLS,
no DANE, no RPKI, no IPv6-compliance scoring. Those are exactly the
measurements Internet.nl already performs authoritatively — the
public, open-source test suite behind the Dutch comply-or-explain
standards list (Forum Standaardisatie). Reimplementing any of them in
Wanderer would be slower, less correct, and politically pointless:
the verdict a Dutch public-sector organisation is held to *is* the
Internet.nl verdict.

The measurement client already exists in-house: **netnl**
(internetnl-cli), a zero-dependency CLI for the Internet.nl batch API
v2 plus a self-hostable multi-tenant facade (`netnl-serve`), already
running in GitHub Actions. This change connects the two **through a
versioned file contract, not through code**: netnl exports findings,
Wanderer imports them. Either tool remains fully useful without the
other.

## Stripped from Wanderer — permanent non-goals

Wanderer SHALL NOT implement its own probes for: DNSSEC validation,
RPKI/route-origin validation, SPF/DKIM/DMARC evaluation,
STARTTLS/DANE probing, TLS cipher/protocol configuration grading, or
IPv6 standards compliance. These are delegated to Internet.nl via
this import path, permanently. The deferral policy for future probe
proposals: **Wanderer builds a probe only when it needs finding-level
evidence Internet.nl does not expose, or when the check must work
without the batch API.** Under that policy the accountability
proposal's own probes stay first-party on purpose: security.txt
(Wanderer needs the raw Contact/Expires fields for the answer sheet)
and the variants probe (per-path redirect chains as evidence), even
though Internet.nl covers adjacent ground. The rules for those SHALL
never double-score against Internet.nl findings.

## Loose coupling — the contract

- Coupling point: one versioned JSON schema, `netnl-findings/v1`
  (see `NETNL-CONTRACT.md`, which lists the requirements for the
  netnl repo). Wanderer depends on the schema, never on netnl code,
  the facade, or the batch API.
- No import present → the standards dimension renders "not measured"
  (its rules score onbekend with that reason). Wanderer's scan
  pipeline is untouched.
- netnl keeps working standalone in CI as today; exporting findings
  is an additive output format on its side.
- v2 (facade probe: scheduler submits via `netnl-serve`, ingests on
  completion webhook) changes *when* the file arrives, not its shape.
  The importer is the only consumer either way.

## DICTU dimension(s)

Informs Technologie and Operationeel indirectly, but lands as a
wand-native **`standards`** dimension: the norm behind it is the
Forum Standaardisatie comply-or-explain list, a Dutch public-sector
obligation in its own right — precisely wand's audience. (Same
extension precedent as the accountability dimension.)

## Passive / active boundary

v1 initiates **zero** network connections: it reads a file. All
probing is performed by Internet.nl under its own measurement policy.
v2 talks only to the operator's own facade. Nothing here can generate
an abuse complaint attributable to Wanderer.

## What Changes

- `wanderer import internetnl <file>`: parses `netnl-findings/v1`,
  matches domains to known targets (warning + skip for unknown
  domains), persists findings `internetnl.web.*` / `internetnl.mail.*`
  under an import-kind scan, carrying verdict, category, measured_at,
  and the Internet.nl report URL as evidence.
- Assessor: `standards` dimension with six rules mapping Internet.nl
  category verdicts — `dnssec`, `mail_auth` (SPF+DKIM+DMARC),
  `starttls_dane`, `ipv6`, `rpki`, `tls_config`. Verdict mapping
  only; Wanderer SHALL NOT recompute Internet.nl's percentage score.
  Findings older than a configurable `standards.max_age` (default 30
  days) score onbekend ("measurement stale").
- UI: no new IA. Standards renders through the existing rule
  rendering on the report page, with the report URL linked in the
  evidence; the Overview row reuses the existing per-framework pill
  mechanism.

## Scope / Not in scope

**In:** the importer, the schema consumer, six standards rules,
staleness handling, the netnl-side contract document.

**Out:** the v2 facade probe (own proposal once v1 has run in anger);
webhook plumbing; scraping the Internet.nl HTML report (batch API
only, via netnl); any re-weighting or aggregation of Internet.nl
subtests; alerting.

## Parallel-safe

New import command + parser, new rule files + registration, no schema
change to findings (key/value as always), no probe or scanner-path
edits. Fully disjoint from the accountability change except the
shared docs update.
