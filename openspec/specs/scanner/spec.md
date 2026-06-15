# scanner Specification

## Purpose
TBD - created by archiving change init-mvp-scanners. Update Purpose after archive.
## Requirements
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

---

### Requirement: Scanner runs in two passes so the IP probe sees discovered hosts

The scanner SHALL execute perimeter probes in two passes — pass 1
runs DNS, TLS, and HTTP concurrently; pass 2 runs the IP probe
with a Target whose `Related` is the union of the operator-supplied
related list and every host extracted from pass-1 findings whose
`ProbeID` is `dns.mx`, `http.third_party`, or `dns.subdomain` —
so jurisdictional rules that depend on cross-probe correlation
(`mx_vendor_jurisdiction`, `third_parties_eea`, the third-party arm
of `no_us_hyperscaler`) actually receive evidence on real scans.

#### Scenario: MX host gets IP-resolved

- **Given** a target whose DNS probe returns `dns.mx` with
  `host: mail.example.nl`
- **When** the scanner runs against that target
- **Then** the IP probe is invoked with `mail.example.nl` in its
  Related list
- **And** an `ip.asn` Finding is emitted with `Subject:
  mail.example.nl`

#### Scenario: Pass-1 probes do not see their own discoveries

- **Given** the same target
- **When** the scanner runs
- **Then** the DNS, TLS, and HTTP probes each receive the original
  Target (without the discovered hosts merged in)
- **And** only the IP probe's input Target carries the enriched
  Related list

---

### Requirement: Probes within a pass run concurrently with panic isolation

The scanner SHALL run every probe inside a single pass concurrently
via `errgroup`, with each probe wrapped in a recover that converts
a panic into a logged error and a partial-scan outcome rather than
crashing the process, and SHALL respect the configured global
budget as the single deadline for the whole scan.

#### Scenario: Slow probe does not block fast ones

- **Given** three pass-1 probes whose Run methods take 5s, 1s, 1s
- **And** a global budget of 30s
- **When** the scanner runs
- **Then** wall-clock duration for pass 1 is approximately 5s, not
  7s

#### Scenario: Probe panic does not abort the scan

- **Given** one probe whose Run panics
- **When** the scanner runs
- **Then** the resulting Scan has `Status: partial`
- **And** Findings from the surviving probes are persisted
- **And** an `scanner.probe_panic` info Finding records the
  panicking probe's ID

---

### Requirement: HTTP and TLS dialers refuse private and metadata addresses by default

The scanner's HTTP and TLS probes SHALL resolve target hostnames
through a SafeDialer that rejects connections whose destination IP
falls in loopback, link-local, RFC1918, CGNAT, IPv6 ULA, IPv6
link-local, or any well-known cloud metadata range, unless the
operator has set `--allow-private-targets`, so the public
`POST /scans` endpoint cannot be coerced into scanning internal
infrastructure.

#### Scenario: RFC1918 target rejected

- **Given** an attacker calls `POST /scans` with
  `{"domain": "internal.example.nl"}` whose only A record is
  `10.0.0.5`
- **And** the operator has not set `--allow-private-targets`
- **When** the HTTP probe attempts to dial
- **Then** the dial returns the SSRF-block error
- **And** a `http.fetch_failed` Finding records the block reason
- **And** no TCP connection is opened to `10.0.0.5`

#### Scenario: Cloud metadata IP rejected

- **Given** a target whose A record resolves to `169.254.169.254`
- **When** the scanner attempts to dial
- **Then** the dial is refused
- **And** the operator-readable error names cloud-metadata as the
  cause

#### Scenario: Operator can opt out

- **Given** an internal-network operator runs
  `wanderer scan internal.example --allow-private-targets`
- **When** the scanner dials a target with an RFC1918 address
- **Then** the dial proceeds normally
- **And** Findings are produced for the internal target

---

### Requirement: Scanner mines passive subdomains from CT logs and a common-prefix probe

The scanner SHALL passively discover subdomains of the target by
extracting SAN entries from the existing crt.sh response and by
running a fixed-list common-prefix DNS probe (the 18 names listed
in the design), emit each resolving name as a `dns.subdomain`
Finding with a `source` attribute identifying its origin, and feed
the discovered names into pass 2 so they receive ASN/country
annotation.

#### Scenario: SAN discovery

- **Given** a crt.sh response listing `mail.example.nl` and
  `vpn.example.nl` as SAN names for the target
- **When** the TLS probe processes the response
- **Then** two `dns.subdomain` Findings are emitted with
  `Attributes.source: ct_log`

#### Scenario: Common-prefix discovery

- **Given** a target `example.nl` whose `www.example.nl` resolves
  but `auth.example.nl` does not
- **When** the DNS probe runs the common-prefix sweep
- **Then** exactly one `dns.subdomain` Finding is emitted, for
  `www.example.nl`, with `Attributes.source: prefix_probe`
- **And** no Finding is emitted for non-resolving prefixes

#### Scenario: Wildcard collapse

- **Given** a target whose every prefix resolves to the same
  single A record (a wildcard)
- **When** the prefix sweep runs
- **Then** exactly one `dns.subdomain.wildcard` Finding is emitted
- **And** no per-prefix `dns.subdomain` Findings are emitted

---

### Requirement: Scanner accepts an Amass JSON file as additional related hosts

The scanner SHALL merge every FQDN in an Amass `enum -json` output
file into `target.Related` before pass 1 begins when one is
supplied via `--amass <path>` (CLI) or an `amass_json` field on
`POST /scans`, treating malformed lines as warnings rather than
fatal errors so a partial Amass file does not block the scan.

#### Scenario: Amass file ingested

- **Given** an Amass JSONL file with three FQDNs
  (`mail.example.nl`, `vpn.example.nl`, `www.example.nl`)
- **When** the operator runs `wanderer scan example.nl --amass
  out.json`
- **Then** the IP probe in pass 2 receives those three names in
  its Related list (in addition to anything the perimeter probes
  discovered)

#### Scenario: Missing file is fatal

- **Given** `--amass /does/not/exist`
- **When** the operator runs `wanderer scan`
- **Then** the process exits non-zero
- **And** stderr names the missing path

#### Scenario: Malformed line is skipped

- **Given** an Amass file containing one valid line and one line
  that is not JSON
- **When** the scanner ingests it
- **Then** the valid FQDN is merged into Related
- **And** the malformed line is logged at WARN level
- **And** the scan continues

---

### Requirement: Scanner ships an RDAP-based WHOIS probe

The scanner SHALL include a WHOIS probe that calls
`https://rdap.org/domain/<domain>` with a bounded timeout, emits
`whois.registrant` and `whois.registrar` Findings on success, and
emits exactly one `whois.unavailable` Finding on failure (network
error, non-200 status, or parse error) so the assessor can score
registrar jurisdiction.

#### Scenario: RDAP success

- **Given** an RDAP service returning a domain document with a
  registrant entity in NL
- **When** the WHOIS probe runs
- **Then** a `whois.registrant` Finding is emitted with
  `Attributes.country: "NL"`
- **And** a `whois.registrar` Finding is emitted with the
  registrar's name

#### Scenario: RDAP timeout

- **Given** an RDAP endpoint that hangs past the 5-second timeout
- **When** the WHOIS probe runs
- **Then** exactly one `whois.unavailable` Finding is emitted
- **And** the rest of the scan continues

### Requirement: Organisations group Targets

Wanderer SHALL model an `Organisation` as a first-class entity
that groups one or more Targets (perimeter `Kind=domain`,
agent-host `Kind=host`, or a mix). Every Target MUST belong to
exactly one Organisation. A seeded `default` Organisation
SHALL exist on every freshly-migrated store, and any Target that
predates the migration MUST be attached to `default` by the
backfill step.

#### Scenario: New Target without explicit organisation

- **Given** an instance with the seed `default` Organisation
- **When** an operator runs `wanderer scan example.nl` without
  `--organisation`
- **Then** the resulting Target is attached to `default`

#### Scenario: Existing Target after migration

- **Given** a pre-migration store with N Targets and zero
  Organisations
- **When** migration 004 runs
- **Then** the `default` Organisation is created
- **And** every existing Target's `organisation_id` is set to
  the `default` Organisation's ID
- **And** the column is NOT NULL after the backfill

---

### Requirement: Organisation slugs are unique and validated

Organisation slugs SHALL be 2–40 characters, lowercase letters,
digits, and hyphens only, MUST NOT start or end with a hyphen,
and MUST be unique across the store. The slug is the operator-
facing handle (used in `--organisation <slug>` and the URL
`/ui/orgs/{slug}`); the Name is the display label.

#### Scenario: Invalid slug rejected

- **Given** an operator runs `wanderer org add --slug -bad
  --name "Bad"`
- **When** the command processes the slug
- **Then** the command exits non-zero
- **And** the error names the slug rule that failed

### Requirement: Wanderer marketplace app is a separately-tracked product surface

A Wanderer Nextcloud marketplace app, if pursued, SHALL ship
as a separate top-level surface — its own directory, its own
release cadence, its own quality bar — and SHALL NOT introduce
PHP / Composer / Nextcloud-app dependencies into the core Go
codebase. The picked architecture (A: PHP shim + Go sidecar,
B: PHP reimplementation, C: WebAssembly) MUST be recorded in
`add-nextcloud-marketplace-app`'s status block before any
code lands.

#### Scenario: Marketplace surface does not pollute the core Go module

- **GIVEN** an active marketplace app
- **WHEN** a contributor runs `go test ./...` from the repo
  root
- **THEN** no PHP / Composer / Nextcloud-app dependency is
  required to pass

