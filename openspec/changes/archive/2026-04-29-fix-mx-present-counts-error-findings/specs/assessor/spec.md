# Delta for assessor

## ADDED Requirements

### Requirement: Rules ignore meta Findings

Every assessor rule SHALL skip Findings whose attributes mark them
as meta (an `error` attribute is present, `no_answer` is true, or
`unavailable` is true) when deciding whether evidence backs a
verdict, so a non-resolvable domain or a missing probe never
produces a positive score.

#### Scenario: NXDOMAIN does not score voldoende on MX presence

- **Given** a Finding set whose only `dns.mx` rows are
  `lookupError` records (each with an `error` attribute set)
- **When** the assessor runs the `dictu.data_ai.mx_present` rule
- **Then** the rule's RuleResult has `Score: onbekend`
- **And** the rule's `Evidence` list is empty

#### Scenario: A `no_answer` row is not evidence

- **Given** a Finding with `ProbeID: dns.caa` and
  `Attributes.no_answer: true`
- **When** the `dictu.operationeel.caa_restricts_issuance` rule
  evaluates the set
- **Then** the rule does not treat the row as a positive CAA
  observation
- **And** the rule's verdict reflects the absence of CAA records,
  not their presence
