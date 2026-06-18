# Delta for dns-hosting

> New capability — graduates from `research-high-signal-observability`
> (Wave 1). Leads with the observed DNS-hosting fact; the existing
> `wand.juridisch.ns_vendor_jurisdiction` rule annotates it.
> DICTU dimensions: Juridisch (operator jurisdiction) and Operationeel
> (authoritative DNS as control plane). Structural twin of email-routing.

## ADDED Requirements

### Requirement: Wanderer states who runs a target's authoritative DNS and where

Wanderer SHALL, after resolving a target's authoritative `NS` records and
attributing each NS host to an ASN/country, emit one aggregate observed
Finding that names, for each NS host, the recognisable managed-DNS
**operator** (derived from a known-NS-host table with an
ASN-organisation fallback) and the **hosting country** (when a GeoLite2
database is configured). The Finding SHALL read as a plain statement —
"DNS for {domain} is run by {operator} ({country})" — and SHALL retain
the raw NS host and ASN organisation as evidence so the observed fact
stands even when the friendly operator name is uncertain. This Finding is
an observation, not a verdict; the EEA-jurisdiction score is carried
separately by `wand.juridisch.ns_vendor_jurisdiction`.

#### Scenario: DNS run by a non-EEA operator reads as a plain statement

- **GIVEN** a target whose `NS` records resolve to hosts under
  `cloudflare.com`, hosted on a US-registered AS
- **WHEN** the scan correlates `dns.ns` with `ip.asn`
- **THEN** the scan contains an aggregate `dns.ns_hosting` Finding naming
  the operator "Cloudflare" and country "US",
- **AND** the raw NS host and ASN organisation are retained as evidence,
- **AND** `wand.juridisch.ns_vendor_jurisdiction` still scores the
  authoritative DNS as outside the EEA

#### Scenario: No GeoIP database degrades to operator-only

- **GIVEN** an instance with no GeoLite2 database configured
- **WHEN** a target's NS records resolve but no `ip.asn` country is
  available
- **THEN** the DNS-hosting Finding still names the operator (from the
  NS-host suffix table or rDNS) with the country omitted, and does not
  fail

#### Scenario: Anycast nameservers with no resolvable country

- **GIVEN** NS hosts on an AS operator that carries no country (anycast —
  the common case for large DNS providers)
- **WHEN** the scan synthesises DNS hosting
- **THEN** the Finding names the operator with country reported as
  undetermined (anycast?), mirroring the NS rule's no-country handling

#### Scenario: A domain with no resolvable authoritative DNS does not fail

- **GIVEN** a target whose authoritative `NS` records cannot be resolved
- **WHEN** the scan synthesises DNS hosting
- **THEN** the Finding states there is no resolvable authoritative DNS
  rather than erroring, and the scan completes
