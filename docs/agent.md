# Wanderer agent

`wanderer agent` is the host-side mode of the Wanderer binary. It
runs *on* a host and reports what is installed and running back to a
Wanderer core. The same `Finding` shape carries inventory results
the assessor and reporting pipeline already understand — it is not
a parallel data path.

## What it observes (MVP)

| Inspector  | Source                                         | ProbeID prefix              |
| ---------- | ---------------------------------------------- | --------------------------- |
| `systemd`  | `systemctl list-units --type=service --output=json` | `inventory.systemd.service` |
| `packages` | `dpkg-query -W` and/or `rpm -qa`              | `inventory.packages.dpkg` / `inventory.packages.rpm` |
| `docker`   | placeholder — emits `inventory.docker.unavailable` until the full Docker socket integration ships in a follow-up | — |
| `nextcloud`| `occ app:list --output=json` (opt-in)          | `inventory.nextcloud.app`   |

Inspectors that cannot run on the current host (binary not in PATH,
socket missing, permission denied) emit a single
`inventory.<id>.unavailable` Finding with a human-readable `reason`
and the agent continues with the remaining inspectors.

The agent is **read-only**. It never writes outside its log, config,
and local spool directory; it never invokes mutating subcommands of
the tools it observes.

## Configuration

```yaml
# /etc/wanderer/agent.yaml
hostname: webapp-01.example.internal

core:
  mode: local                              # or remote
  db: /var/lib/wanderer/wanderer.db        # mode=local
  # url: https://wanderer.example.internal # mode=remote
  # hmac_secret_file: /etc/wanderer/agent.hmac
  # target_id: t_abc

scan:
  interval: 1h         # 0 = run once and exit
  timeout: 5m

inspectors:
  systemd:
    enabled: true
  packages:
    enabled: true
    managers: [dpkg, rpm]
  docker:
    enabled: true
    socket: /var/run/docker.sock  # the inspector lists containers and
                                  # images via read-only GET calls
                                  # against the Engine API
  nextcloud:
    enabled: false
    occ_path: /var/www/nextcloud/occ
    run_as: www-data
```

Run with:

```sh
wanderer agent --config /etc/wanderer/agent.yaml
```

Pass `--once` to run inspectors a single time and exit (useful for
cron-driven invocation or one-shot debugging).

### Hostnames without a TLD

The example above uses an FQDN (`webapp-01.example.internal`), but a
bare Linux hostname (`webapp-01`) is also accepted. The agent
registers its Target with `Kind: host`, which relaxes the public-
domain TLD requirement that applies to perimeter scans (`POST /scans`
still rejects a domain without a TLD). On disk, the Target row
carries `kind = 'host'` so an operator inspecting the database can
tell perimeter and agent rows apart.

## Trust model

Two deployment shapes:

- **Co-located** (`mode: local`): agent and core share a SQLite file.
  Simplest and recommended for a single Nextcloud-stack server that
  observes itself.
- **Split** (`mode: remote`): agent on each host posts findings to a
  central core over HMAC-signed HTTPS. The core stores a per-host
  shared secret and rejects requests with timestamps outside ±5
  minutes (replay protection). [ADR-0007](decisions/0007-agent-trust-model.md)
  records why HMAC-over-TLS was chosen over mTLS for the MVP.

## Least-privilege user

Run the agent as a dedicated user with the minimum group memberships
for the enabled inspectors:

```sh
sudo useradd --system --shell /usr/sbin/nologin --home-dir /var/lib/wanderer wanderer-agent
sudo usermod -aG docker wanderer-agent     # only if docker inspector enabled
```

`systemctl list-units` works without elevation. `dpkg-query` and
`rpm` work as any user. The Docker socket requires `docker` group
membership; the `occ` CLI requires read access to the Nextcloud
config (typically run as `www-data`).

## Wire-protocol summary (remote mode)

```
POST /scans/{target_id}/findings
Content-Type: application/json
X-Wanderer-Agent: webapp-01.example.internal
X-Wanderer-Timestamp: 2026-04-26T12:00:00Z
X-Wanderer-Signature: <base64 HMAC-SHA256(secret, timestamp + "\n" + body)>

{"findings": [...]}
```

The core verifies in this order:

1. The hostname is registered (a per-host secret exists).
2. The timestamp is within ±5 minutes of the core's clock.
3. The HMAC matches the registered secret.

A failure on any step returns 401. The 401 response body does not
distinguish which check failed — that prevents an attacker from
mapping out which hostnames are valid.

### Outbox: surviving a network outage

Remote-mode agents persist any batch the core rejects to a local
outbox directory (default `/var/lib/wanderer/agent/outbox`) and
drain it on the next tick before collecting fresh findings.

Behaviour per tick:

1. Drain spooled batches in oldest-first order. A successful POST
   removes the file; a failure aborts the drain so the file stays
   for the next tick.
2. Run inspectors and the egress probe.
3. POST the new batch with up to three attempts (0s / 250ms / 1s
   with ±25% jitter).
4. On persistent failure, write the batch to the outbox.

Configuration:

```yaml
core:
  mode: remote
  url: https://wanderer.example.internal
  hmac_secret_file: /etc/wanderer/agent.hmac
  target_id: t_abc
  outbox_dir: /var/lib/wanderer/agent/outbox  # default shown
  outbox_max_bytes: 104857600                 # 100 MiB; default
```

When the on-disk total exceeds `outbox_max_bytes`, the oldest
spooled batches are pruned before a new one is written. A corrupt
spool file (invalid JSON envelope) is renamed `<filename>.corrupt`
and skipped so it does not block the drain forever.

## Operating tips

- Run as a `systemd` service. SIGTERM stops the agent cleanly between
  inspector batches.
- Check `wanderer assess <scan-id>` after each agent batch to confirm
  the inventory data raises Completeness on relevant DICTU dimensions.
- For development: `wanderer agent --once --config example-agent.yaml`
  with a `mode: local` config and an inline SQLite file gives you the
  full pipeline in one process invocation.
