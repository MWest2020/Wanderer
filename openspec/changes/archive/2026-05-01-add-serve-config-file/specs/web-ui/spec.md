# Delta for web-ui

## ADDED Requirements

### Requirement: `wanderer serve` MAY load settings from a YAML config file

The `wanderer serve` command SHALL accept a `--config <path>`
flag (and `WANDERER_CONFIG` env var equivalent) that loads a
YAML file covering its operator-tunable settings: `listen`, `db`,
`geoip.{asn,country,optional}`, `ui.{enabled,htpasswd}`,
`schedules`, and `scan.{per_probe_timeout,budget,user_agent,
allow_private_targets}`. The YAML parse MUST be strict — unknown
fields fail the process at startup with an error naming the bad
field, never silently defaulted. When `--config` is unset, the
command SHALL behave byte-identically to the no-YAML form.

#### Scenario: Empty config equals no config

- **Given** a config file containing only `{}`
- **When** `wanderer serve --config empty.yaml` runs
- **Then** every setting resolves to its hard-coded default
- **And** the process behaves identically to `wanderer serve`
  with no `--config` flag

#### Scenario: YAML value applied when no flag or env

- **Given** a config file with `listen: ":9090"`
- **When** `wanderer serve --config x.yaml` runs with
  `WANDERER_LISTEN` unset and no `--addr` flag
- **Then** the HTTP server listens on `:9090`

#### Scenario: Unknown field rejected

- **Given** a config file containing `htpasswrd: /etc/htpasswd`
  (typo for `htpasswd`, under the wrong nesting)
- **When** `wanderer serve --config x.yaml` runs
- **Then** the process exits non-zero before opening any port
- **And** stderr contains an error naming the unknown field

---

### Requirement: Setting precedence is flag, env, YAML, default

`wanderer serve` SHALL resolve every setting by applying the
highest-precedence layer that is present, in the order: (1) CLI
flag explicitly passed on the command line; (2) environment
variable explicitly set in the process env; (3) YAML config
value; (4) hard-coded default. A flag explicitly set to its
default value (e.g. `--ui=false`) MUST still count as
"explicitly set" and override a YAML value.

#### Scenario: Flag overrides YAML

- **Given** a config file with `listen: ":9090"`
- **When** `wanderer serve --config x.yaml --addr :7070` runs
- **Then** the HTTP server listens on `:7070`

#### Scenario: Env var overrides YAML

- **Given** a config file with `db: /var/lib/wanderer/wanderer.db`
- **And** `WANDERER_DB=/tmp/test.db` set in the process env
- **When** `wanderer serve --config x.yaml` runs without `--db`
- **Then** the store is opened against `/tmp/test.db`

#### Scenario: Explicit flag-false overrides YAML true

- **Given** a config file with `ui.enabled: true`
- **When** `wanderer serve --config x.yaml --ui=false` runs
- **Then** the UI is not mounted at `/ui/`
