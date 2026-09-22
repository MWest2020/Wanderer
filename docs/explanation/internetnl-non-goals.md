---
status: draft
last_reviewed: 2026-09-22
---

# Permanent non-goals: what Wanderer will never probe itself

Wanderer's `standards` dimension (see
[assessor.md](../reference/assessor.md#the-standards-dimension)) scores
DNSSEC, SPF/DKIM/DMARC, STARTTLS/DANE, IPv6 reachability, RPKI
route-origin validation, and TLS/HTTPS configuration grading — six
measurement domains that already have an authoritative, publicly
trusted implementation: **Internet.nl**, the open-source test suite
behind the Dutch comply-or-explain standards list (Forum
Standaardisatie). Wanderer consumes those verdicts by import (see
[Feed Internet.nl results from CI](../how-to/internetnl-ci.md)) instead
of reimplementing any of the underlying probes. This is not a
scheduling choice deferred to a future release — it is a **permanent**
non-goal, declared in `openspec/changes/2026-09-19-propose-internetnl-standards/proposal.md`
("Stripped from Wanderer").

## Why permanent, not "not yet"

- **Correctness.** DNSSEC validation, SPF/DKIM/DMARC evaluation,
  DANE/TLSA verification, and RPKI route-origin validation are each
  substantial, security-sensitive pieces of software in their own
  right. Internet.nl's implementations are maintained, audited by
  their own community, and versioned independently of Wanderer — a
  second implementation inside Wanderer would only be a second place
  for the same class of bug to hide, with no compensating benefit.
- **Political pointlessness.** For a Dutch public-sector target, the
  verdict that matters *is* the Internet.nl verdict — it is the
  instrument Forum Standaardisatie and comply-or-explain reporting are
  built around. A Wanderer-computed DNSSEC score that disagreed with
  the public Internet.nl report for the same domain would not read as
  "an independent second opinion"; it would read as "one of these two
  tools is wrong", and undermine trust in whichever one the reader
  checks second.
- **The API already exists and is authoritative.** Reimplementing a
  probe Wanderer can otherwise obtain, for free, from the system of
  record is effort spent recreating a wheel that is also the *only*
  wheel anyone measures a Dutch public-sector domain against.

## The deferral policy for future probe proposals

> Wanderer builds a probe only when it needs finding-level evidence
> Internet.nl does not expose, or when the check must work without the
> batch API.

Two existing Wanderer probes sit close to Internet.nl's coverage and
were kept first-party under this exact policy — they are not
exceptions to the rule above, they are what the rule above allows:

- **`security.txt`** (`wand.accountability.securitytxt`,
  `http.securitytxt`) — Wanderer needs the raw `Contact` and `Expires`
  field values for the accountability answer sheet. Internet.nl's
  `web_appsecpriv_securitytxt` test reports presence/absence, not the
  field contents Wanderer's rule reads.
- **Variants probe** (`wand.operationeel.variant_convergence`,
  `http.variants`) — per-path apex/www × IPv4/IPv6 × http/https
  redirect-chain evidence, which the batch API does not expose at
  that granularity.

`wand.standards.*`'s rules SHALL never double-score against the same
ground these two already cover — see
`internal/assessor/wand/standards_rules_test.go`'s
`TestStandardsRules_NeverScoreAppsecpriv`: `web_appsecpriv` (the
Internet.nl category `security.txt` and the security-headers tests
fall under) is deliberately absent from the standards category → rule
mapping.

## What this means for a new proposal

A future probe proposal that would duplicate one of the six
`standards`-dimension measurement domains — a Wanderer-native DNSSEC
validator, SPF/DKIM/DMARC parser, STARTTLS/DANE prober, TLS
cipher-suite grader, or RPKI validator — is out of scope by design,
not by omission. The Internet.nl-import path, including its v2 facade
follow-up (submit-and-ingest via `netnl-serve`, a named future change,
not part of this one), is the intended permanent answer for this
ground. A proposal that finds Internet.nl's API genuinely does not
expose the finding-level evidence a rule needs is the one exception
this policy already anticipates — argue that case explicitly, the same
way `security.txt` and the variants probe did.
