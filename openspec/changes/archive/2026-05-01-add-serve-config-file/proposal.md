# Proposal: YAML config file for `wanderer serve`

## Intent

`wanderer agent` already accepts a YAML config (`wanderer-agent.yaml`)
that holds every operator-tunable setting in one auditable file.
`wanderer serve` does not. The flag surface for serve has grown —
`--addr`, `--db`, `--geoip`, `--geoip-country`, `--no-geoip`,
`--ui`, `--ui-htpasswd`, `--schedules`, `--budget`,
`--per-probe-timeout`, `--user-agent`, `--allow-private-targets` —
and invoking serve from a systemd unit, a Compose file, or a
hand-rolled wrapper means a long argv that's hard to review and
easy to mistype.

Add a `--config /path/to/serve.yaml` flag (with `WANDERER_CONFIG`
env equivalent) that loads a YAML config covering the entire
flag surface. Precedence is explicit:

```
CLI flag (set on command line)
  > environment variable (set in process env)
    > YAML config value
      > hard-coded default
```

So an operator can lay down a serve.yaml as the durable source of
truth and still override one knob from a one-off invocation
without editing the file.

## Scope

**In scope:**
- New package `internal/serveconfig/` with `Config`, `Load`,
  `Parse`, and `Validate`. `Parse` uses strict YAML unmarshal so
  typos like `htpasswrd` fail fast at startup, never silently.
- A `--config` flag (and `WANDERER_CONFIG` env var) for
  `wanderer serve`. Empty path = no YAML loaded; behaviour is
  byte-identical to today.
- Helper-based precedence resolution that distinguishes:
  - Flag explicitly set on the command line (via `flag.Visit`)
  - Env var explicitly set (via `os.LookupEnv`)
  - YAML value present (non-zero / non-empty in the parsed struct)
  - Hard-coded default
- Tests covering each layer of the precedence stack.
- Docs: `docs/operator.md` snippet showing a minimal serve.yaml
  + an example systemd unit with `--config` instead of a wall of
  flags.

**Out of scope:**
- Migrating `wanderer scan` to YAML. Scan is one-shot; flags
  fit. If an operator wants to drive scan from a file, that is
  what `wanderer serve --schedules` already covers.
- Migrating `wanderer agent`. It already has its own YAML.
- A unified single-binary config (`wanderer.yaml` with
  top-level `serve:` / `agent:` / `scan:` sections). That's a
  bigger surface change with its own proposal if ever desired.
- Live config reload (SIGHUP). Restart-to-reload is the boring
  default and matches how schedulers and most daemons behave.
- Validation of the YAML file's *paths* (does the geoip mmdb
  exist? is htpasswd readable?) beyond the existing fail-fast
  the binary already does at startup.

## Why this fits the project posture

The flag surface grows with every probe and every UI page; a
config file cleanly contains that growth. Mark named the
asymmetry directly: "flags zijn cool voor cli, maar voor de app
denk weken met config files". The agent already proves the YAML
shape works.

## Wand dimensions informed

None directly. This is operator ergonomics — making the existing
behaviour easier to deploy and audit. No new findings, no new
rules, no scoring change.

## Passive / active boundary

Configuration only. No new outbound calls; no new privileges
required. The strict-YAML parser does not execute or interpret
anything beyond field assignment.

## Parallel-safe

New package `internal/serveconfig/`, a small refactor in
`cmd/wanderer/serve.go` to use the resolver helper, plus tests
and docs. No schema changes, no DB migration, no public-API
shift.
