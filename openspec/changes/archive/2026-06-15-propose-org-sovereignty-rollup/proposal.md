# Proposal: Org-level sovereignty roll-up

> **Status:** Accepted + implemented (2026-06-15). Follow-up to the
> Sovereignty overview (ADR-0015) — the same synthesis at organisation
> scale. Under the high-signal observability umbrella.

## Why

The Sovereignty overview answers "where does *this* service's traffic
go" per scan. The organisation-as-the-spider-in-the-web question is
broader: across all my services, where are the weak spots? "Mail: 3 of
5 services route outside the EEA" is the org-scale insight an operator
steers by.

## What Changes

- A `SovereigntyFlowRollup` aggregation that rolls the per-target flows
  up across an organisation's (or the instance's) latest scans into one
  row per flow category — total assessed, count outside the EEA, and
  the worst score reached.
- A "Sovereignty by flow" section on the dashboard (instance and
  per-organisation), reusing the same flow model. No new collection.

## Not in scope

- An interactive node-graph ("spider in the web") — a deferred
  follow-up (the no-JS UI makes it a separate design decision; the
  tabular roll-up is the verifiable first cut).

## Parallel-safe

Extends `internal/ui/flows.go` + the dashboard view/template + a
Playwright assertion. No schema, probe, or rule change.
