## ADDED Requirements

### Requirement: Every assessor Rule carries a plain-language Rationale

The `assessor.Rule` struct SHALL include a `Rationale string`
field that holds a one-paragraph plain-language explanation of
what the rule observes and why it matters for sovereignty
posture, populated alongside the existing `Description`. Every
Rule registered by `dictu.DefaultRules()` and
`eucsf.DefaultRules()` SHALL have a non-empty Rationale; an empty
string is a build-breaking error in the corresponding registry's
test suite.

#### Scenario: Rationale present on every default rule

- **GIVEN** the rule sets returned by
  `internal/assessor/dictu.DefaultRules()` and
  `internal/assessor/eucsf.DefaultRules()`
- **WHEN** a test iterates every Rule and reads `Rationale`
- **THEN** every Rule's `Rationale` is a non-empty string

#### Scenario: Empty Rationale fails CI

- **GIVEN** a contributor adds a new Rule with an empty
  `Rationale` field to either rule pack
- **WHEN** `go test ./internal/assessor/...` runs
- **THEN** the rule pack's `TestEveryRuleHasRationale` test fails
  with a message naming the offending `CriteriumID`

#### Scenario: Rationale is independent of Description

- **GIVEN** a Rule whose `Description` is a single-sentence
  summary ("TLS certificate issued by an authority in the EEA.")
- **WHEN** the renderer reads the Rule
- **THEN** `Rationale` is a separate string carrying the
  consequence of the rule firing
- **AND** `Description` and `Rationale` are not the same value
