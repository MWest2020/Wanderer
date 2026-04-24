# Proposal: Exporters — CSV + JSONL for Dashboards

## Intent

The Findings and Assessments Wanderer produces are valuable outside
Wanderer: analysts want them in Excel, operators want them in Grafana,
and governance tooling wants them in JSONL. This change adds a small
export surface — flat CSV and JSONL — that reads from the store and
writes to stdout or a file.

No server-side streaming export; no Prometheus pushgateway; no Parquet.
Those are all plausible follow-ups when a concrete consumer appears.

## Scope

**In scope:**

- CLI: `wanderer export <resource> [flags]` where `<resource>` is one
  of `findings | scans | assessments`.
- Formats: `--format csv` (default) and `--format jsonl`.
- Selectors: `--scan <id>`, `--since <timestamp>`, `--until <timestamp>`,
  `--probe <prefix>` (e.g. `--probe tls`), `--dimension <name>`.
- Output: stdout by default, `-o <path>` for file.
- Deterministic column ordering in CSV so diffs between exports are
  reviewable.

**Out of scope:**

- HTTP endpoint for export (operators already have CLI access to the
  DB file; a network endpoint is a separate trust question — revisit
  when a concrete remote consumer appears).
- Prometheus `textfile` or pushgateway format — `/metrics` already
  exposes runtime counters; per-finding time series is a different
  question, handled when scheduling + diffing lands.
- Streaming very large exports — at MVP scales a single scan produces
  ~20–100 Findings; we do not need chunked output yet.

## DICTU dimensions informed

Indirect. Exporters carry Findings to tools that serve the assessment
story across all five dimensions.

## Passive/active boundary

N/A — reads the local store, no outbound traffic.

## Parallel-safe

Touches only `cmd/wanderer/export.go` and `internal/export/`. No changes
to `internal/scanner`, `internal/store` (read-only), `pkg/models`.
Modifies `cmd/wanderer/main.go` command dispatch (one case statement
entry — mergeable).
