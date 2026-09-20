# Proposal: an answer, then the reasoning, then the reporting

## Why

Mark, 2026-09-20, about the UI as it stands: *"dit is niet intuïtief"*.
He is right, and the reason is structural. The UI is built around
**scans and rules**; a person arrives with a **question about one
domain**. Today that person lands on a fleet overview, clicks a scan,
and meets a rule×score matrix — three screens before anything answers
what they came for.

What he asked for, in his words: a user *"snel een url in kunnen
vullen"* — or, when an agent is installed on a system, pick that
system — and get *"heel snel ja/nee"*. Then the drill-down: *"hoe en
wat de besluitvorming"*, in the register of an Internet.nl result page.
And only if wanted, an analyst dives into the reporting.

That is the DAR layering this repo already documents
(`docs/explanation/architecture.md`: Dashboard, Analysis, Reporting)
and ADR-0017's Tourist / Explorer / Farmer — but with the dashboard
layer turned into an **answer** instead of an inventory of scans.

## What Changes

**1. The door.** `/ui/` leads with one input: a domain, or a host that
already reports through an agent. Submitting starts a scan and lands on
that domain's answer. Recently answered domains sit underneath, as
one-line verdicts. No scan IDs, no rule IDs, no matrix.

**2. The answer.** One sentence, in Dutch, in the register the
accountability dimension already uses: *"Ja — dit domein staat onder
Nederlands recht"* / *"Nee — de mail loopt via een Amerikaanse
aanbieder"* / *"Dat weten we niet — GeoIP ontbreekt"*. The four-value
scale is unchanged underneath: `soeverein` and `voldoende` read as
**ja**, `afhankelijk` reads as **nee**, `onbekend` stays honestly
unknown. The headline names the worst thing that made it a "nee", so
the first line already carries the reason.

**3. It fills in while you watch.** A scan takes 30–60 seconds, mostly
the traceroute. The answer page renders what is already known and
refreshes as probes land: DNS and TLS within a second, transit later.
Nothing waits for everything.

**4. The reasoning.** One click from the answer: the seven flows
(hosting, mail, DNS, transit, CDN/hyperscaler, third parties,
certificate) plus accountability, each a plain question with ja / nee /
onbekend / n.v.t., the observed fact named in the verdict, and the
evidence collapsed beneath it. Rule IDs and RDAP jargon live in the
evidence, never in the headline.

**5. The reporting stays where it is.** Trends, the rule catalogue and
the exports do not change. They stop being the first thing a person
meets.

## Scope / Not in scope

**In:** the entry surface, the verdict mapping and its copy, the
progressive answer page, the reorganised drill-down, and the routes
between them.

**Out:** the assessor, the rules, the probes, the exporters, the API.
No new observation, no new scoring — this change moves and renames what
is already measured. Also out: anonymous scanning. Every user of this
instance is authenticated through Keycloak (homelab, realm
`westerweel`), so the scan form is available to signed-in users and the
`--ui-allow-scan` dev-mode flag stops being how this is gated.

## Decisions taken with Mark (2026-09-20)

- The question is "does this stand under Dutch / European law?", with
  the four-value scale mapped to ja / nee / onbekend as above.
- No anonymous scanning: *"we hebben enkel nog ingelogde gebruikers"*.
- Progressive rendering over a "fast mode" that skips probes: you see
  something immediately and it grows more complete while you look.

## Risk that belongs in writing

A yes/no headline that hides an `onbekend` is a lie by rounding. A
domain whose GeoIP failed is not "ja". The mapping therefore never
promotes unknown to yes, and the headline says how many questions could
not be answered.
