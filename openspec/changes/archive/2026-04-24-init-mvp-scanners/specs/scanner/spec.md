# Delta for scanner

## ADDED Requirements

### Requirement: Single-domain scan execution

The system SHALL accept a single domain name as a scan target and execute
the full MVP probe suite against it, producing a `Scan` record with a
collection of `Finding` records.

#### Scenario: Happy path — responsive domain

- **Given** a domain `example.nl` that resolves, serves HTTPS, and returns
  an HTML homepage
- **When** the operator runs `wanderer scan example.nl`
- **Then** the resulting Scan has status `complete`
- **And** Findings are present from each of the four probes
- **And** the process exits with code 0

#### Scenario: Partial scan — one probe fails

- **Given** a domain that resolves but whose HTTPS endpoint times out
- **When** the scan runs
- **Then** the Scan has status `partial`
- **And** Findings from `dns`, `ip`, and `http` (if reachable over HTTP)
  are still recorded
- **And** a Finding of severity `info` records that the TLS probe timed out
- **And** the process exits with code 0 (partial is not an error)

#### Scenario: Failed scan — domain does not resolve

- **Given** a domain that returns NXDOMAIN
- **When** the scan runs
- **Then** the Scan has status `failed`
- **And** a single Finding records the resolution failure
- **And** the process exits with code 1

---

### Requirement: Passive observation boundary

The system SHALL NOT send any network traffic to the target beyond what a
normal DNS client, TLS client, or browser would send when simply resolving,
connecting to, or fetching the apex page of the domain.

#### Scenario: No subdomain enumeration

- **Given** a target domain
- **When** the scan runs
- **Then** the only DNS queries issued are for the target itself and any
  hosts discovered through CNAME, MX, or third-party HTTP resources observed
  in the response

#### Scenario: No port scanning

- **Given** a target domain
- **When** the scan runs
- **Then** the only TCP connections made to the target are on ports 80 and
  443

#### Scenario: robots.txt respected

- **Given** a target whose `robots.txt` disallows `/`
- **When** the HTTP probe runs
- **Then** no HTTP fetch of `/` is attempted
- **And** a Finding of severity `info` records `http.robots_blocked`

---

### Requirement: Finding schema stability

Every `Finding` produced by any probe SHALL conform to the `pkg/models.Finding`
shape, and the fields `ProbeID`, `Subject`, `Severity`, and `Attributes`
MUST be populated.

#### Scenario: Evidence is retained

- **Given** the TLS probe inspects a certificate chain
- **When** a Finding is recorded
- **Then** the raw certificate (PEM form) is stored in `Evidence`
- **And** the Finding can be re-evaluated later without re-scanning

#### Scenario: Probe-specific data stays in Attributes

- **Given** the DNS probe finds an MX record
- **When** the Finding is recorded
- **Then** the MX host and preference live in `Attributes`, not in a
  top-level Finding field
- **And** the assessor can read this via the shared JSON structure without
  depending on the `probe/dns` package

---

### Requirement: Scan persistence and retrieval

The system SHALL persist every Scan and its Findings such that they are
retrievable by scan ID after the process that produced them has exited.

#### Scenario: Scan survives restart

- **Given** a completed scan with ID `s_abc`
- **When** the server process restarts
- **And** the operator requests `GET /scans/s_abc`
- **Then** the response contains the full Scan record and all associated
  Findings

#### Scenario: Idempotent scan writes

- **Given** a scan that writes N Findings to the store
- **When** the same scan is replayed (e.g. a retry of a persistence failure)
- **Then** the store contains exactly N Findings for that scan ID, not 2N

---

### Requirement: Per-probe isolation

A panic or timeout in one probe SHALL NOT prevent the remaining probes from
running to completion.

#### Scenario: Panic in one probe

- **Given** a hypothetical probe that panics mid-run
- **When** the scanner executes it
- **Then** the panic is recovered and recorded as a Finding of severity
  `concern` with `probe.panic` in Attributes
- **And** the remaining probes run to completion
- **And** the Scan status reflects `partial`
