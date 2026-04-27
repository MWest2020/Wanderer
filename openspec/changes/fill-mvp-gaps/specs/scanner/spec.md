# Delta for scanner

## ADDED Requirements

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
