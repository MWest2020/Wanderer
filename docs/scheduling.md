# Scheduling

Wanderer can run scans on a cron schedule from inside `wanderer
serve`. One scan is a snapshot; a sovereignty observatory becomes
useful when it runs continuously and tells you what *changed* —
that is what scheduling and [drift](drift.md) deliver together.

## Configuration

Schedules live in a YAML file. Pass its path with `--schedules` or
`WANDERER_SCHEDULES`.

```yaml
# wanderer-schedules.yaml
schedules:
  - name: conduction-apex-daily
    target:
      domain: conduction.nl
    cron: "0 6 * * *"      # 06:00 every day
    timeout: 5m

  - name: customer-weekly
    target:
      domain: customer.example.nl
      related: [customer.example.com]
    cron: "30 4 * * 1"     # 04:30 every Monday
    timeout: 10m
```

Cron syntax follows POSIX `cron(5)` — five fields, no seconds field.
Cron expressions are validated at startup. An invalid expression
fails the process; the operator sees which entry is bad before any
schedule silently never fires.

## Lifecycle

- **Startup**: Wanderer reads the schedules file, validates every
  cron expression, and registers the jobs. If `--schedules` is
  empty, the scheduler is simply not started — `wanderer serve`
  works exactly as before.
- **SIGHUP**: Wanderer re-reads the file. Existing in-flight jobs
  finish on their current schedule; subsequent ticks use the new
  set. SIGHUP is the only reload signal — Wanderer does not watch
  the file for changes (see ADR-0006).
- **SIGINT / SIGTERM**: in-flight jobs receive context cancellation;
  the process waits up to 30 seconds for them to wind down before
  exiting.

## What each tick does

Each cron tick runs `scanner.Scan` against the configured target
using the same pipeline `wanderer scan` and the HTTP `POST /scans`
endpoint use. After the scan finishes, the scheduler invokes the
[drift engine](drift.md) which compares the new scan against the
previous one for the same target and persists drift Findings to the
store.

That is the whole loop: scan, diff, persist. Drift Findings flow
through the same exporters, MCP resources, and assessor pipeline as
probe-produced Findings — they are not a parallel data path.

## Failure modes

- **Job panics**: the scheduler recovers, logs `scheduler.panic`,
  attaches a synthetic Finding to the most recent scan, and continues
  with other schedules.
- **Two schedules for the same target overlap**: the scanner
  serialises through the store's write lock; one scan waits for the
  other. We do not add a per-target mutex in the scheduler — that
  would hide contention from metrics.
- **Drift compute fails**: the scan still persists; the drift
  Findings simply are not produced this round. The next tick
  computes drift against whatever scans exist at that point.

## Operating tips

- Run `wanderer serve --schedules wanderer-schedules.yaml` under a
  process supervisor (`systemd`, `supervisord`, `pm2`, …) so SIGHUP
  reloads are part of normal operations.
- A `systemd` snippet:

  ```ini
  [Service]
  ExecStart=/usr/local/bin/wanderer serve --schedules /etc/wanderer/schedules.yaml
  ExecReload=/bin/kill -HUP $MAINPID
  Restart=on-failure
  ```

- For ad-hoc "what changed between these two scans?" without waiting
  for the next tick, use [`wanderer diff`](drift.md#wanderer-diff).
