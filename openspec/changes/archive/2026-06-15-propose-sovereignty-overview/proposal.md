# Proposal: Sovereignty overview — the "spider in the web" synthesis

> **Status:** Design pass. The Wave-3 synthesis from
> research-high-signal-observability. Awaiting Mark's nod on scope (Q1).

## Why

The diligence pass on the high-signal leads found that Wanderer
*already collects* the signals that matter — apex hosting jurisdiction
(`apex_ip_in_eea` / `ip.asn`), mail routing (`mx_vendor_jurisdiction`),
authoritative DNS (`ns_vendor_jurisdiction`), web third parties
(`third_parties_eea`), US hyperscaler/CDN fronting (`no_us_hyperscaler`),
and now the transit path (`wand.transit.eu_path`). The "weinigzeggend"
problem is therefore **not missing data — it is presentation**. Those
signals live scattered across dozens of findings and per-rule verdicts;
nowhere does Wanderer say, in one glance:

> Your service lives at **CYSO (NL)**. Its mail goes to **Google (US)**,
> its DNS is run by **Cloudflare (US)**, the path crosses **NL → US**,
> and the page pulls in **3 non-EEA third parties**.

That org/host-as-the-spil-in-the-web picture — what goes where, what is
used (or misused) where — is the synthesis this change adds.

## What Changes

- A pure aggregation (`SovereigntyFlows`) that reads a scan's existing
  findings and produces an ordered set of **flows** — Hosting, Mail,
  DNS, Transit, Third parties, CDN/front — each with the observed
  destination (org + country) and an EEA/non-EEA classification.
- A **Sovereignty overview** panel on the scan/assessment view that
  renders the flows as a compact "what goes where" list (the textual
  first cut of the spider-in-the-web map).
- No new probe or rule: it consumes what the existing probes/rules
  already observe — observed-fact-first, per the project-hygiene
  design principle.

## Open questions

1. **Textual list vs. graphical map.** Recommendation: **ship the
   textual flow list first** (it answers the feedback at a fraction of
   the cost; the data is all in the store), and treat an interactive
   node-graph as a follow-up once the flow model proves out.
2. **Scope of "flows".** Recommendation: Hosting, Mail, DNS, Transit,
   Third-parties, CDN/front in the first cut — the six the existing
   rules already feed. Add more as new signals land.
3. **Where it renders.** Recommendation: a panel on the scan view
   (the per-scan synthesis) first; an org-level roll-up later.

## Not in scope

- New collection (the signals exist).
- A full interactive graph engine (follow-up).
- TLS-chain intermediate geography (a separate low-value lead — the
  issuing CA jurisdiction is already scored by `cert_issuer_eea`).

## Parallel-safe

New aggregation in `internal/ui` (pure, unit-tested) + one template
panel + a Playwright assertion + an ADR with a UI-surface section. No
schema, probe, or rule change.
