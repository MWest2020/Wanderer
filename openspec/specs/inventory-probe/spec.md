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

---

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

---

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

---

### Requirement: Docker inspector reports containers and images

When the Docker socket is reachable, the inventory agent SHALL emit
one `inventory.docker.container` Finding per container and one
`inventory.docker.image` Finding per image, populating the
attributes documented in `docs/findings.md`, and SHALL continue to
honour the read-only contract by issuing only GET calls against
the Docker socket.

#### Scenario: Containers listed

- **Given** a host with the Docker daemon running and one
  container present
- **When** the agent's Docker inspector runs
- **Then** exactly one `inventory.docker.container` Finding is
  produced
- **And** its Subject is the container name
- **And** Attributes contain `image`, `image_digest`, `state`

#### Scenario: Read-only contract holds

- **Given** the inspector is running
- **When** it inspects the daemon
- **Then** every HTTP request issued against the socket has method
  GET
- **And** no request URL contains `/exec`, `/wait`, `/start`,
  `/stop`, `/kill`, `/pause`, or `/unpause`

#### Scenario: API error surfaces gracefully

- **Given** the daemon returns an HTTP 500 to `/containers/json`
- **When** the inspector runs
- **Then** an `inventory.docker.error` Finding is produced with a
  `status_code` attribute
- **And** the agent process exits 0 (other inspectors continue)

---

### Requirement: Nextcloud inspector emits version + trust signals

The Nextcloud inspector SHALL, when enabled, emit five classes
of Finding from one `occ` session per tick: the installed
Nextcloud version, the trusted-domain list, every configured
objectstore backend, every configured OIDC provider, and (when
`user_oidc` is absent) an `oidc.unavailable` meta-finding naming
the detected alternative app. Every Finding carries the ProbeID
prefix `inventory.nextcloud.` and `SourceModus: inventory`.

#### Scenario: Version probe emits a supported flag

- **GIVEN** an agent host where `inspectors.nextcloud.enabled: true`
  and the installed Nextcloud is version 28.0.5
- **WHEN** the agent ticks
- **THEN** the inspector emits exactly one
  `inventory.nextcloud.version` Finding with `versionstring: "28.0.5"`,
  `major: 28`, and `supported: true`

#### Scenario: Objectstore probe annotates with geoip

- **GIVEN** an agent host whose Nextcloud has an S3 objectstore
  backend configured with `hostname: s3.amazonaws.com`
- **WHEN** the agent ticks with a HostResolver wired
- **THEN** the resulting `inventory.nextcloud.objectstore`
  Finding carries `endpoint_host: s3.amazonaws.com` plus the
  geoip enrichment (`asn`, `asn_organisation`, `country: "US"`)

#### Scenario: Missing user_oidc names the alternative

- **GIVEN** an agent host whose Nextcloud has `social_login`
  installed but not `user_oidc`
- **WHEN** the agent ticks
- **THEN** the inspector emits one
  `inventory.nextcloud.oidc.unavailable` Finding with
  `alternative_app: "social_login"` in Attributes, instead of an
  `inventory.nextcloud.oidc_provider` Finding

---

### Requirement: Three Nextcloud-as-target rules score the surface

The assessor SHALL register three rules covering the Nextcloud
sovereignty surface: `wand.nextcloud.objectstore_eu`,
`wand.nextcloud.oidc_provider_eu`, and
`eucsf.sov6.nextcloud_supply_chain`. Each rule SHALL follow the
soeverein / afhankelijk / onbekend shape established by the host
telemetry rules: a soeverein verdict cites at least one
inspected Finding ID in Evidence and includes the inspected
count in the Verdict text.

#### Scenario: US objectstore scores afhankelijk

- **GIVEN** one `inventory.nextcloud.objectstore` Finding with
  `country: "US"`
- **WHEN** the assessor runs `wand.nextcloud.objectstore_eu`
- **THEN** the resulting Rationale has Score `afhankelijk` and
  the Verdict names the offending bucket + country

#### Scenario: SEAL combined rule rolls both signals up

- **GIVEN** one US objectstore + one US OIDC provider Finding
- **WHEN** the assessor runs `eucsf.sov6.nextcloud_supply_chain`
- **THEN** the Verdict mentions both subjects, includes the
  `[SEAL 1]` tag, and the Score is `afhankelijk`

---

### Requirement: Container image sovereignty rules score the Docker inventory

The assessor SHALL register three rules covering the Docker
inventory surface: `wand.docker.images_us_registry`,
`wand.docker.containers_us_registry`, and
`eucsf.sov6.container_supply_chain`. Each rule reads
`inventory.docker.image` and/or `inventory.docker.container`
Findings, classifies their image references against
`internal/assessor/container_registries.yaml`, and follows the
soeverein / afhankelijk / onbekend shape established by the
host telemetry rules.

#### Scenario: gcr.io image scores afhankelijk

- **GIVEN** one `inventory.docker.image` Finding whose
  `repo_tags` includes `gcr.io/foo/bar:v1`
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the Rationale has Score `afhankelijk`, Verdict
  naming the offending image + Google as the vendor of record,
  and Evidence citing the Finding's ID

#### Scenario: Bare nginx is treated as docker.io implicit

- **GIVEN** one `inventory.docker.image` Finding whose
  `repo_tags` is `["nginx:1.27"]` (no registry prefix)
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the Verdict text names `docker.io (implicit)` and
  the Score is `afhankelijk`

#### Scenario: Self-hosted EU registry scores soeverein

- **GIVEN** three `inventory.docker.image` Findings whose
  `repo_tags` all start with `harbor.example.de/`
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the Score is `soeverein`, the Verdict text
  includes `"inspected 3 images"`, and Evidence cites at
  least one Finding ID

---

### Requirement: Package inspectors emit vendor + maintainer

The RPM inspector SHALL include the package's Vendor field in
the Finding's `vendor` attribute. The DPKG inspector SHALL
include the package's Maintainer field in the Finding's
`maintainer` attribute. Both attributes carry the raw upstream
value; classification happens in the assessor.

#### Scenario: RPM emits vendor

- **GIVEN** rpm output includes a Fedora-built bash with Vendor
  `"Fedora Project"`
- **WHEN** the agent inspects
- **THEN** the resulting `inventory.packages.rpm` Finding
  carries `vendor: "Fedora Project"` in Attributes

#### Scenario: DPKG emits maintainer

- **GIVEN** dpkg output includes a postgres-server with
  Maintainer
  `"Debian PostgreSQL Maintainers <team+pg@tracker.debian.org>"`
- **WHEN** the agent inspects
- **THEN** the resulting `inventory.packages.dpkg` Finding
  carries the full Maintainer string on `maintainer`

#### Scenario: Locally-built RPMs do not produce a vendor attribute

- **GIVEN** an rpm whose `%{VENDOR}` is `"(none)"` (the
  placeholder for locally-built or unsigned packages)
- **WHEN** the agent inspects
- **THEN** the resulting Finding has NO `vendor` attribute,
  so the classifier falls through to "unknown" rather than
  attributing on placeholder noise

---

### Requirement: wand.host.eu_package_origin classifies package vendor jurisdiction

The assessor SHALL register `wand.host.eu_package_origin`,
reading `inventory.packages.*` findings and classifying each
finding's vendor / maintainer against
`internal/assessor/package_vendors.yaml`. Scoring:

- `afhankelijk` on any single US-tied vendor hit
- `soeverein` when every classified package resolves to an
  EU-tied vendor (with negative-evidence sample)
- `voldoende` when no US hits AND not every classified
  package is EU-tied (mixed / unclassified)
- `onbekend` without `inventory.packages.*` findings

#### Scenario: Fedora host scores afhankelijk

- **GIVEN** every `inventory.packages.rpm` Finding carries
  `vendor: "Fedora Project"`
- **WHEN** the assessor runs `wand.host.eu_package_origin`
- **THEN** the Score is `afhankelijk` and the Verdict names
  Red Hat as the parent_org of record

#### Scenario: Mixed EU + unclassified scores voldoende

- **GIVEN** an inventory where some packages classify EU and
  some have no classifiable vendor
- **WHEN** the assessor runs the rule
- **THEN** the Score is `voldoende` — no red flag, no positive
  sovereign call possible
