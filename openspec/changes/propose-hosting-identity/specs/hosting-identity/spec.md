# Delta for hosting-identity

> New capability — graduates from `research-high-signal-observability`
> (Wave 1, the fourth and last cheap who/where twin). Leads with the
> observed hosting fact; the existing `wand.juridisch.apex_ip_eea` rule
> annotates it. DICTU dimension: Juridisch (hosting-operator
> jurisdiction). Structural twin of DNS hosting and email routing.

## ADDED Requirements

### Requirement: Wanderer states who hosts a target's apex and where

Wanderer SHALL, after resolving a target's apex `A`/`AAAA` addresses and
attributing each to an ASN/country, emit one aggregate observed Finding
that names the recognisable hosting **operator** (derived from the
`ip.asn` organisation via a normalisation table that friendly-names the
common hosts and falls back to the raw organisation) and the **hosting
country** (when a GeoLite2 database is configured). The Finding SHALL read
as a plain statement — "{domain} is hosted at {operator} ({country})" —
and SHALL retain the raw ASN organisation and address as evidence so the
observed fact stands even when the friendly operator name is uncertain.
This Finding is an observation, not a verdict; the EEA-jurisdiction score
is carried separately by `wand.juridisch.apex_ip_eea`.

#### Scenario: Apex on a non-EEA host reads as a plain statement

- **GIVEN** a target whose apex `A` record resolves to an address on a
  US-registered AS organisation "AMAZON-02"
- **WHEN** the scan correlates `dns.a` with `ip.asn`
- **THEN** the scan contains an aggregate `ip.hosting` Finding naming the
  operator "AWS" and country "US",
- **AND** the raw ASN organisation and address are retained as evidence,
- **AND** `wand.juridisch.apex_ip_eea` still scores the apex as outside
  the EEA

#### Scenario: Unrecognised operator falls back to the raw organisation

- **GIVEN** an apex hosted on an AS organisation absent from the
  normalisation table
- **WHEN** the scan synthesises hosting identity
- **THEN** the Finding names the operator using the raw ASN organisation
  string, so the observed "who" still stands

#### Scenario: No GeoIP database degrades without failing

- **GIVEN** an instance with no GeoLite2 database configured
- **WHEN** a target's apex resolves but no `ip.asn` organisation or
  country is available
- **THEN** the hosting Finding reports the hosting operator as
  undetermined (no GeoIP) and does not fail

#### Scenario: Anycast apex with no resolvable country

- **GIVEN** an apex IP on an AS operator that carries no country (anycast)
- **WHEN** the scan synthesises hosting identity
- **THEN** the Finding names the operator with country reported as
  undetermined (anycast?), mirroring the apex rule's no-country handling

#### Scenario: A domain with no resolvable apex host does not fail

- **GIVEN** a target whose apex `A`/`AAAA` records cannot be resolved
- **WHEN** the scan synthesises hosting identity
- **THEN** the Finding states there is no resolvable apex host rather than
  erroring, and the scan completes
