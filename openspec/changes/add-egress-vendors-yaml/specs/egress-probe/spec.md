# Delta for egress-probe

## ADDED Requirements

### Requirement: Vendor list is externally overridable

The egress probe SHALL load its vendor / region lookup table from
an external YAML file when one is supplied (via `--vendors` or
`WANDERER_VENDORS`), and SHALL fall back to a Go-embedded default
otherwise, so an operator can extend the classifier with
organisation-specific vendors without rebuilding the binary.

#### Scenario: Override with a custom vendor

- **Given** a YAML file containing one new `log_shippers` entry
  with `host_contains: vendor.example.nl`
- **When** the agent starts with `--vendors <file>`
- **And** the egress probe processes a config value of
  `https://vendor.example.nl/ingest`
- **Then** the resulting Finding has `ProbeID:
  egress.log_shipper`
- **And** Attributes contain `classifier_rule` matching the
  custom entry

#### Scenario: Default is the embedded file

- **Given** an agent without `--vendors` / `WANDERER_VENDORS`
- **When** the egress probe processes
  `s3.eu-west-1.amazonaws.com`
- **Then** the resulting Finding has `Attributes.classifier_rule:
  aws_s3_region_host`

#### Scenario: Malformed override fails fast

- **Given** an override YAML file with invalid syntax
- **When** the agent starts
- **Then** the process exits non-zero
- **And** stderr names the offending file and parse position
