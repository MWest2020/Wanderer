---
status: draft
last_reviewed: 2026-07-12
---

# 0006. In-process cron scheduler, not Kubernetes CronJob

- Status: accepted
- Date: 2026-04-26

## Context

The scheduling capability runs scans on a cron schedule and feeds
each scan into the drift engine. There were three obvious ways to
deliver this:

1. An in-process scheduler running inside `wanderer serve`, owning
   the cron lifecycle and invoking the same scanner pipeline the CLI
   and HTTP API use.
2. An external cron daemon (`cron`, `systemd-timer`) that calls
   `wanderer scan` on a schedule.
3. A Kubernetes `CronJob` per target invoking a Wanderer container.

Option 3 is an attractive long-term answer at scale: parallelism for
free, retries for free, observability through the cluster's existing
machinery. Option 2 keeps Wanderer simple and shifts cadence to the
host operator.

## Decision

Wanderer ships option 1: an in-process scheduler driven by
`github.com/robfig/cron/v3`.

Schedules live in a YAML file pointed at by `--schedules`. The file
is validated at startup; a misconfigured cron expression fails the
process so the operator sees the problem before silently missed
runs accumulate. SIGHUP re-reads the file; the file is not watched
for changes.

The scheduler is owned by the `serve` command and shares its
lifecycle. It does not have its own binary, its own listener, or its
own configuration directory.

## Consequences

- **One process, one source of truth.** The scheduler runs inside
  the same `wanderer serve` instance that serves the HTTP API, holds
  the SQLite handle, and answers MCP requests. Operators have one
  thing to monitor.
- **No deployment ceremony.** A small team running Wanderer on a
  single host gets recurring scans without learning Kubernetes,
  without writing a separate `systemd` template per target, and
  without per-target containerisation.
- **Limited horizontal scale.** Two `wanderer serve` instances
  pointing at the same SQLite file would race; either we add leader
  election (premature) or operators run one instance per dataset.
  At MVP scales (≤ a few hundred targets, scans every few hours)
  this is not a concern.
- **Hot-reload is explicit, not magical.** SIGHUP is the boring
  reload mechanism. A file watcher would feel modern but would also
  be the path most likely to silently miss a tick after a syntax
  error. SIGHUP forces the operator to acknowledge the change.
- **Cron implementation is a real dependency.** `robfig/cron/v3`
  enters `go.mod`. It is pure Go, BSD-3, widely deployed, and
  small. Per [ADR-0003](0003-dependency-policy.md) this is a
  deliberate addition, justified by the ergonomics of having a
  cron parser we trust.

When to revisit: if Wanderer grows multi-tenant (per-org scheduling),
or if a single instance saturates a host's resources, or if drift
analysis becomes expensive enough that we want fan-out. At that
point, expect a follow-up proposal that introduces either external
orchestration (Kubernetes CronJob with Wanderer-as-worker) or
explicit leader election. Until that demand is concrete, keep the
in-process design.
