# Proposal: Rules ignore error and no-answer Findings

## Intent

A `dictu.data_ai.mx_present` rule run against a non-resolvable
domain currently scores `voldoende`: it counts every `dns.mx`
Finding it sees, including `lookupError` and `noAnswer` variants
the DNS probe emits when no MX records exist or the host does not
resolve. The smoke test in this session caught this: a domain in
the reserved `.invalid` TLD scored `voldoende` on MX presence.

The bug is one rule but the failure mode is shared across the rule
set: any rule that filters by `ProbeID` and counts hits will also
count error/no-answer findings whose attribute set has nothing to
do with the rule's intent.

## Scope

**In scope:**

- A general principle in the assessor: rules SHALL ignore Findings
  that are *meta* (`*.unavailable`, `*.error`, `*.unconfigured`,
  `no_answer == true`, `error` attribute set) when those Findings
  cannot back the rule's verdict.
- Implementation: `mx_present` and any rule that currently does
  bare ProbeID filtering gets a small predicate helper to check
  the meta-status before counting.
- A regression test reproducing the smoke-test scenario.

**Out of scope:**

- Reshaping the Finding model to make meta-findings a separate
  type. The current convention (meta-findings under their own
  ProbeID suffix or a `no_answer` attribute) is fine; this change
  asks rules to read it.

## DICTU dimensions informed

Same as before; this is correctness, not coverage.

## Passive/active boundary

N/A — rule logic only.

## Why now

The smoke test we ran on 2026-04-27 produced visibly wrong
output. A fix is faster than letting the bug live.
