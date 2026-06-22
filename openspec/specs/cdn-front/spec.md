# cdn-front Specification

## Purpose
TBD - created by archiving change propose-cdn-front. Update Purpose after archive.
## Requirements
### Requirement: Wanderer states when a target's apex is fronted by a CDN edge and whose

Wanderer SHALL, after attributing a target's apex IP to an ASN/country and
reading the apex `http.response` `server` header, emit one aggregate
observed Finding that states whether the apex is fronted by a recognisable
CDN/edge operator (detected from a curated table keyed on the ASN
organisation and the server header) and, when it is, names the **edge
operator** and the **hosting country** of the edge IP. The Finding SHALL
read as a plain statement — "{domain}'s apex is fronted by {edge}
({country})" — and SHALL record which signal(s) fired (ASN organisation,
server header) along with the raw values as evidence, so the observed fact
stands even when the friendly edge name is uncertain. This Finding is an
observation, not a verdict; the US-hyperscaler-reach score is carried
separately by `wand.technologie.no_us_hyperscaler`.

#### Scenario: A Cloudflare-fronted apex reads as a front, not a host

- **GIVEN** a target whose apex IP resolves to ASN organisation
  "CLOUDFLARENET" and whose apex `http.response` server header is
  "cloudflare"
- **WHEN** the scan correlates `ip.asn`, `http.response`, and `tls.issuer`
- **THEN** the scan contains an aggregate `http.cdn_front` Finding naming
  the edge "Cloudflare" and its country,
- **AND** the fired signals (ASN organisation and server header) are
  recorded as evidence,
- **AND** `wand.technologie.no_us_hyperscaler` still scores the apex as
  inside US-hyperscaler reach

#### Scenario: A directly-served apex reads as no front

- **GIVEN** a target whose apex IP organisation and server header match no
  CDN signature
- **WHEN** the scan synthesises CDN/front detection
- **THEN** the Finding states no CDN/edge front was detected and the apex
  is served directly, and does not fail

#### Scenario: Server-header-only detection still names the edge

- **GIVEN** a target with no GeoIP database (no apex `ip.asn`) whose apex
  server header is "Vercel"
- **WHEN** the scan synthesises CDN/front detection
- **THEN** the Finding names the edge from the server header with the
  country omitted, and records the server header as the fired signal

#### Scenario: Anycast edge with no resolvable country

- **GIVEN** an apex fronted by an edge whose anycast IP carries no country
- **WHEN** the scan synthesises CDN/front detection
- **THEN** the Finding names the edge with country reported as
  undetermined (anycast?), mirroring the existing no-country handling

#### Scenario: No apex evidence does not fail

- **GIVEN** a target whose apex produced neither an `ip.asn` nor an
  `http.response` finding (the IP and HTTP probes did not run)
- **WHEN** the scan synthesises CDN/front detection
- **THEN** no `http.cdn_front` Finding is emitted and the scan completes

