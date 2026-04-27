# Delta for egress-probe

## ADDED Requirements

### Requirement: Secrets are never written to Findings

The egress probe SHALL replace any value identified as a secret (by
key name pattern or by content pattern) with the literal string
`«redacted»` before storing it in a Finding's attributes, evidence,
or logging it via slog.

#### Scenario: API key in an env var

- **Given** a running process with `AWS_SECRET_ACCESS_KEY=abcDEF123...`
- **When** the egress probe reads `/proc/<pid>/environ`
- **Then** any resulting Finding's `Attributes` and `Evidence`
  contain `«redacted»` in place of the key value
- **And** no slog line anywhere in the agent's output contains the
  raw value

#### Scenario: DB URL with inline password

- **Given** a config file containing
  `DATABASE_URL=postgres://app:hunter2@db.example:5432/app`
- **When** the egress probe processes it
- **Then** the emitted `egress.database` Finding's `Attributes`
  contain `url: "postgres://app:«redacted»@db.example:5432/app"`
- **And** `Evidence` contains the same redacted string

#### Scenario: Short non-secret values are preserved

- **Given** a value `DB_USER=app` (key does not match a secret
  pattern, value is not long/random)
- **When** the egress probe processes it
- **Then** the value is passed through unchanged into `Attributes`

---

### Requirement: Classification is explainable

Every non-`unknown` egress Finding SHALL carry a `classifier_rule`
attribute identifying which pattern caused the classification, so an
operator can trace why a host was flagged as, for example, an S3
endpoint.

#### Scenario: AWS S3 classification

- **Given** a config value `s3.eu-west-1.amazonaws.com`
- **When** classification runs
- **Then** the resulting Finding has
  `classifier_rule: "aws_s3_region_host"` in Attributes
- **And** the `provider` attribute is `aws`
- **And** the `region` attribute is `eu-west-1`

#### Scenario: Unknown host

- **Given** a URL pointing at `exports.random-company.example`
- **When** classification runs and no pattern matches with confidence ≥ `low`
- **Then** the Finding's `ProbeID` is `egress.unknown`
- **And** it carries `confidence: "none"` in Attributes

---

### Requirement: Host resolution reuses the IP probe

When the IP probe is available, every egress Finding SHALL be
annotated with `asn`, `organisation`, and `country` attributes for
its subject host.

#### Scenario: ASN/country annotation

- **Given** an agent running both the IP probe (with GeoLite2 DB)
  and the egress probe
- **When** egress finds `s3.eu-west-1.amazonaws.com`
- **Then** the Finding carries `country: "IE"` (or the correct ISO
  code for Amazon's eu-west-1 POP)
- **And** `organisation` contains "Amazon"

#### Scenario: IP probe unavailable

- **Given** an agent with no GeoLite2 DB configured
- **When** egress findings are emitted
- **Then** each Finding contains the subject host but no `asn`,
  `organisation`, or `country` attributes
- **And** a sibling Finding with `ProbeID: egress.host_resolution.unavailable`
  is emitted exactly once per run, not per subject host

---

### Requirement: Config-path configurability

The configfiles scanner SHALL read only the paths explicitly
enumerated in the agent configuration. It SHALL NOT walk the entire
filesystem.

#### Scenario: Default is empty

- **Given** a `wanderer-agent.yaml` without an `egress.configfiles.paths`
  entry
- **When** the egress probe runs
- **Then** the configfiles scanner emits no Findings
- **And** emits one `egress.configfiles.unconfigured` info Finding

#### Scenario: Explicit paths

- **Given** `egress.configfiles.paths: ["/etc/wanderer-sample/"]`
- **When** the scanner runs
- **Then** only files under that path tree are read
- **And** symlinks pointing outside the path tree are not followed
