# Proposal: scheduler-driven Internet.nl facade ingestion (v2)

> **Status:** Stub — not a change. No specs, no tasks. This file
> records the follow-up and why it waits; it is not ready to be
> planned or built.

## Why this exists

`2026-09-19-propose-internetnl-standards` (v1) ships the standards
dimension as a **file import**: an operator runs `netnl` by hand or in
CI, exports `netnl-findings/v1`, and runs `wanderer import internetnl`
(see
[Feed Internet.nl results from CI](../../docs/how-to/internetnl-ci.md)).
Proposal.md already
named the natural next step: **v2**, a facade probe where Wanderer's
own scheduler submits measurements to `netnl-serve` (netnl's
self-hostable multi-tenant facade) and ingests results on a completion
webhook, instead of an operator running netnl by hand. v2 changes
*when* the findings file arrives, not its shape — the importer stays
the only consumer either way (design.md "Why a file, not a library or
an RPC").

## Why it waits

v1 has not run in anger yet. Concretely, before a webhook-ingestion
proposal is worth writing:

- **No production experience with the file contract under real CI
  failure modes.** The design gate for v1
  (`design.md` "Design gate outcome") already found five corrections
  against the *documented* contract by measuring one real batch — a
  facade probe adds a second live system (netnl-serve) whose own
  failure modes (auth, timeout, a batch still pending when the webhook
  fires) are unknown until v1's simpler file path has exposed what
  actually goes wrong in practice, on a real fleet, over time.
- **No signal yet on cadence.** v1 does not answer how often a fleet
  actually needs re-measurement, whether `standards.max_age`'s default
  30 days is right, or how batch-API rate limits interact with a fleet
  larger than one domain — data only a few weeks of real v1 operation
  produces.
- **Premature coupling risk.** Building the scheduler integration
  before v1's contract has proven stable in CI risks designing v2
  against assumptions the file contract itself will still be
  correcting (as it already did twice: `runs/02b-import-zonder-doel.md`,
  `runs/03b-correlatie-over-scansoorten.md`).

## What "ready to propose" looks like

Once v1 has run in CI for a few weeks: write a real proposal.md with a
design.md covering netnl-serve's auth model, webhook delivery
guarantees (at-least-once? retries?), and what changes (if anything)
in the importer to accept a push instead of a CLI invocation — at that
point this stub is deleted or superseded by the real change directory.
