# Proposal: Agent local outbox spool with retry

## Intent

In remote mode, `wanderer agent` posts findings to a Wanderer core
over HMAC-signed HTTPS. When the core is unreachable today the
agent logs an error and drops the batch — the next tick simply
re-runs the inspectors. That is acceptable for the MVP but means
findings produced during a network outage are lost forever.

This change adds a local **outbox spool**: when a POST fails, the
agent serialises the batch to disk in a configured directory; on
the next tick (and at startup) it drains the outbox before running
fresh inspectors. The outbox is bounded so a long outage does not
fill the disk.

## Scope

**In scope:**

- A spool directory (default `/var/lib/wanderer/agent/outbox`),
  one file per failed batch, JSON-encoded `{findings: [...]}`.
- Drain-then-collect order: on each tick, drain the outbox first,
  then collect new findings, then post the new batch.
- Exponential backoff with jitter inside a single tick when the
  core fails — capped at 3 retries before spooling.
- A bound on outbox size: oldest files are pruned when total size
  exceeds 100 MiB (configurable).
- Tests: HTTP failure paths, spool round-trip, prune semantics.

**Out of scope:**

- A persistent retry queue (Kafka, BadgerDB). The on-disk JSON
  files are sufficient at this scale.
- Cross-host replication of the outbox. Each agent owns its own.
- Encryption of spooled files. They contain redacted-already
  findings — the secrets are gone before the spool sees them.

## DICTU dimensions informed

None directly. This is operational reliability for remote-mode
agents.

## Passive/active boundary

Local disk only.

## Why now

The deferred 8.3 in `add-inventory-probe` flagged this as a known
gap. The fix is small enough (one new file plus a couple of
helpers) and the value (no data loss during a network blip) is
concrete.
