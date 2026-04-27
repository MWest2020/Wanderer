# Delta for assessor

## ADDED Requirements

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
