# Delta for assessor

> Implemented 2026-06-15; ready to merge into the canonical assessor spec.

## ADDED Requirements

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
