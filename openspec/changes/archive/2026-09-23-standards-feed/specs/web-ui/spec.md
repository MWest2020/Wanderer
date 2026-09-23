## ADDED Requirements

### Requirement: An import scan never stands in for a domain's latest scan

Every view that shows a domain's "latest scan" — the fleet screen, the
dashboard and its roll-ups, the trends layer, the demo page — SHALL choose the
newest scan that is not an import scan. An import scan is one written by
`wanderer import internetnl` or `POST /imports/internetnl`: it carries only
imported Findings and has no assessment of its own. Its Findings reach the
assessor through the perimeter scan (`FindingsForAssessment`), so skipping it
loses nothing. Showing it instead replaces a domain's scored posture with an
empty one.

An import scan SHALL remain reachable on its own page (`/ui/scans/{id}`).

#### Scenario: Import after a perimeter scan

- **GIVEN** a domain with a perimeter scan scored 5/7, and afterwards an
  import scan for the same domain
- **WHEN** the fleet screen renders
- **THEN** the domain's row shows 5/7 and the perimeter scan's time, not 0/0
- **AND** its delta compares that perimeter scan with the perimeter scan
  before it

#### Scenario: Only imports, no perimeter scan yet

- **GIVEN** a fleet domain whose only scans are import scans
- **WHEN** the fleet screen renders
- **THEN** the domain is listed as not yet scanned
