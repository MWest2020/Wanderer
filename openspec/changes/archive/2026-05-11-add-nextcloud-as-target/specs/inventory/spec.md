# Delta for inventory

> Held active until Mark picks the integration direction.
> Specs below assume direction = (1) Nextcloud as a target;
> if a different direction wins, this delta is rewritten.

## ADDED Requirements

### Requirement: Nextcloud inspector emits version + trust signals

The Nextcloud inspector SHALL, when enabled, emit four
classes of Finding from a single `occ` session: the
installed Nextcloud version, the trusted-domain list, every
configured objectstore backend, and every configured OIDC
provider. Each Finding carries the ProbeID prefix
`inventory.nextcloud.` and the SourceModus `inventory`.

#### Scenario: Version probe runs first

- **Given** an agent host where Nextcloud is installed and
  `inspectors.nextcloud.enabled: true`
- **When** the agent ticks
- **Then** the inspector emits exactly one
  `inventory.nextcloud.version` Finding with `versionstring`
  in Attributes

#### Scenario: Objectstore probe annotates with geoip

- **Given** an agent host whose Nextcloud has an S3
  objectstore backend configured with a
  `s3.amazonaws.com` endpoint
- **When** the agent ticks with geoip configured
- **Then** the resulting `inventory.nextcloud.objectstore`
  Finding carries `endpoint` + `bucket` + the geoip
  enrichment (`asn`, `country: "US"`)

### Requirement: Three sovereignty rules score the Nextcloud surface

The assessor SHALL register three new rules covering the
Nextcloud-as-a-target dimension: `wand.nextcloud.objectstore_eu`,
`wand.nextcloud.oidc_provider_eu`, and
`eucsf.sov6.nextcloud_supply_chain`. Each follows the same
soeverein / voldoende / afhankelijk / onbekend shape the
host telemetry rules established.

#### Scenario: Objectstore on US hyperscaler scores afhankelijk

- **Given** an Assessment built from Findings that include
  one `inventory.nextcloud.objectstore` Finding annotated
  `country: "US"`, `organisation: "Amazon.com, Inc."`
- **When** the assessor runs `wand.nextcloud.objectstore_eu`
- **Then** the resulting Rationale has Score `afhankelijk`
  and the Verdict cites the offending endpoint + ASN
