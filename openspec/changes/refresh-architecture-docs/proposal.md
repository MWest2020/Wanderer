# Proposal: Refresh architecture documentation

## Intent

`docs/architecture.md` was written for the MVP scanner suite —
when "Wanderer" meant `wanderer scan` and the four perimeter
probes. It has not been updated since the assessor, scheduling,
inventory, and egress capabilities landed. A new contributor
opening it today gets a misleading picture: the agent modus is
absent, the assessor is described as "future work", and the
diagram still shows only DNS/TLS/IP/HTTP.

This change refreshes the document to reflect the three-modi
triad (perimeter / inventory / egress) and the cross-cutting
layers (assessor, drift, exporters, MCP, scheduling) that
consume Findings.

## Scope

**In scope:**

- A revised component diagram showing the three modi and the
  cross-cutting consumers.
- A short section per modus explaining when it applies and what
  it observes.
- A "How to add a probe" section for the perimeter scanner —
  retained from the original — refreshed for the current code.
- A "How to add an inspector" section for the agent modi.
- Cross-references to the per-capability docs (`assessor.md`,
  `agent.md`, `egress.md`, etc.).

**Out of scope:**

- Renaming `docs/architecture.md`. The path stays so existing
  links keep working.
- Generating diagrams from code. We hand-author the Mermaid
  block; a generator is a separate concern.
- Architecture *changes*. This is documentation-only.

## DICTU dimensions informed

None directly. Operational maturity / contributor onboarding.

## Why now

Every other capability shipped with its own focused doc. The
top-level architecture page is now the only stale entry point a
new contributor reads first. Fixing it before more capabilities
land is cheaper than fixing it after.
