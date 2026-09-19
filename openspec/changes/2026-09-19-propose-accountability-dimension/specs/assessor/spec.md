# Delta for assessor

> New wand dimension `accountability`, two operationeel rules, and the
> generic reason-code mechanism (reused by the standards change).

## ADDED Requirements

### Requirement: Registrant identifiability is scored

The wand rule pack SHALL score whether the target's RDAP registrant is
identifiable and matches a name in the `config.expected_registrant`
finding recorded for the scan's organisation. The rule SHALL check, in
this order: when the target's TLD is listed in
`registry_redaction.yaml` the rule SHALL score onbekend with reason
`registry_redacted`, whatever the vcard says; when the registrant
matches `privacy_proxies.yaml` it SHALL score afhankelijk; when the
name is present and matches an expected name it SHALL score
soeverein; present but unmatched SHALL score voldoende; when RDAP data
is unavailable it SHALL score onbekend with reason
`probe_unavailable`. The verdict SHALL name the observed registrant
string, the proxy, or the registry.

#### Scenario: Declared registrant matches

- **GIVEN** an organisation with `expected_registrant: ["Gemeente
  Voorbeeld"]` recorded as `config.expected_registrant`, a `.com`
  target, and a `whois.registrant_identity` finding of "Gemeente
  Voorbeeld B.V."
- **WHEN** the assessor runs the wand rule pack
- **THEN** `wand.accountability.registrant_identifiable` scores
  soeverein and names the registrant

#### Scenario: Commercial privacy proxy scores afhankelijk

- **GIVEN** a `.com` target whose `whois.registrant_identity` matches
  the privacy-proxy list ("Domains By Proxy, LLC")
- **WHEN** the assessor runs
- **THEN** the rule scores afhankelijk and the verdict names the proxy

#### Scenario: Registry redaction is not the organisation's choice

- **GIVEN** a `.nl` target (TLD in `registry_redaction.yaml`) whose
  registrant vcard reads "REDACTED FOR PRIVACY"
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend with reason `registry_redacted`
- **AND** the verdict states that the registry does not publish
  registrant data for .nl, not that the registrant hides

#### Scenario: RDAP unavailable scores onbekend

- **GIVEN** only a `whois.unavailable` finding (RDAP 429 rate-limit)
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend with reason `probe_unavailable`
  and the verdict names the RDAP failure, not a guess

---

### Requirement: Direct registrar relationship is scored

The wand rule pack SHALL score afhankelijk when RDAP reports a
reseller entity between registrant and registrar, soeverein when the
registrar relationship is direct, and onbekend when RDAP data is
unavailable. The verdict SHALL name the reseller when present.

#### Scenario: Reseller present

- **GIVEN** a `whois.reseller` finding naming "Cheap Domains BV"
- **WHEN** the assessor runs
- **THEN** `wand.accountability.no_reseller` scores afhankelijk and
  names the reseller

#### Scenario: Organisation is its own registrar

- **GIVEN** the RDAP fixture of `rijksoverheid.nl` (registrar
  "Rijksoverheid", no reseller entity at any nesting level)
- **WHEN** the assessor runs
- **THEN** `wand.accountability.no_reseller` scores soeverein

#### Scenario: RDAP redacted hides the reseller field

- **GIVEN** a whois scan where entity roles could not be parsed
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend

---

### Requirement: SOA RNAME contactability is scored

The wand rule pack SHALL score the RFC 2142 / RFC 1035 contactability
of the zone's SOA RNAME: soeverein when the RNAME mailbox domain
resolves and publishes MX, voldoende when it resolves without MX,
afhankelijk when it does not resolve, onbekend when the SOA query
failed. The rule SHALL NOT claim mailbox delivery was verified.

#### Scenario: Functioning RNAME domain

- **GIVEN** `dns.soa` with RNAME `hostmaster.voorbeeld.nl` whose
  domain resolves and has MX
- **WHEN** the assessor runs
- **THEN** `wand.accountability.soa_rname` scores soeverein and the
  verdict states delivery itself was not verified

#### Scenario: Dangling RNAME

- **GIVEN** an RNAME whose mailbox domain returns NXDOMAIN
- **WHEN** the assessor runs
- **THEN** the rule scores afhankelijk and names the dangling domain

#### Scenario: SOA query timeout

- **GIVEN** a `dns.soa.unavailable` finding
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend

---

### Requirement: security.txt presence and freshness are scored

The wand rule pack SHALL score RFC 9116 compliance: soeverein when
`/.well-known/security.txt` is present, parseable, and carries a
Contact and an unexpired Expires; voldoende when present but expired
or missing fields; afhankelijk when absent (HTTP 404 is a valid
observation, not an error); onbekend when the fetch failed at
transport level.

#### Scenario: Valid security.txt

- **GIVEN** an `http.securitytxt` finding with Contact and Expires
  30 days in the future
- **WHEN** the assessor runs
- **THEN** `wand.accountability.securitytxt` scores soeverein

#### Scenario: Expired security.txt

- **GIVEN** an `http.securitytxt` finding whose Expires lies in the
  past
- **WHEN** the assessor runs
- **THEN** the rule scores voldoende and the verdict names the expiry
  date

#### Scenario: Transport failure

- **GIVEN** an `http.securitytxt.unavailable` finding (TLS timeout)
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend

---

### Requirement: Nameserver holder transparency is scored

The wand rule pack SHALL score whether the registrable domain of each
authoritative nameserver has an identifiable RDAP registrant:
soeverein when all NS holder lookups return an unproxied registrant,
voldoende when some do, afhankelijk when none do, onbekend when no NS
holder could be looked up. The verdict SHALL list opaque NS domains.

#### Scenario: Mixed transparency

- **GIVEN** two `whois.ns_holder` findings, one identifiable and one
  privacy-proxied
- **WHEN** the assessor runs
- **THEN** `wand.accountability.ns_holder_transparent` scores
  voldoende and names the opaque nameserver domain

#### Scenario: All NS lookups rate-limited

- **GIVEN** only `whois.ns_holder.unavailable` findings
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend

---

### Requirement: Domain expiry is scored

The wand rule pack SHALL score the registration expiry event under
the operationeel dimension: soeverein beyond 90 days, voldoende
within 90 days, afhankelijk within 30 days or past. When RDAP answered
but the registry publishes no expiration event the rule SHALL score
onbekend with reason `not_published_by_registry`; when RDAP was
unavailable it SHALL score onbekend with reason `probe_unavailable`.
The verdict SHALL name the date.

#### Scenario: Expiry within 30 days

- **GIVEN** a `whois.expiry` finding 12 days in the future
- **WHEN** the assessor runs
- **THEN** `wand.operationeel.domain_expiry` scores afhankelijk and
  names the date

#### Scenario: Registry publishes no expiry

- **GIVEN** the `rijksoverheid.nl` RDAP fixture (no expiration event)
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend with reason
  `not_published_by_registry`
- **AND** the rule does not lower the operationeel dimension's
  completeness

---

### Requirement: Variant convergence is scored

The wand rule pack SHALL score, under the operationeel dimension,
whether all observed apex/www × IPv4/IPv6 × http/https paths converge
on a single canonical HTTPS origin: soeverein when every observed live
path converges, voldoende when paths diverge or a family (v4/v6) is
dead while its DNS record exists and the scanner could reach that
family, afhankelijk when a plain-HTTP path serves content without
redirecting to HTTPS, onbekend when the variants probe failed
entirely. Paths recorded as `not_followed_budget` or `not_tested`
SHALL NOT count as dead; the verdict SHALL say how many paths were
observed.

#### Scenario: Full convergence

- **GIVEN** an `http.variants` finding where all eight paths
  converge on `https://www.voorbeeld.nl`
- **WHEN** the assessor runs
- **THEN** `wand.operationeel.variant_convergence` scores soeverein

#### Scenario: HTTP serves without redirect

- **GIVEN** an `http.variants` finding where `http://voorbeeld.nl`
  answers 200 with content
- **WHEN** the assessor runs
- **THEN** the rule scores afhankelijk and names the offending path

#### Scenario: Scanner without IPv6 does not blame the target

- **GIVEN** an `http.variants` finding whose four v6 paths are
  `not_tested` with reason `scanner_no_ipv6` and whose v4 paths
  converge
- **WHEN** the assessor runs
- **THEN** the rule scores soeverein on the observed v4 paths and does
  not score voldoende for a dead v6 family
- **AND** the verdict states that 4 of 8 paths were observed
- **AND** the rationale carries reason `scanner_no_ipv6`

#### Scenario: Probe timeout

- **GIVEN** an `http.variants.unavailable` finding
- **WHEN** the assessor runs
- **THEN** the rule scores onbekend with reason `probe_unavailable`

---

### Requirement: Rule results carry generic reason codes

A rule result MAY carry a `reason` code, persisted as `reason`
(omitted when empty) on the rationale. Every code SHALL be registered
in one table with exactly one class: `structural` (not applicable) or
`gap` (measurement hole), and one subject: `target` (default) or
`scanner` (a limitation of Wanderer's own environment). A rationale with a reason SHALL score
onbekend; the four-value scale SHALL NOT be extended. Emitting a code
absent from the table SHALL fail the assessor's tests. The mechanism
SHALL NOT be specific to any dimension or pack.

#### Scenario: Seeded codes are registered

- **WHEN** the reason-code table is loaded
- **THEN** it contains `registry_redacted` and
  `not_published_by_registry` and `scanner_no_ipv6` as structural,
  and `probe_unavailable` as gap
- **AND** `scanner_no_ipv6` has subject `scanner`, all others subject
  `target`

#### Scenario: Old assessment JSON still loads

- **GIVEN** an assessment stored before this change (no `reason`
  fields, no `accountability` entry)
- **WHEN** it is loaded and rendered
- **THEN** it loads without error and its scores are unchanged

---

### Requirement: Structural reasons do not count in aggregation

When scoring a dimension the assessor SHALL exclude rationales with a
`structural` reason from both the worst-score computation and the
completeness denominator. A `gap` reason SHALL count as a missing
observation, as an evidence-less rule does today. A dimension whose
every rationale is structural SHALL be reported as not applicable and
left out of any overall score.

#### Scenario: n.v.t. does not drag completeness down

- **GIVEN** an operationeel dimension where `domain_expiry` is
  structural (`not_published_by_registry`) and every other rule has
  evidence
- **WHEN** the dimension is scored
- **THEN** its completeness is `complete`

#### Scenario: A gap still counts

- **GIVEN** an accountability dimension where `soa_rname` scored
  onbekend with reason `probe_unavailable`
- **WHEN** the dimension is scored
- **THEN** its completeness is `partial`

## MODIFIED Requirements

### Requirement: CLI output formats

The `wanderer assess <scan-id>` command SHALL support at least three
output formats: human-readable text, markdown, and JSON.

#### Scenario: Markdown format

- **Given** a completed Scan
- **When** `wanderer assess <id> --format markdown` runs
- **Then** stdout contains a markdown document starting with
  `# Wanderer Assessment`
- **And** every dimension section includes a heading, a one-line
  verdict, and an "Evidence:" line per rule

#### Scenario: JSON format

- **Given** a completed Scan
- **When** `wanderer assess <id> --format json` runs
- **Then** stdout is a valid JSON object matching the `Assessment`
  schema in `pkg/models`
- **And** `.dimensions` has one entry per `WandDimensions` entry, in
  that order (no test asserts a fixed count)

#### Scenario: Missing scan

- **Given** a scan ID that does not exist in the store
- **When** `wanderer assess <id>` runs
- **Then** the process exits non-zero
- **And** stderr contains "scan not found"
