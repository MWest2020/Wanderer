# Delta for web-ui

## ADDED Requirements

### Requirement: Reporting index lists rules across targets

The UI SHALL expose `/ui/reporting`, listing every rule that has
produced at least one Rationale across the persisted Assessments,
with one row per rule and per-score columns (`soeverein`,
`voldoende`, `afhankelijk`, `onbekend`) counting **distinct
target IDs** per score. A rule that fires multiple times on the
same target SHALL count once. Rule rows MUST be ordered
deterministically: framework first (`wand` > `eucsf` >
alphabetical), then rule ID alphabetical within each framework.

#### Scenario: Rule firing on two targets with different scores

- **Given** rule `wand.juridisch.cert_issuer_eea` scored
  `soeverein` on target A and `afhankelijk` on target B (each via
  the most recent Assessment)
- **When** the operator opens `/ui/reporting`
- **Then** the row for that rule shows soeverein=1, afhankelijk=1
- **And** other score columns (voldoende, onbekend) show 0

#### Scenario: Rule firing twice on same target counts once

- **Given** target A's most recent Assessment contains two
  Rationale entries for the same rule (multi-host MX dimension,
  for example), both `afhankelijk`
- **When** the operator opens `/ui/reporting`
- **Then** the rule's afhankelijk count is 1, not 2

---

### Requirement: Reporting rule detail page lists targets

The UI SHALL expose
`/ui/reporting/{framework}/{ruleID}`, showing the rule's
description (from the in-process rule registry) plus a row per
target the rule has fired on, ordered by score severity
(`afhankelijk` rows first, then `voldoende`, `soeverein`,
`onbekend`). Each row's target cell SHALL link to the originating
scan's assessment page (`/ui/scans/{scanID}/assessment`). When
a target has more than one Assessment for the rule's framework,
the **most recent** Assessment SHALL drive the row. Unknown
rules (neither registered framework knows the criteriumID) SHALL
return HTTP 404 with no body content beyond the 404 default.

#### Scenario: Detail page renders rule and targets

- **Given** rule `wand.juridisch.cert_issuer_eea` registered in
  the wand pack and fired on two targets
- **When** the operator opens
  `/ui/reporting/wand/wand.juridisch.cert_issuer_eea`
- **Then** the page returns 200
- **And** shows the rule's description text from the registry
- **And** lists both targets with their score and Verdict

#### Scenario: Unknown rule returns 404

- **Given** no registered rule matches
  `wand.juridisch.does_not_exist`
- **When** the operator opens
  `/ui/reporting/wand/wand.juridisch.does_not_exist`
- **Then** the response status is 404
