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

### Requirement: CAA wordt echt opgezocht, inclusief de boom omhoog

De dns-probe SHALL CAA-records daadwerkelijk opvragen. Vindt zij op de
gevraagde naam niets, dan SHALL zij de boom omhoog lopen tot het
registreerbare domein, zoals RFC 8659 voorschrijft, en SHALL de finding
vastleggen op welke naam de records gevonden zijn. Een resolver die
stilzwijgend niets teruggeeft SHALL de tests laten falen. "Geen CAA"
SHALL alleen gemeld worden wanneer de hele keten leeg is.

#### Scenario: Records staan op het subdomein zelf

- **GIVEN** een naam met eigen CAA-records
- **WHEN** de dns-probe draait
- **THEN** staan die records in de finding, met die naam als herkomst

#### Scenario: Records staan op de apex

- **GIVEN** `iam.voorbeeld.nl` zonder eigen CAA en `voorbeeld.nl` met
  CAA-records
- **WHEN** de dns-probe draait
- **THEN** staan de records van de apex in de finding, met de
  vermelding dat ze van `voorbeeld.nl` komen

#### Scenario: Nergens CAA

- **GIVEN** een zone zonder CAA op enig niveau
- **WHEN** de dns-probe draait
- **THEN** meldt zij "geen CAA", en oordeelt de regel zoals nu

#### Scenario: Resolver geeft stilzwijgend niets

- **GIVEN** een resolver-implementatie die altijd een lege lijst zonder
  fout teruggeeft
- **WHEN** de tests draaien
- **THEN** falen ze

### Requirement: An agent enrols itself once and can be revoked

The core SHALL issue short-lived enrolment tokens and SHALL let an
agent exchange one for its own HMAC secret, recording the agent by
hostname with the moment it enrolled. A token SHALL be usable once and
SHALL expire. An operator SHALL be able to revoke one agent, after
which that agent's findings are refused while every other agent
continues. Secrets SHALL be stored so that the plain secret cannot be
read back from the core.

#### Scenario: First start of a new agent

- **GIVEN** an operator who created an enrolment token
- **WHEN** the agent starts with that token
- **THEN** it receives its own secret, writes it 0600, and the core
  lists the agent as enrolled

#### Scenario: Token used twice

- **GIVEN** a token that an agent already exchanged
- **WHEN** a second agent tries the same token
- **THEN** the exchange is refused and nothing is enrolled

#### Scenario: One agent revoked

- **GIVEN** two enrolled agents
- **WHEN** one is revoked and both send findings
- **THEN** the revoked agent is refused and the other is accepted

---

### Requirement: The served core accepts findings from enrolled agents

`wanderer serve` SHALL build its router with the agent secrets from the
store, so the findings route works for enrolled agents. With no agent
enrolled the route SHALL keep refusing every request, and the startup
log SHALL say that agent ingestion is inactive.

#### Scenario: Enrolled agent delivers

- **GIVEN** an enrolled agent and a running `wanderer serve`
- **WHEN** the agent posts findings signed with its secret
- **THEN** the core accepts them and they appear on the scan

#### Scenario: No agents enrolled

- **GIVEN** a core with no enrolled agents
- **WHEN** anything posts to the findings route
- **THEN** it is refused, and the startup log stated that agent
  ingestion is inactive

---

### Requirement: A replayed batch is stored once

Each batch of findings SHALL carry an identifier. The core SHALL store
a batch once; a repeat SHALL be answered as already received, without
storing the findings again. The agent's outbox SHALL keep the
identifier across a restart, so a batch that was spooled during an
outage is recognised when it is finally delivered.

#### Scenario: Outage and replay

- **GIVEN** an agent that spooled three batches while the core was down
- **WHEN** the core returns and the outbox drains, and the same drain
  runs twice
- **THEN** each batch's findings appear exactly once

#### Scenario: Restart keeps the identifier

- **GIVEN** a spooled batch and an agent that restarts before delivery
- **WHEN** the agent delivers it after the restart
- **THEN** the batch carries the identifier it was spooled with

### Requirement: whois probe emits identity, reseller, status, and expiry findings

The whois probe SHALL, from the same RDAP response it already
fetches, additionally emit `whois.registrant_identity` (name, kind,
privacy-proxy flag), `whois.reseller` (when a reseller entity is
present), `whois.status` (RDAP status codes), and `whois.expiry`
(the expiration event date, or `absent` when the registry publishes
none). Entities SHALL be parsed recursively, so a reseller nested
under the registrar entity is found. Redacted or missing fields SHALL
be emitted as explicit redacted/absent values, never omitted silently.

#### Scenario: Full RDAP response

- **GIVEN** an RDAP response with registrant vcard, reseller entity,
  status codes, and an expiration event
- **WHEN** the whois probe runs
- **THEN** all four new findings are emitted alongside the existing
  registrant/registrar findings

#### Scenario: Reseller nested under the registrar

- **GIVEN** an RDAP response whose reseller entity appears only inside
  the registrar entity's `entities` array
- **WHEN** the whois probe runs
- **THEN** `whois.reseller` names that reseller

#### Scenario: Organisation is its own registrar (live fixture)

- **GIVEN** the RDAP response for `rijksoverheid.nl` captured
  2026-09-19 (registrant "REDACTED FOR PRIVACY", registrar
  "Rijksoverheid", no reseller, events registration / last changed
  only)
- **WHEN** the whois probe runs
- **THEN** `whois.reseller` records no reseller and `whois.expiry`
  records `absent`

#### Scenario: Redacted vcard

- **GIVEN** an RDAP response whose registrant vcard is "REDACTED FOR
  PRIVACY"
- **WHEN** the whois probe runs
- **THEN** `whois.registrant_identity` is emitted with the proxy flag
  set, so the assessor can score afhankelijk instead of onbekend

---

### Requirement: SOA probe observes RNAME contactability

A new soa probe SHALL query the zone's SOA record and emit `dns.soa`
carrying MNAME and RNAME, plus the resolution result of the RNAME
mailbox domain (resolves yes/no, MX present yes/no). The probe SHALL
NOT open SMTP connections. On query failure it SHALL emit
`dns.soa.unavailable`.

#### Scenario: Normal zone

- **GIVEN** a zone whose SOA RNAME is `hostmaster.voorbeeld.nl`
- **WHEN** the soa probe runs
- **THEN** `dns.soa` reports the RNAME with resolves=true and the MX
  observation

#### Scenario: Lame delegation

- **GIVEN** a zone whose authoritative servers time out
- **WHEN** the soa probe runs
- **THEN** the probe emits `dns.soa.unavailable` within its timeout
  and other probes are unaffected

---

### Requirement: HTTP probe fetches security.txt

The HTTP probe SHALL fetch `/.well-known/security.txt` over HTTPS and
emit `http.securitytxt` with presence, parseability, Contact, and
Expires. A 404 SHALL be emitted as present=false (a valid
observation); only transport-level failure SHALL emit
`http.securitytxt.unavailable`.

#### Scenario: Present and valid

- **GIVEN** a target serving a parseable security.txt
- **WHEN** the HTTP probe runs
- **THEN** `http.securitytxt` carries Contact and Expires

#### Scenario: HTML error page at the path

- **GIVEN** a target answering 200 with an HTML page at the
  well-known path
- **WHEN** the HTTP probe runs
- **THEN** `http.securitytxt` is emitted with parseable=false

---

### Requirement: Variants probe observes path convergence

A new variants probe SHALL attempt the 8 paths apex/www × IPv4/IPv6 ×
http/https with redirect depth at most 5 per path, a hard budget of 24
connections per target across all paths, no retries, within the
scan's global timeout, and emit `http.variants` recording per path:
status (`reachable`, `unreachable`, `refused`, `not_followed_budget`,
`not_tested`), redirect chain, and final origin. A chain MAY stop as
soon as it reaches an origin already verified in the same run. Every
redirect hop SHALL pass the existing SSRF guard. When the scanner host
has no IPv6 route, v6 paths SHALL be recorded as `not_tested` with
reason `scanner_no_ipv6`. On total failure the probe SHALL emit
`http.variants.unavailable`.

#### Scenario: Converging paths

- **GIVEN** a target where all live paths redirect to
  `https://www.voorbeeld.nl`
- **WHEN** the variants probe runs
- **THEN** `http.variants` records convergence on that single origin

#### Scenario: Budget exhausted

- **GIVEN** a target whose paths each redirect 5 times before
  converging
- **WHEN** the variants probe runs
- **THEN** no more than 24 connections are opened
- **AND** the paths not reached are recorded as `not_followed_budget`

#### Scenario: Redirect into private address space

- **GIVEN** a path whose redirect chain points at 10.0.0.5
- **WHEN** the variants probe runs
- **THEN** the SSRF guard refuses the hop and the path is recorded as
  `refused`, not followed

---

### Requirement: Scanner looks up NS holder transparency

The scanner SHALL, for each unique registrable domain among the
target's `dns.ns` hosts, perform one RDAP lookup and emit
`whois.ns_holder` (registrant present/proxied/absent). Lookups SHALL
be cached per scan so N nameservers under one provider cost one
lookup; failures emit `whois.ns_holder.unavailable` per domain.

#### Scenario: Two providers, three nameservers

- **GIVEN** ns1/ns2.provider-a.nl and ns1.provider-b.eu
- **WHEN** the scan runs
- **THEN** exactly two RDAP lookups occur and two `whois.ns_holder`
  findings are emitted

#### Scenario: Registry without RDAP

- **GIVEN** a nameserver under a TLD with no RDAP service (404 from
  the bootstrap)
- **WHEN** the scan runs
- **THEN** `whois.ns_holder.unavailable` is emitted for that domain
  and the scan continues

---

### Requirement: Organisations declare expected registrant names

An organisation SHALL carry a list `expected_registrant` of names
(stored on the `organisations` table, default empty), set with
`wanderer org add <slug> --expected-registrant NAME` (repeatable;
`org add` upserts) and shown by `wanderer org show`. There SHALL be no
per-target override. At scan time the scanner SHALL record the
list of the scan's organisation as a `config.expected_registrant`
finding, also when the list is empty, so the assessment stays a
function of findings and shows which names were expected when the
scan ran.

#### Scenario: Declared names reach the assessment

- **GIVEN** organisation `voorbeeld` with `expected_registrant:
  ["Gemeente Voorbeeld"]`
- **WHEN** a scan of one of its targets runs
- **THEN** the scan contains a `config.expected_registrant` finding
  listing "Gemeente Voorbeeld"

#### Scenario: Changing the list does not rewrite history

- **GIVEN** a stored scan whose `config.expected_registrant` lists
  "Gemeente Voorbeeld"
- **WHEN** the organisation's list is later changed and the old scan
  is re-assessed
- **THEN** the re-assessment uses the names recorded in the old scan
