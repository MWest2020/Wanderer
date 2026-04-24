# Design: Scheduling + Drift

## Component placement

```
internal/scheduler/
  scheduler.go        # in-process cron using github.com/robfig/cron/v3
  config.go           # wanderer-schedules.yaml parsing
  run.go              # invoke scanner.Scan with per-schedule config

internal/drift/
  diff.go             # Scan A vs Scan B → []Finding (source_modus=drift)
  rules.go            # what counts as a drift worth surfacing
  rules_test.go

cmd/wanderer/diff.go  # CLI: wanderer diff <scan-a> <scan-b>
```

## Schedules file

```yaml
# wanderer-schedules.yaml
schedules:
  - name: conduction-apex-daily
    target:
      domain: conduction.nl
    cron: "0 6 * * *"         # 06:00 every day
    probes: [dns, tls, ip, http]
    timeout: 5m

  - name: customer-weekly
    target:
      domain: customer.example.nl
      related: [customer.example.com]
    cron: "30 4 * * 1"        # 04:30 Monday
    timeout: 10m
```

Cron parsing via `github.com/robfig/cron/v3` — battle-tested, used
everywhere.

## Scheduler lifecycle

The scheduler is owned by `wanderer serve` and shares its lifecycle:

1. On startup: read `wanderer-schedules.yaml` (or file from
   `--schedules` flag / `WANDERER_SCHEDULES` env), validate cron
   expressions, register jobs.
2. On SIGHUP: re-read the file (explicit operator action, not
   hot-reload via file watcher — see valkuil below).
3. On SIGINT/SIGTERM: cancel in-flight jobs via context, wait up to
   `shutdown_timeout` (default 30s), then exit.

Each job runs `scanner.Scan(ctx, target)` with the server's existing
store and scanner instance. After the scan finishes, it invokes
`drift.Compute(ctx, store, newScan)` which looks up the previous scan
for the same target and emits drift Findings.

## Drift rules

`internal/drift/rules.go` lists what counts as a surface-worthy drift:

| Category                       | Triggers on                                     | Severity    |
| ------------------------------ | ----------------------------------------------- | ----------- |
| `drift.tls.issuer_changed`     | `tls.issuer` attribute `issuer_cn` differs     | finding     |
| `drift.tls.days_left_dropped`  | `tls.validity.days_left` < 30 for first time   | concern     |
| `drift.dns.mx_set_changed`     | set-diff of `dns.mx.host` values               | observation |
| `drift.dns.ns_set_changed`     | set-diff of `dns.ns.host` values               | observation |
| `drift.ip.country_changed`     | subject host's `ip.asn.country` differs        | finding     |
| `drift.http.third_party_added` | new host in `http.third_party` subjects        | observation |
| `drift.http.third_party_removed` | host disappeared                              | info        |
| `drift.http.security_headers`  | presence set changed                           | observation |
| `drift.inventory.package`      | added or version-changed (requires inventory)  | observation |
| `drift.egress.endpoint_changed` | subject host changed per config key           | finding     |

Each rule is a Go function taking `(prev, curr *models.Scan)` and
returning `[]models.Finding`. Adding a drift rule is a one-file edit
plus a test.

**No drift findings are emitted for the first scan of a target** —
there is nothing to compare against. A `drift.baseline_established`
info Finding is emitted instead, so an operator can distinguish "new
target, no history" from "no changes since last scan" (which emits
`drift.no_changes`).

## CLI: `wanderer diff`

```sh
wanderer diff s_abc s_def
```

Outputs markdown identical in shape to the drift Findings that would
have been persisted, without touching the store. Useful for ad-hoc
"what changed" queries against historical scans.

## External systems

None new. The scheduler reads the store (to fetch previous scans) and
writes to it (drift Findings). The scanner handles all network
activity already.

## Failure modes

- **Cron expression invalid.** Fail at startup (not silently), print
  which schedule entry is bad, exit non-zero. The operator sees it
  before the schedule silently never fires.
- **Previous scan for target missing.** Emit
  `drift.baseline_established`; this is the expected first-run state.
- **Two schedules for same target overlap.** Scanner serialises via
  the store's write lock; one waits for the other. We do not add a
  per-target mutex in the scheduler because it would hide the
  contention from metrics.
- **Scheduler job panics.** Recover, log the panic, emit a
  `scheduler.panic` info Finding, continue running other schedules.

## Clever valkuil

1. **Watcher-based hot-reload.** Tempting, keeps "config is live"
   feeling. Wrong — a misconfigured cron expression during a live
   edit could miss a run or duplicate it, and the operator gets no
   signal. Explicit SIGHUP is the boring, auditable choice.

2. **Distributed scheduling with leader election.** Premature. MVP
   is one `wanderer serve` instance. When we need two, we introduce
   either external orchestration (Kubernetes CronJob with Wanderer as
   worker) or leader election — but that is a separate proposal with
   its own trade-off discussion.

3. **Complex diff logic (semantic cert-chain equivalence, redirect-
   graph normalisation).** For MVP, set-diff on stable attribute
   keys is what we need. Semantic equivalence is a yak-shave.

## Relation to other changes

- **`add-assessor`** — drift Findings get their own
  `DimensionHint` (copied from the underlying rule). The assessor
  treats them like any other Finding. An optional future extension
  surfaces drift as a separate section in the markdown report
  ("what changed since the last assessment").
- **`add-exporters`** — drift Findings flow through CSV/JSONL like
  any other Finding, filterable by `source_modus: drift`.
- **`add-mcp-server`** — new MCP tool `diff_scans(scan_a, scan_b)`
  calls into `internal/drift.Compute` and returns the Findings.
- **`add-inventory-probe` / `add-egress-probe`** — provide additional
  attributes to diff. Each probe change adds its drift rules to
  `internal/drift/rules.go`.

## Coverage

- `internal/drift/rules.go` target: 90% — rules are pure functions.
- `internal/scheduler` target: 75%.
