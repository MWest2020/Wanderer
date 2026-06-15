# Delta for web-ui

> Implemented 2026-06-15; ready to merge into the canonical web-ui spec.

## ADDED Requirements

### Requirement: The assessment view synthesises a Sovereignty overview

The assessment page SHALL render a "Sovereignty overview" that
synthesises the scan's scored rule rationales into an ordered set of
flows — Hosting, Mail, DNS, Transit path, CDN / hyperscaler, Third
parties — each shown with the rule's observed verdict and its score.
The overview SHALL derive solely from the existing assessment data (no
new collection, no jurisdiction logic in the view). When no flow rule
fired, the overview SHALL be omitted rather than shown empty.

#### Scenario: A scored scan shows the synthesis panel

- **GIVEN** a scan whose wand assessment contains flow rules (apex
  hosting, mail, DNS, hyperscaler, third parties)
- **WHEN** an operator opens `/ui/scans/{id}/assessment`
- **THEN** a "Sovereignty overview" section lists each flow with its
  category, score pill, and observed verdict

#### Scenario: No flow rules → no empty panel

- **GIVEN** a scan whose assessment contains none of the flow rules
- **WHEN** the assessment page renders
- **THEN** the Sovereignty overview section is absent
