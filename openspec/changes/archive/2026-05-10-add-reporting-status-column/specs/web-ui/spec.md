# Delta for web-ui

## MODIFIED Requirements

### Requirement: Reporting layer is the rule catalogue

The UI SHALL render `/ui/reporting` as a rule catalogue:
**every** registered rule from every framework, with the
framework, dimension, rule ID, human-readable description, and
rationale text. The catalogue MAY carry a compact "current
state" column per row — the worst score reached across the
snapshots in scope plus a triage hint of how many targets sit
at that score — but MUST NOT render the full per-score matrix
(that stays on Analysis). Each row links to the existing
per-rule deep-dive page at
`/ui/reporting/{framework}/{ruleID}`.

#### Scenario: Catalogue lists rules with descriptions

- **Given** the wand pack registers
  `wand.juridisch.cert_issuer_eea` and the eucsf pack registers
  `eucsf.sov2.cert_issuer_eu`
- **When** the operator opens `/ui/reporting`
- **Then** both rules appear in the catalogue with their
  description text

#### Scenario: Catalogue carries a per-rule status hint

- **Given** rule `wand.juridisch.cert_issuer_eea` has fired
  across the scope with a mix of soeverein + afhankelijk
  rationales
- **When** the operator opens `/ui/reporting`
- **Then** the row for that rule shows a compact "afhankelijk"
  status pill (the worst score reached) with a "X of Y
  targets" hint
- **And** the row does NOT show the full soeverein / voldoende
  / afhankelijk / onbekend score breakdown (that lives on
  Analysis)

#### Scenario: Rules without rationale render an explicit placeholder

- **Given** rule `wand.juridisch.cert_issuer_eea` has not yet
  produced a rationale (no scans assessed under wand)
- **When** the operator opens `/ui/reporting`
- **Then** the row's status column reads "no rationale yet"
  rather than a fake score
