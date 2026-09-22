---
status: draft
last_reviewed: 2026-07-12
---

# Scheduling

Wanderer can run scans on a cron schedule from inside `wanderer
serve`. One scan is a snapshot; a sovereignty observatory becomes
useful when it runs continuously and tells you what *changed* —
that is what scheduling and [drift](../reference/drift.md) deliver together.

## Configuration

Schedules live in a YAML file. Pass its path with `--schedules` or
`WANDERER_SCHEDULES`.

```yaml
# wanderer-schedules.yaml
schedules:
  - name: voorbeeld-apex-daily
    target:
      domain: voorbeeld.nl
    cron: "0 6 * * *"      # 06:00 every day
    timeout: 5m

  - name: customer-weekly
    target:
      domain: customer.example.nl
      related: [customer.example.com]
    cron: "30 4 * * 1"     # 04:30 every Monday
    timeout: 10m
    assess: false           # scan only, no judged report for this one
```

Cron syntax follows POSIX `cron(5)` — five fields, no seconds field.
Cron expressions are validated at startup. An invalid expression
fails the process; the operator sees which entry is bad before any
schedule silently never fires.

`assess` is optional and defaults to `true`: a successful scan is
judged with both rule packs unless the schedule sets `assess: false`.

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
endpoint use. Once the scan succeeds, the scheduler judges it with
both rule packs (wand and EUCSF) — the same assessor pipeline
`wanderer assess --framework both` and `POST /scans/{id}/assessments`
use — and persists the resulting Assessments, unless the schedule
sets `assess: false`. After that, the scheduler invokes the
[drift engine](../reference/drift.md) which compares the new scan against the
previous one for the same target and persists drift Findings to the
store.

That is the whole loop: scan, assess, diff, persist. Drift Findings
flow through the same exporters, MCP resources, and assessor pipeline
as probe-produced Findings — they are not a parallel data path.

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
- **Assessing fails**: the scan still persists; the scheduler logs
  `scan.assess_failed` and moves on to drift compute and the next
  schedule. The report page shows no judgement for that scan until a
  later run (or a manual `wanderer assess`) succeeds.

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
  for the next tick, use [`wanderer diff`](../reference/drift.md#wanderer-diff).
