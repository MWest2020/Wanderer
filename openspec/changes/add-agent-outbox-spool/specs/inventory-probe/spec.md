# Delta for inventory-probe

## ADDED Requirements

### Requirement: Remote-mode agent never silently drops findings

In remote mode, `wanderer agent` SHALL persist any batch the core
rejects to a local outbox directory and SHALL drain that outbox at
the start of every subsequent tick before collecting new findings,
so a transient network outage cannot lose data already produced
on the host.

#### Scenario: Batch survives a transient outage

- **Given** an agent in remote mode whose core is unreachable
- **When** the agent's tick produces a batch and the POST fails
  three times
- **Then** the batch is written to the outbox directory as a
  single JSON file
- **And** the agent process keeps running

#### Scenario: Outbox drains on the next tick

- **Given** an outbox containing one spooled batch
- **And** the core has come back online
- **When** the next tick begins
- **Then** the spooled batch is POSTed before any new inspector
  runs
- **And** the file is removed only after the POST returns 2xx

### Requirement: Outbox stays bounded

The outbox SHALL refuse to grow past a configured maximum size
(default 100 MiB), pruning the oldest spooled batches when the
limit is exceeded.

#### Scenario: Long outage prunes oldest

- **Given** an outbox configured at 1 MiB and a series of 200 KiB
  batches all failing to post
- **When** the seventh batch arrives
- **Then** the oldest batch is removed before the new one is
  written
- **And** the total on-disk footprint stays at or below 1 MiB
