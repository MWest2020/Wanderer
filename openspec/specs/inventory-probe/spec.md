# inventory-probe Specification

## Purpose
TBD - created by archiving change add-inventory-probe. Update Purpose after archive.
## Requirements
### Requirement: Agent runs read-only

The `wanderer agent` process SHALL NOT modify system state — no
package installs, no service starts/stops, no file writes outside its
own log, config, and local spool directory.

#### Scenario: Systemd inspector is read-only

- **Given** an agent with the systemd inspector enabled
- **When** the agent runs on a host with 100 systemd units
- **Then** no `systemctl start|stop|enable|disable|mask` calls are made
- **And** `systemctl list-units` / `systemctl show` are the only
  systemctl invocations in the agent's process tree

#### Scenario: Docker inspector is read-only

- **Given** an agent with the Docker inspector enabled
- **When** it inspects the Docker daemon
- **Then** only GET calls are made against the Docker socket
- **And** no `docker run|pull|rm` equivalent calls are issued

---

### Requirement: Inspector unavailability is graceful

The inventory agent SHALL emit a single `inventory.<id>.unavailable`
Finding with a human-readable `reason` when an inspector cannot run
on the current host (missing socket, binary not in PATH, permission
denied), and SHALL continue with the remaining inspectors.

#### Scenario: Docker not installed

- **Given** a host without the Docker daemon
- **When** the agent runs with the Docker inspector enabled
- **Then** a Finding with `ProbeID: inventory.docker.unavailable` is
  produced
- **And** the agent exit code is 0
- **And** other inspectors' Findings are still written

#### Scenario: Permission denied on Docker socket

- **Given** a host where the agent user is not in the `docker` group
- **When** the agent runs
- **Then** the `inventory.docker.unavailable` Finding's `reason`
  attribute contains "permission denied" or the OS-level error

---

### Requirement: Agent-to-core transport is authenticated

In remote mode, `POST /scans/{id}/findings` SHALL reject any request
lacking a valid HMAC signature over the timestamp + body, with a
timestamp skew greater than ±5 minutes, or from an unknown agent
hostname.

#### Scenario: Valid HMAC is accepted

- **Given** a core configured with a shared secret for host `webapp-01`
- **When** an agent posts with a correctly-computed HMAC header
- **Then** the response is 201 Created
- **And** the Findings are persisted and tagged with
  `source_modus: inventory`

#### Scenario: Replayed request

- **Given** a captured valid request from 10 minutes ago
- **When** the attacker replays it against the core
- **Then** the response is 401 Unauthorized
- **And** the `X-Wanderer-Timestamp` validation error is logged

#### Scenario: Wrong secret

- **Given** an agent with the wrong HMAC secret
- **When** it posts findings
- **Then** the response is 401 Unauthorized
- **And** no Findings are persisted

---

### Requirement: Findings carry their source modus

Every Finding produced by the inventory probe SHALL be persisted with
a `source_modus: "inventory"` attribute or column so the assessor's
completeness calculation can distinguish it from perimeter data.

#### Scenario: Perimeter + inventory in same scan

- **Given** a scan running `wanderer serve` + `wanderer agent` for
  the same target
- **When** the assessor runs after both have reported
- **Then** the assessor's Completeness calculation for relevant
  dimensions moves from `partial` to `complete`
- **And** the Assessment rationale cites findings from both source
  modi by their IDs

#### Scenario: Query by modus

- **Given** a scan with 40 findings: 27 perimeter, 13 inventory
- **When** the HTTP API call `GET /scans/{id}?modus=inventory` runs
- **Then** exactly the 13 inventory findings are returned

---

### Requirement: Agent registers host-only Targets

The agent SHALL be able to write Findings under a Target whose
`Domain` is a bare hostname (no TLD) by tagging the Target with
`Kind: host`, and the model layer's TLD requirement SHALL apply
only to Targets whose Kind is `domain` (the default).

#### Scenario: Bare hostname accepted

- **Given** a host with `hostname` set to `webapp-01` (no dot)
- **When** `wanderer agent --once` writes its Findings
- **Then** the Target row in the store has `domain = "webapp-01"`
- **And** `kind = "host"`
- **And** no validation error is raised

#### Scenario: Public-domain validation still firm

- **Given** a perimeter scan invoked via `POST /scans` with body
  `{"domain": "no-tld-here"}`
- **When** the scanner validates the Target
- **Then** the request is rejected with the existing TLD error
- **And** no scan row is created

