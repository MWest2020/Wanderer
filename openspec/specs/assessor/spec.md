# assessor Specification

## Purpose
TBD - created by archiving change add-assessor. Update Purpose after archive.

## Requirements

### Requirement: Deterministic assessment

The assessor SHALL produce the same Assessment for the same set of
Findings, regardless of when or where it runs.

#### Scenario: Replay produces identical score

- **Given** a Scan `s_abc` with a frozen set of Findings
- **When** the assessor runs against it now
- **And** the assessor runs again an hour later without new Findings
- **Then** the two Assessments have identical per-dimension Scores
- **And** identical Rationale lists (ignoring ID and CreatedAt)

#### Scenario: Same findings, two machines

- **Given** the same Scan JSON exported from one Wanderer instance
- **When** a second Wanderer instance imports it and runs the assessor
- **Then** the resulting Assessment has identical Scores per dimension

---

### Requirement: Completeness transparency

The assessor SHALL record a `Completeness` flag per dimension so a
reader can distinguish "low score due to bad posture" from "low score
due to missing evidence".

#### Scenario: Perimeter-only scan

- **Given** a Scan produced entirely by the MVP probes (no inventory,
  no egress)
- **When** the assessor runs
- **Then** Data & AI dimension has `Completeness: incomplete` for rules
  requiring egress/OIDC data
- **And** the markdown report explicitly states this in the dimension
  section
- **And** the Score for an `incomplete` dimension is `onbekend`
  unless evidence-backed rules gave a lower score

#### Scenario: IP probe unavailable

- **Given** a Scan where `ip.unavailable` is the only IP-related
  finding
- **When** the assessor runs rules depending on IP ASN country
- **Then** those rules do not contribute a score
- **And** the dimension's Completeness is reduced from `complete` to
  `partial` or `incomplete`

---

### Requirement: Evidence citations

Every Rationale entry in an Assessment SHALL reference at least one
`Finding.ID` in its `Evidence` field, or be marked as
"no evidence — rule did not match".

#### Scenario: Verdict cites evidence

- **Given** the `Juridisch` dimension rule "Cert issuer in EU"
- **When** the rule matches a `tls.issuer` finding with
  `issuer_country: ["US"]`
- **Then** the resulting Rationale contains the finding's `ID` in its
  `Evidence` list
- **And** the markdown report renders the citation as
  `Evidence: tls.issuer (finding f_xyz)`

#### Scenario: Missing attribute handled gracefully

- **Given** a `tls.issuer` finding whose `Attributes` map lacks
  `issuer_country`
- **When** the corresponding rule runs
- **Then** the rule does not panic
- **And** the rule either contributes no verdict or contributes
  `Score: onbekend` with a verdict explaining the missing attribute
- **And** a `slog.Warn` is logged

---

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

### Requirement: Assessments are persisted

Every run of the assessor against a Scan via the HTTP API SHALL
produce a persisted `Assessment` record retrievable by its ID.

#### Scenario: Assessment survives restart

- **Given** an Assessment `a_abc` produced via `POST /scans/{id}/assessments`
- **When** the server restarts
- **And** the operator requests `GET /assessments/a_abc`
- **Then** the response contains the full Assessment record

#### Scenario: Re-running produces a new record

- **Given** an Assessment `a_1` already persisted for scan `s_abc`
- **When** `POST /scans/s_abc/assessments` is called again
- **Then** a new Assessment `a_2` is persisted
- **And** `a_1` remains retrievable

---

### Requirement: Assessor ships a SEAL rule pack alongside DICTU

The assessor SHALL provide a second rule pack implementing the EU
Cloud Sovereignty Framework (SEAL) under `internal/assessor/eucsf`,
selectable per assessment via `wanderer assess --framework
dictu|eucsf|both`, scoring on a 0–4 SEAL level scale that maps onto
the existing `models.Score` enum so the engine and the persisted
Assessment shape stay unchanged.

#### Scenario: SEAL framework selectable

- **Given** a stored Scan with TLS issuer attributes from a Dutch CA
- **When** the operator runs `wanderer assess <id> --framework eucsf`
- **Then** the resulting Assessment has `Framework: "eucsf"`
- **And** every Rationale's `CriteriumID` starts with `eucsf.`

#### Scenario: Both frameworks persist independently

- **Given** the same scan
- **When** the operator runs `wanderer assess <id> --framework both`
- **Then** two Assessments are persisted — one with
  `Framework: "dictu"` and one with `Framework: "eucsf"`
- **And** both cite the same Findings via their Evidence lists

#### Scenario: SEAL level maps to model Score

- **Given** an `eucsf.sov2.cert_issuer_eu` rule that fires SEAL level 4
- **When** the rule is evaluated
- **Then** the resulting Rationale has `Score: soeverein`
- **And** the SEAL level is recorded in `Attributes.seal_level: "seal_4"`

---

### Requirement: Probe-ID / rule-ID drift is build-breaking

Both the DICTU and SEAL rule packs SHALL each ship at least one
integration test that runs the relevant real probe against a fake
resolver / fake HTTP source through the real assessor, asserting
the actual ProbeID / attribute names the rule pack consumes, so a
casing mismatch or an attribute rename in either side breaks `go
test ./...` rather than silently producing Onbekend on production
scans.

#### Scenario: Casing change breaks the build

- **Given** the DNS probe currently emits `dns.a` and the
  `apex_ip_eea` rule consumes `dns.a`
- **When** a contributor accidentally changes one side to `dns.A`
- **Then** the integration test in
  `internal/assessor/dictu/integration_test.go` fails

#### Scenario: Attribute rename breaks the build

- **Given** the TLS probe currently writes `issuer_country` on
  `tls.issuer` and the `cert_issuer_eu` rule reads
  `issuer_country`
- **When** the probe renames the attribute to `issuerCountry`
- **Then** the eucsf integration test fails before the change can land

---

### Requirement: Rules ignore meta Findings

Every assessor rule SHALL skip Findings whose attributes mark them
as meta (an `error` attribute is present, `no_answer` is true, or
`unavailable` is true) when deciding whether evidence backs a
verdict, so a non-resolvable domain or a missing probe never
produces a positive score.

#### Scenario: NXDOMAIN does not score voldoende on MX presence

- **Given** a Finding set whose only `dns.mx` rows are
  `lookupError` records (each with an `error` attribute set)
- **When** the assessor runs the `dictu.data_ai.mx_present` rule
- **Then** the rule's RuleResult has `Score: onbekend`
- **And** the rule's `Evidence` list is empty

#### Scenario: A `no_answer` row is not evidence

- **Given** a Finding with `ProbeID: dns.caa` and
  `Attributes.no_answer: true`
- **When** the `dictu.operationeel.caa_restricts_issuance` rule
  evaluates the set
- **Then** the rule does not treat the row as a positive CAA
  observation
- **And** the rule's verdict reflects the absence of CAA records,
  not their presence

---

### Requirement: Every assessor Rule carries a plain-language Rationale

The `assessor.Rule` struct SHALL include a `Rationale string`
field that holds a one-paragraph plain-language explanation of
what the rule observes and why it matters for sovereignty
posture, populated alongside the existing `Description`. Every
Rule registered by `dictu.DefaultRules()` and
`eucsf.DefaultRules()` SHALL have a non-empty Rationale; an empty
string is a build-breaking error in the corresponding registry's
test suite.

#### Scenario: Rationale present on every default rule

- **GIVEN** the rule sets returned by
  `internal/assessor/dictu.DefaultRules()` and
  `internal/assessor/eucsf.DefaultRules()`
- **WHEN** a test iterates every Rule and reads `Rationale`
- **THEN** every Rule's `Rationale` is a non-empty string

#### Scenario: Empty Rationale fails CI

- **GIVEN** a contributor adds a new Rule with an empty
  `Rationale` field to either rule pack
- **WHEN** `go test ./internal/assessor/...` runs
- **THEN** the rule pack's `TestEveryRuleHasRationale` test fails
  with a message naming the offending `CriteriumID`

#### Scenario: Rationale is independent of Description

- **GIVEN** a Rule whose `Description` is a single-sentence
  summary ("TLS certificate issued by an authority in the EEA.")
- **WHEN** the renderer reads the Rule
- **THEN** `Rationale` is a separate string carrying the
  consequence of the rule firing
- **AND** `Description` and `Rationale` are not the same value

---

### Requirement: First-party rule pack is named `wand`, not `dictu`

The MWest2020-owned rule pack SHALL be identified as `wand`
(Wanderer-NL) in every output Wanderer produces: the persisted
`Assessment.Framework` value, the rule IDs (under the
`wand.<dimension>.<short>` shape), the CLI flag value
(`--framework wand|eucsf|both`), and every documentation or UI
surface that names the framework. The DICTU
*Toetsingsinstrument Soevereiniteit Clouddiensten* SHALL be
credited in the assessor docs and ADR-0011 as the public
framework that inspired the rule set; the implementation,
ownership, and label are MWest2020's.

#### Scenario: New assessment carries the wand framework label

- **GIVEN** the renamed rule pack is in production
- **WHEN** an operator runs `wanderer assess <scan-id> --framework wand`
- **THEN** the persisted `Assessment.Framework` is `"wand"`
- **AND** every persisted Rationale's `CriteriumID` starts with
  `wand.`

#### Scenario: Output documents the inspiration without claiming endorsement

- **GIVEN** a contributor reads `docs/assessor.md` after the rename
- **WHEN** they look for the relationship to DICTU
- **THEN** the doc explicitly names DICTU's *Toetsingsinstrument
  Soevereiniteit Clouddiensten* as the inspiration for the rule
  set
- **AND** the doc does not state or imply that DICTU endorses,
  certifies, or otherwise sanctions the Wanderer rule pack

---

### Requirement: Existing assessments are migrated from dictu to wand on store open

The schema migration runner SHALL convert every persisted
assessment row whose `framework = 'dictu'` to `framework =
'wand'` and rewrite every JSON-encoded `criterium_id` string
starting with `dictu.` to start with `wand.` instead. The
update SHALL run inside a single transaction so a partial
failure rolls back cleanly. This migration runs automatically
on `store.Open` once the new binary is in place.

#### Scenario: Pre-rename assessment becomes a wand assessment after open

- **GIVEN** a database containing one assessment row with
  `framework = 'dictu'` and a Rationale whose `criterium_id` is
  `dictu.juridisch.cert_issuer_eea`
- **WHEN** the new binary calls `store.Open` against that
  database
- **THEN** the migration runs to completion
- **AND** the row's `framework` column is `'wand'`
- **AND** the JSON-encoded Rationale's `criterium_id` is
  `wand.juridisch.cert_issuer_eea`

#### Scenario: Already-renamed row is left untouched

- **GIVEN** a database where the migration already ran (the
  assessments table contains rows with `framework = 'wand'` and
  `wand.*` criterium-IDs)
- **WHEN** `store.Open` runs again
- **THEN** the migration is not re-applied (its version is
  already in `schema_migrations`)
- **AND** no rows are modified

---

### Requirement: CLI accepts `dictu` as a deprecated alias for one release

`wanderer assess --framework dictu` SHALL continue to work for
exactly one release after this change ships, scoring the scan
against the `wand` rule pack and emitting one warning to
stderr that names the deprecation and the replacement
(`--framework wand`). The alias is removed in the release after.

#### Scenario: Legacy script still completes

- **GIVEN** an operator script invokes `wanderer assess <id>
  --framework dictu`
- **WHEN** the new binary runs the assessment
- **THEN** the persisted Assessment has `Framework: "wand"`
- **AND** stderr contains exactly one line beginning with
  `warning:` that names `--framework dictu` as deprecated and
  `--framework wand` as the replacement
- **AND** the exit code is 0 on success

#### Scenario: New invocation produces no warning

- **GIVEN** the same operator updates the script to
  `--framework wand`
- **WHEN** the binary runs
- **THEN** stderr contains no deprecation warning

---

### Requirement: Host-side findings produce a non-onbekend verdict

The assessor SHALL score agent-host scans (Targets with
`Kind=host`) on at least one rule per registered rule pack,
so a host scan produces a non-`onbekend` Assessment whenever
the agent's inspectors land their canonical Findings
(`inventory.packages.*`, `inventory.systemd.service`,
`egress.*` from the static scanner). Rules that target
perimeter ProbeIDs MUST continue to return `onbekend` on host
scans — they describe perimeter behaviour, not host
behaviour — but at least one host-shaped rule per pack must
fire on the agent's canonical findings.

#### Scenario: Agent scan produces a host-side verdict

- **GIVEN** an agent scan with `inventory.packages.rpm` and
  `inventory.systemd.service` findings
- **WHEN** the operator runs `wanderer assess <scan-id> --framework both`
- **THEN** the resulting Assessment has at least one
  dimension with a `soeverein`, `voldoende`, or `afhankelijk`
  score (not all `onbekend`)
- **AND** the host scan's verdict pill on `/ui/orgs/{slug}`
  renders that worst score

---

### Requirement: Host-rule soeverein verdicts cite negative evidence

A host-shaped rule SHALL cite at least one inspected Finding
ID in its `Evidence` slice and SHALL include the inspected
count in the Verdict text whenever it concludes soeverein.
This applies to rules reading `inventory.packages.*` or
`inventory.systemd.service`. The assessor engine forces
verdicts with empty Evidence back to `onbekend`, so this
keeps the soeverein call from being silently degraded.

#### Scenario: Clean host scores soeverein with evidence sample

- **GIVEN** an agent scan with 1790 `inventory.packages.rpm`
  Findings, none of which match the US-telemetry vendor list
- **WHEN** the assessor runs
  `wand.host.no_us_telemetry_packages`
- **THEN** the persisted Rationale has `Score: "soeverein"`,
  Verdict text containing `"inspected 1790 packages"`, and
  Evidence with 1..10 Finding IDs sampled from the inspected
  package Findings

### Requirement: Authoritative DNS jurisdiction is scored

The wand rule pack SHALL score the jurisdiction of a target's
authoritative nameservers by correlating the observed `dns.ns` hosts
with their `ip.asn` geo-lookups. When every located nameserver host
resolves to an EEA-registered AS the rule SHALL score soeverein; when
some do the rule SHALL score voldoende; when none do it SHALL score
afhankelijk; and when no nameserver could be located it SHALL score
onbekend. The verdict SHALL name the observed countries.

#### Scenario: EU-hosted nameservers score soeverein

- **GIVEN** a target whose `dns.ns` hosts all resolve to NL-registered AS
- **WHEN** the assessor runs the wand rule pack
- **THEN** `wand.juridisch.ns_vendor_jurisdiction` scores soeverein and
  names the country

#### Scenario: US-managed DNS scores afhankelijk

- **GIVEN** a target whose nameservers resolve to a US-registered AS
- **WHEN** the assessor runs
- **THEN** the rule scores afhankelijk and names the non-EEA jurisdiction

### Requirement: Passive HTTP exposure is scored

The wand rule pack SHALL score a target's passive HTTP exposure from
the observed security-header set and response banner. When HSTS is
absent it SHALL score afhankelijk; when HSTS is present but other
baseline headers are missing it SHALL score voldoende; when all
baseline headers are present it SHALL score soeverein; and when no
security-header observation exists it SHALL score onbekend. A Server /
X-Powered-By stack disclosure SHALL be named in the verdict. The rule
SHALL NOT perform any active or intrusive probing.

#### Scenario: Missing HSTS scores afhankelijk

- **GIVEN** a scan whose http.security_headers finding lists
  Strict-Transport-Security as missing
- **WHEN** the assessor runs
- **THEN** wand.operationeel.http_exposure scores afhankelijk and names
  the missing headers

### Requirement: Een regel draagt zijn drempels als waarde

Een `Rule` SHALL zijn beslisgrenzen als gegeven meedragen (naam,
waarde, eenheid), zodat de UI ze kan tonen en een test ze kan
controleren. Een regel die op een grens beslist SHALL die grens niet
alleen in zijn vergelijking hebben staan. Regels zonder grens (een
aanwezig-of-niet-controle) SHALL een lege lijst hebben.

#### Scenario: Grenzen zijn uitleesbaar

- **GIVEN** de regel `wand.operationeel.domain_expiry`
- **WHEN** de UI zijn grenzen opvraagt
- **THEN** krijgt die 90 dagen en 30 dagen terug, met hun betekenis

#### Scenario: Grens en vergelijking lopen niet uiteen

- **GIVEN** een regel waarvan de vergelijking een andere waarde
  gebruikt dan de meegedragen grens
- **WHEN** de tests draaien
- **THEN** falen ze

### Requirement: Een oordeelvlag draagt het getal waarop hij berust

Een probe die een finding met een oordeelvlag emitteert (bijvoorbeeld
"verloopt binnenkort") SHALL het getal meeleveren waarop die vlag is
gezet, met zijn eenheid. De vlag en dat getal SHALL uit dezelfde
constante komen, zodat ze niet uiteen kunnen lopen. De UI SHALL die
grens tonen bij de regel die op de vlag oordeelt, met de vermelding dat
de waarneming hem toepaste.

#### Scenario: Certificaat verloopt binnenkort

- **GIVEN** een certificaat dat over 20 dagen verloopt
- **WHEN** de tls-probe zijn finding emitteert
- **THEN** staat er naast de vlag dat de grens 30 dagen is

#### Scenario: Vlag en grens lopen uiteen

- **GIVEN** een probe waarvan de vlag op een andere waarde wordt gezet
  dan het meegeleverde getal
- **WHEN** de tests draaien
- **THEN** falen ze

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

### Requirement: Standards rules score imported Internet.nl verdicts only

The wand rule pack SHALL provide six rules under the `standards`
dimension — `wand.standards.dnssec`, `wand.standards.mail_auth`,
`wand.standards.starttls_dane`, `wand.standards.ipv6`,
`wand.standards.rpki`, `wand.standards.tls_config` — each scoring
solely from imported `internetnl.web.*` / `internetnl.mail.*`
findings by verdict mapping (all passed → soeverein, warnings/mixed →
voldoende, failed → afhankelijk, not_tested/absent/stale → onbekend).
The rules SHALL NOT recompute or aggregate Internet.nl's percentage
score, and no first-party Wanderer finding SHALL feed these rules.

#### Scenario: Signed and valid DNSSEC

- **GIVEN** imported findings where every dnssec-category test is
  passed
- **WHEN** the assessor runs the wand rule pack
- **THEN** `wand.standards.dnssec` scores soeverein and links the
  Internet.nl report URL in the evidence

#### Scenario: Failed mail authentication

- **GIVEN** imported findings where DMARC tests are failed
- **WHEN** the assessor runs
- **THEN** `wand.standards.mail_auth` scores afhankelijk and names
  the failing standard(s)

#### Scenario: No import present

- **GIVEN** a target with perimeter findings but no `internetnl.*`
  findings
- **WHEN** the assessor runs
- **THEN** every standards rule scores onbekend with reason "not
  measured" and no other dimension is affected

---

### Requirement: Stale measurements are not presented as current

The standards rules SHALL score onbekend, and the verdict SHALL name
the measurement date, when a target's newest `internetnl.*` findings
are older than the configured `standards.max_age` (default 30 days)
— so an old pass can never mask a regression.

#### Scenario: Measurement past max_age

- **GIVEN** imported findings with measured_at 45 days ago and the
  default max_age
- **WHEN** the assessor runs
- **THEN** `wand.standards.dnssec` scores onbekend with "measurement
  stale" and the date

#### Scenario: Fresh re-import restores scoring

- **GIVEN** a subsequent import with measured_at yesterday
- **WHEN** the assessor runs
- **THEN** the rules score from the fresh findings and the stale ones
  are ignored

---

### Requirement: Standards rules never double-score first-party ground

Categories where Wanderer keeps first-party probes for evidence granularity (security.txt, HTTPS variant convergence, security headers, certificate validity) SHALL remain scored exclusively by their existing first-party rules; the standards dimension SHALL NOT add rules over the corresponding Internet.nl subtests.

#### Scenario: security.txt stays first-party

- **GIVEN** an import whose web results include Internet.nl's
  security.txt subtest
- **WHEN** the assessor runs
- **THEN** no standards rule scores security.txt and
  `wand.accountability.securitytxt` scores solely from the
  first-party `http.securitytxt` finding
