## ADDED Requirements

### Requirement: Operator documentation explains GeoLite2 setup

`docs/operator.md` SHALL include a "GeoLite2 setup" section
covering MaxMind license-key acquisition, the recommended
periodic-update mechanism (`geoipupdate` via systemd timer or
crontab), the file paths the agent expects, and how to silence
the startup warning when GeoLite2 is intentionally absent.
`docs/architecture.md` and `docs/tutorial.md` SHALL link into
the section.

#### Scenario: Operator opens the docs

- **GIVEN** a contributor reading `docs/operator.md` for the
  first time
- **WHEN** they search for "GeoLite2"
- **THEN** they find a top-level section explaining the setup
- **AND** the section names both `--geoip` / `WANDERER_GEOIP_ASN`
  (required) and `--geoip-country` / `WANDERER_GEOIP_COUNTRY`
  (optional)
- **AND** the section names `WANDERER_GEOIP_OPTIONAL=1` /
  `--no-geoip` as the opt-out for hosts that consciously run
  without ASN annotation

#### Scenario: Architecture doc cross-references the setup

- **GIVEN** the rewritten `docs/architecture.md`
- **WHEN** a reader scrolls to the "External systems and their
  failure modes" table
- **THEN** the GeoLite2 row links to the operator doc's
  "GeoLite2 setup" section

---

### Requirement: CLI warns once at startup when GeoLite2 is missing

`wanderer scan` and `wanderer serve` SHALL emit one
warning-level log line to stderr at startup when no
`--geoip` value is set and the `--no-geoip` /
`WANDERER_GEOIP_OPTIONAL=1` opt-out is not present. The warning
SHALL name the `--geoip` flag, name the operator-doc URL or
path, and state explicitly that the scan continues with reduced
assessment coverage rather than failing.

#### Scenario: Default invocation surfaces the warning

- **GIVEN** an operator runs `wanderer scan example.nl` on a
  host with no `--geoip`, no `WANDERER_GEOIP_ASN`, and no
  `WANDERER_GEOIP_OPTIONAL`
- **WHEN** the binary starts
- **THEN** stderr contains exactly one warning line beginning
  with `warning:`
- **AND** the line names `--geoip` and points at `docs/operator.md`
- **AND** the scan still completes (exit code 0 on success)

#### Scenario: Opt-out silences the warning

- **GIVEN** the same operator with `--no-geoip` set on the
  command line (or `WANDERER_GEOIP_OPTIONAL=1` in the env)
- **WHEN** the binary starts
- **THEN** stderr contains no warning about GeoLite2
- **AND** the scan still completes
- **AND** the per-scan `ip.unavailable` Finding is unchanged

#### Scenario: Configured GeoLite2 produces no warning

- **GIVEN** an operator with `--geoip /var/lib/wanderer/GeoLite2-ASN.mmdb` set and the file is readable
- **WHEN** the binary starts
- **THEN** stderr contains no GeoLite2 warning
- **AND** the IP probe runs in its populated mode

---

### Requirement: Test suite has a stub mmdb path

The test suite SHALL ship a way to exercise the `internal/probe/ip`
populated-but-empty path without a real MaxMind license. The
mechanism SHALL be either a `scripts/geoip-stub.sh` that produces
a deterministic minimal mmdb file, or an equivalent Go test
helper, with a single command/function call documented in
`docs/operator.md`.

#### Scenario: Stub builder produces a valid empty mmdb

- **GIVEN** `scripts/geoip-stub.sh` (or the equivalent test
  helper)
- **WHEN** a test invokes it pointing at a temp directory
- **THEN** the resulting file opens cleanly via the
  `oschwald/maxminddb-golang` reader
- **AND** any IP lookup against it returns "not found" without
  error
