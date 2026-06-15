# Delta for transit-probe

> New capability. Held until the Q1–Q3 scope calls in proposal.md.

## ADDED Requirements

### Requirement: Wanderer traces the network path to a target and attributes each hop

When the transit probe runs against a target, Wanderer SHALL trace the
network path to each resolved target IP and emit one Finding per
responding hop carrying the hop number, hop IP, reverse-DNS hostname
(when resolvable), ASN, ASN organisation, and country (when a GeoLite2
database is configured), plus the round-trip time. Hops that do not
reply SHALL be recorded as gaps rather than aborting the trace. The
probe SHALL also emit one aggregate Finding summarising the distinct
countries and ASNs the path crosses and naming any non-EU transit
providers.

#### Scenario: A trace reveals the hosting provider and transit jurisdictions

- **GIVEN** a target whose path egresses through an NL access network,
  a non-EU-headquartered transit carrier, and terminates at an NL
  hosting provider
- **WHEN** the transit probe runs against the target
- **THEN** the scan contains a per-hop Finding for each responding hop
  with its IP, reverse-DNS hostname, ASN, organisation, and country,
- **AND** an aggregate Finding names the non-EU transit carrier and the
  destination hosting provider

#### Scenario: Unresponsive hops do not fail the trace

- **GIVEN** a path where several intermediate hops do not answer
- **WHEN** the transit probe runs
- **THEN** those hops are recorded as gaps and the trace continues to
  the destination, and the probe completes without error

#### Scenario: No GeoIP database degrades gracefully

- **GIVEN** an instance with no GeoLite2 database configured
- **WHEN** the transit probe runs
- **THEN** each hop Finding still carries the hop IP and reverse-DNS
  hostname, with ASN/country omitted, and the probe does not fail
