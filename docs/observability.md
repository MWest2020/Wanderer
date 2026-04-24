# Observability

Wanderer emits three kinds of signal: structured logs, Prometheus counters,
and (future) OpenTelemetry traces.

## Logs

`log/slog` is wired in from the top: the CLI builds a `*slog.Logger` and
the scanner derives per-scan and per-probe loggers via `With`. The HTTP
API adds a request-ID middleware and propagates a logger with that ID on
the request context. Downstream handlers can retrieve it via
`api.LoggerFrom(ctx)`.

Log format:

- CLI `scan` subcommand — text handler on stderr (human-readable).
  Pass `--json-logs` to switch to JSON.
- CLI `serve` subcommand — JSON handler, always.

Key fields: `scan_id`, `probe`, `domain`, `req_id`, `status`, `ms`.

## Metrics

`/metrics` on the `wanderer serve` HTTP server exposes Prometheus
metrics via `promhttp`. The MVP counters are:

- `wanderer_probe_runs_total{probe}`
- `wanderer_probe_failures_total{probe,reason}` (reason: timeout | panic | error)
- `wanderer_scan_started_total`
- `wanderer_scan_ended_total{status}` (status: complete | partial | failed)

Add a counter here only when an operator will actually alert on it.
Gauges for in-flight scans and histograms for probe duration are
plausible next steps.

## Traces

OpenTelemetry tracing across scanner → probes is explicitly marked
**optional for the MVP** in the change design. It is not wired in. When
we add it, the right integration point is `Scanner.runOne`: create a
span per probe, set `probe.id` as an attribute, record findings count
as a span event. The public `probe.Probe` interface does not need to
change — the scanner carries the tracer, the probe carries only its
`context.Context`.
