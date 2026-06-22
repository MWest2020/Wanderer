# third-party-origin Specification

## Purpose
TBD - created by archiving change propose-third-party-origin. Update Purpose after archive.
## Requirements
### Requirement: Wanderer names the third-party vendors a page loads and where they sit

Wanderer SHALL, after recording a target page's third-party hosts (with
the kinds of resource each serves) and attributing each host to an
ASN/country, emit one aggregate observed Finding that groups the hosts by
recognisable **vendor** (derived from a known-host-suffix table with an
ASN-organisation fallback), naming for each vendor **what it serves**
(the union of resource kinds) and its **country** (when a GeoLite2
database is configured). The Finding SHALL read as a plain origin map and
SHALL call out the non-EEA vendors as the export surface. It SHALL retain
each raw host and ASN organisation as evidence so the observed map stands
even when a friendly vendor name is uncertain. This Finding is an
observation, not a verdict; the in/out-EEA count is carried separately by
`wand.technologie.third_parties_eea`.

#### Scenario: A page loading a non-EEA vendor reads as a named map

- **GIVEN** a target page that loads `fonts.googleapis.com` (font) and
  `fonts.gstatic.com` (font) hosted on a US-registered AS
- **WHEN** the scan correlates `http.third_party` with `ip.asn`
- **THEN** the scan contains an aggregate `http.origin_map` Finding
  grouping both hosts under the vendor "Google Fonts", naming the served
  kind "font" and country "US",
- **AND** the raw hosts and ASN organisation are retained as evidence,
- **AND** `wand.technologie.third_parties_eea` still reports the in/out-EEA
  host count

#### Scenario: An all-EEA page reads cleanly with no scary lead

- **GIVEN** a target page whose every third-party host resolves inside the
  EEA
- **WHEN** the scan synthesises the origin map
- **THEN** the map names the EEA vendors with no non-EEA export surface
  called out, and the rule verdict keeps its "all N hosts in the EEA"
  reading

#### Scenario: No GeoIP database degrades to vendor-only

- **GIVEN** an instance with no GeoLite2 database configured
- **WHEN** a target's third-party hosts are recorded but no `ip.asn`
  country is available
- **THEN** the origin map still names the vendors (from the host suffix
  table) with the country omitted, and does not fail

#### Scenario: Unrecognised host falls back to its ASN organisation

- **GIVEN** a third-party host absent from the vendor suffix table
- **WHEN** the scan synthesises the origin map
- **THEN** the vendor is named from the host's ASN organisation (or the
  raw host when neither is known), so the observed entry still stands

#### Scenario: A page with no third parties does not fail

- **GIVEN** a target whose page loads no external hosts (or whose HTTP
  probe did not run)
- **WHEN** the scan synthesises the origin map
- **THEN** the Finding states the page loads no third parties rather than
  erroring, and the scan completes

