# Delta for email-routing

> New capability — graduates from `research-high-signal-observability`
> (Wave 1). Leads with the observed mail-routing fact; the existing
> `wand.juridisch.mx_vendor_jurisdiction` rule annotates it.
> DICTU dimensions: Juridisch (operator jurisdiction) and Data & AI
> (inbound correspondence).

## ADDED Requirements

### Requirement: Wanderer states where a target's inbound mail lands and who runs it

Wanderer SHALL, after resolving a target's `MX` records and attributing each MX host to an ASN/country, emit one aggregate observed Finding that names, for each MX host in preference order, the recognisable mail **operator** (derived from a known-MX-host table with an ASN-organisation fallback) and the **hosting country** (when a GeoLite2 database is configured). The Finding SHALL read as a plain statement — "inbound mail
for {domain} lands at {operator} ({country})" — and SHALL retain the raw
MX host and ASN organisation as evidence so the observed fact stands even
when the friendly operator name is uncertain. This Finding is an
observation, not a verdict; the EEA-jurisdiction score is carried
separately by `wand.juridisch.mx_vendor_jurisdiction`.

#### Scenario: Mail routed to a non-EEA operator reads as a plain statement

- **GIVEN** a target whose `MX` records resolve to hosts under
  `aspmx.l.google.com`, hosted on a US-registered AS
- **WHEN** the scan correlates `dns.mx` with `ip.asn`
- **THEN** the scan contains an aggregate `dns.mx_routing` Finding
  naming the operator "Google" and country "US",
- **AND** the raw MX host and ASN organisation are retained as evidence,
- **AND** `wand.juridisch.mx_vendor_jurisdiction` still scores the
  routing as outside the EEA

#### Scenario: No GeoIP database degrades to operator-only

- **GIVEN** an instance with no GeoLite2 database configured
- **WHEN** a target's MX records resolve but no `ip.asn` country is
  available
- **THEN** the mail-routing Finding still names the operator (from the
  MX-host suffix table or rDNS) with the country omitted, and does not
  fail

#### Scenario: A domain with no inbound mail does not fail

- **GIVEN** a target with no `MX` record (or a single null MX `.`)
- **WHEN** the scan synthesises mail routing
- **THEN** the Finding states there is no inbound mail routing rather
  than erroring, and the scan completes

#### Scenario: Anycast MX with no resolvable country

- **GIVEN** an MX host on an AS operator that carries no country
  (anycast)
- **WHEN** the scan synthesises mail routing
- **THEN** the Finding names the operator with country reported as
  undetermined (anycast?), mirroring the MX rule's no-country handling
