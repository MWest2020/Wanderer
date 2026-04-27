# Design: Inventory Probe

## Component placement

```
cmd/wanderer/agent.go            # CLI: wanderer agent [--config ...]
internal/probe/inventory/
  inventory.go                   # Probe implementation (runs all enabled inspectors)
  systemd/                       # systemd inspector
  docker/                        # Docker inspector
  packages/                      # dpkg + rpm inspector
  nextcloud/                     # Nextcloud occ inspector
internal/agent/
  config.go                      # wanderer-agent.yaml parsing
  remote.go                      # POST /findings to remote core
internal/api/findings.go         # new endpoint + HMAC verification
docs/agent.md                    # operator guide for the agent
```

## Inspector interface

```go
type Inspector interface {
    // ID returns a stable short identifier, e.g. "systemd", "docker".
    // Used as the probe-prefix segment in ProbeIDs
    // (e.g. "inventory.systemd.service").
    ID() string

    // Available reports whether this inspector can run on the
    // current host (socket present, binary in PATH, etc.).
    // Unavailable inspectors emit a single "inventory.<id>.unavailable"
    // Finding and are skipped.
    Available() bool

    // Inspect runs the inspection.
    Inspect(ctx context.Context) ([]models.Finding, error)
}
```

Each inspector is a package under `internal/probe/inventory/<id>/`.
No plugin loading; `inventory.go` imports each package explicitly
(same pattern as `scanner` importing probes).

## Finding ProbeIDs

| ProbeID                          | Subject           | Dimension      | Attributes                                                              |
| -------------------------------- | ----------------- | -------------- | ----------------------------------------------------------------------- |
| `inventory.systemd.service`      | service unit name | `operationeel` | `load_state`, `active_state`, `sub_state`, `description`, `fragment_path` |
| `inventory.docker.container`     | container name    | `technologie`  | `image`, `image_digest`, `state`, `created_at`, `labels`                |
| `inventory.docker.image`         | image ref         | `technologie`  | `digest`, `repo_tags`, `size_bytes`, `created_at`                       |
| `inventory.packages.dpkg`        | package name      | `operationeel` | `version`, `arch`, `status`                                             |
| `inventory.packages.rpm`         | package name      | `operationeel` | `version`, `arch`                                                       |
| `inventory.nextcloud.app`        | app id            | `technologie`  | `version`, `enabled`, `author`, `type`                                  |
| `inventory.<id>.unavailable`     | hostname          | —              | `reason` — inspector could not run (missing binary, permission denied)  |

Severities:

- Default `info` for observation.
- `observation` when an attribute hints at a known EOL (e.g. PHP
  < 8.1, Ubuntu < 20.04). EOL detection is a small lookup table
  shipped with the packages inspector.
- `concern` when a running service is from a known non-EU SaaS vendor
  (e.g. Datadog agent, New Relic). Lookup table lives next to the
  inspector.

## Agent config (`wanderer-agent.yaml`)

```yaml
hostname: webapp-01.example.internal
core:
  mode: local          # local | remote
  db: /var/lib/wanderer/wanderer.db    # for mode=local
  # -- or --
  url: https://wanderer.example.internal
  hmac_secret_file: /etc/wanderer/agent.hmac
  target_id: t_abc     # the Wanderer target this host belongs to

scan:
  interval: 1h         # 0 = run once and exit
  timeout: 5m

inspectors:
  systemd:
    enabled: true
  docker:
    enabled: true
    socket: /var/run/docker.sock
  packages:
    enabled: true
    managers: [dpkg, rpm]   # which to try; reports unavailable cleanly
  nextcloud:
    enabled: false
    occ_path: /var/www/nextcloud/occ
    run_as: www-data
```

Env-var overrides follow the Wanderer convention (uppercased, dot →
underscore). No secrets in the config itself — HMAC secret is a file
path, not an inline value.

## Remote transport (HMAC-over-HTTPS)

Agent → Core:

```
POST /scans/{id}/findings
Content-Type: application/json
X-Wanderer-Agent: webapp-01.example.internal
X-Wanderer-Timestamp: 2026-04-23T19:50:11Z
X-Wanderer-Signature: base64(HMAC-SHA256(secret, timestamp + "\n" + body))
```

Core verifies:

1. Timestamp within ±5 minutes of now (replay protection).
2. Signature matches for the shared secret of the reporting host.
3. Hostname is registered — unknown hosts return 401.

## External systems + failure modes

| System              | Inspector        | Failure mode                                                       |
| ------------------- | ---------------- | ------------------------------------------------------------------ |
| D-Bus / systemd     | systemd          | No PID 1 systemd (e.g. a container) → `unavailable`, no error     |
| `/var/run/docker.sock` | docker        | Not present / no perm → `unavailable`                              |
| `dpkg-query`        | packages/dpkg    | Not in PATH → skipped                                              |
| `rpm`               | packages/rpm     | Not in PATH → skipped                                              |
| `occ` CLI           | nextcloud        | Not in path or fails → `unavailable` with the CLI stderr as reason |
| Wanderer core (remote) | remote.go     | Network error → retry with exponential backoff up to 5×, then write to a local "outbox" spool file and retry on next run |

## Clever valkuil

Tempting: scrape every file on disk, compute hashes, generate full
SPDX SBOM on every run. That is a bad fit for the MVP of this change:
it consumes CPU, it produces findings at a scale the assessor does
not yet know how to use, and the DICTU toets does not ask for a file
hash. Package-level inventory (dpkg/rpm) is the right granularity:
names and versions, which is what a supply-chain question is actually
about.

Equally tempting: hot-reload the agent config on SIGHUP. Skip it. A
restart is 100 ms, systemd handles it, operators understand restarts.
Hot-reload is a future source of "why is production still using the
old config" bugs.

Third temptation: eBPF for process/network observation. The value is
huge but the operational surface is heavy (kernel version matrices,
root privileges, symbol resolution). Egress observation via eBPF is
the natural landing place — if we decide to, it is a follow-up change
called `add-egress-flow-probe` separate from `add-egress-probe`.

## Relation to other changes

- **`add-assessor`** — inventory Findings with `operationeel` and
  `technologie` DimensionHints flow into assessor rules. The "modus"
  column on Findings lets the assessor's completeness calculation
  know inventory data is present.
- **`add-egress-probe`** — runs inside the same agent process
  (`wanderer agent`) alongside the inventory inspectors. Shares the
  `wanderer-agent.yaml` config file.
- **`add-scheduling`** — when both agent and core schedule scans, the
  agent uses its local `interval` config; the core's scheduler
  covers remote perimeter scans.

## Coverage

- `internal/probe/inventory/*` target: 70%.
- `internal/agent/remote.go` target: 85% (security-sensitive).
- HMAC verification has golden-vector tests against a known secret.

## Security review note

A later `security-review` skill pass is required on this change before
merge — HMAC handling, timestamp skew, replay protection, and the
agent's privilege surface are all attack-relevant.
