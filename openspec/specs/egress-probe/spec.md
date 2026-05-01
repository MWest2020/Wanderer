# egress-probe Specification

## Purpose
TBD - created by archiving change add-egress-probe. Update Purpose after archive.
## Requirements
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

---

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

---

### Requirement: Flow probe captures runtime egress

The egress flow probe SHALL record outbound `connect()`
destinations during its configured sampling window when enabled
and supported by the kernel, and SHALL emit one
`egress.flow.<category>` Finding per unique
`(destination_ip, destination_port)` pair, reusing the existing
classifier and redactor so the wire format stays consistent with
the static egress probe.

#### Scenario: Unique destination produces a Finding

- **Given** a flow probe with a 30-second window and a process
  that calls `connect()` to `203.0.113.5:443` once
- **When** the window closes
- **Then** one `egress.flow.<category>` Finding is produced
- **And** Attributes contain `destination_ip`, `destination_port`,
  `runtime: true`, and `classifier_rule`

#### Scenario: Privilege missing surfaces gracefully

- **Given** a host without `CAP_BPF` or `CAP_PERFMON`
- **When** the flow probe is enabled and the agent starts
- **Then** an `egress.flow.unavailable` Finding is emitted exactly
  once
- **And** the agent process exits 0 (other inspectors continue)

---

### Requirement: Flow probe is opt-in

The flow probe SHALL be disabled by default and SHALL only run
when `egress.flow.enabled: true` is set in the agent config, so
operators must consciously accept the kernel-level capability cost
before any kernel attach happens.

#### Scenario: Default config does not load the program

- **Given** an agent with no `egress.flow` block in its config
- **When** the agent starts
- **Then** no eBPF program is loaded
- **And** no `egress.flow.*` Finding (including unavailable) is
  emitted

### Requirement: Flow probe MAY annotate Findings with reverse DNS

The egress flow probe SHALL support optional reverse DNS (PTR)
annotation of its Findings. When enabled, each unique destination
IP observed in the sampling window is resolved to a hostname via
the host's resolver, and the Finding's `Attributes` gain a
`reverse_dns` key carrying that hostname. Resolution failure
(NXDOMAIN, timeout, refused, transport error) SHALL leave the
Finding unchanged: no `reverse_dns` key, no error Finding. Reverse
DNS is enrichment, not a probe in its own right.

#### Scenario: Successful PTR annotates the Finding

- **Given** the flow probe with reverse DNS enabled, observing
  one connect to `203.0.113.5:443`, and a resolver that returns
  `ec2-203-0-113-5.eu-west-1.compute.amazonaws.com.`
- **When** the window closes
- **Then** the resulting `egress.flow.<category>` Finding's
  Attributes contain
  `reverse_dns: "ec2-203-0-113-5.eu-west-1.compute.amazonaws.com"`
  (trailing dot stripped)

#### Scenario: Failed PTR is silent

- **Given** the flow probe with reverse DNS enabled and a resolver
  that returns NXDOMAIN for every IP in the sampling window
- **When** the window closes
- **Then** every Finding's Attributes omit the `reverse_dns` key
- **And** no `egress.flow.reverse_dns.error` Finding is emitted

#### Scenario: Same IP across ports queries once

- **Given** the flow probe observes connects to `203.0.113.5:443`
  and `203.0.113.5:8443` within one window
- **When** the window closes
- **Then** the reverse DNS resolver is called exactly once for
  `203.0.113.5`
- **And** both Findings carry the same `reverse_dns` annotation

---

### Requirement: Reverse DNS is opt-in

The flow probe's reverse DNS annotation SHALL be disabled by
default and SHALL only run when
`egress.flow.reverse_dns.enabled: true` is set in the agent
config. PTR queries leak the observation back through the host's
DNS path — the resolver and every cache between learn that this
host saw the destination IP — which a sovereignty monitor cannot
default-on without consent.

#### Scenario: Default config makes no PTR queries

- **Given** an agent with the flow probe enabled but no
  `egress.flow.reverse_dns` block in its config
- **When** the agent starts and runs one tick
- **Then** the reverse DNS resolver is never constructed
- **And** no Finding's Attributes contain a `reverse_dns` key

#### Scenario: Per-lookup timeout caps blocking

- **Given** the flow probe with
  `egress.flow.reverse_dns: { enabled: true, timeout: 500ms }` and
  a resolver that hangs past the deadline
- **When** the window closes
- **Then** affected Findings have no `reverse_dns` key
- **And** total resolution time per IP does not exceed the
  configured timeout

