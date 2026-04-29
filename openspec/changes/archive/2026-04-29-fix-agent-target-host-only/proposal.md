# Proposal: Agent accepts hostnames without TLD

## Intent

`wanderer agent` writes its findings under a `Target` whose
`Domain` is the host's hostname. On a real Linux host that
hostname is often a bare label (`webapp-01`, `nl-prod-db-1`) with
no dot. `models.Target.Validate` requires a TLD — the smoke test
on 2026-04-27 hit this:

```
wanderer agent: upsert target: domain: "wanderer-test-host" has no TLD
```

The rule is right for the perimeter modus (we scan public domains)
but wrong for the agent modus where the "target" is a host
identity, not a public domain.

## Scope

**In scope:**

- A way for the agent to register a Target whose Domain is a bare
  hostname without tripping public-domain validation.
- A small distinction at the model layer: bare-host targets carry
  enough information to be unambiguous in the store (we can keep
  a single `targets` table and use the `Domain` column for both
  shapes; we just relax the validator's TLD check for hosts).

**Out of scope:**

- A separate `hosts` table or a polymorphic Target type. The store
  already carries one entity per scan subject; adding a second
  table multiplies queries without benefit at this scale.
- Renaming `Target.Domain` to `Target.Subject`. That is a public
  API rename and a breaking change to `pkg/models` for no
  practical gain.

## DICTU dimensions informed

None. Correctness fix.

## Passive/active boundary

N/A — model-level change.

## Why now

The smoke test produced a hard error on a realistic config; the
fix is small and unblocks the agent on every host that follows
the standard Linux hostname convention.
