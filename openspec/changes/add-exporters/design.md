# Design: Exporters

## Placement

```
internal/export/
  export.go          # dispatch by resource + format
  csv_findings.go    # Findings → CSV writer
  csv_scans.go       # Scans → CSV writer
  csv_assessments.go # Assessments → CSV writer (only compiled if assessor is present)
  jsonl.go           # generic JSONL writer using encoding/json

cmd/wanderer/export.go  # CLI flag parsing + selector application
```

All writers accept `io.Writer`. Store access is read-only via the
existing `*store.Store` handle.

## CSV column schema — Findings

Fixed, deterministic column order:

```
id,scan_id,probe_id,dimension_hint,criterium_hint,subject,severity,created_at,attributes_json
```

- `attributes_json` is the `Attributes` map serialised as a single JSON
  string. Flattening keys into columns would break as soon as a probe
  adds a new attribute; keeping a JSON column is the boring choice.
- `evidence` is omitted from CSV (often binary, often large). It is
  retrievable via the API or direct DB query when needed.
- Timestamps are RFC 3339 UTC.

## CSV column schema — Scans

```
id,target_id,domain,started_at,ended_at,status,error,finding_count
```

`finding_count` is computed via `SELECT COUNT(*)` per scan, not a
separate row set, so the CSV stays one-row-per-scan.

## CSV column schema — Assessments

(Only present when `add-assessor` has landed.)

```
id,scan_id,framework,created_at,dimension,score,completeness
```

One row per dimension per assessment. The detailed rationale list is
not flattened into CSV — use JSONL if the rationale matters.

## JSONL format

One JSON object per line. For findings:

```json
{"id":"f_abc","scan_id":"s_abc","probe_id":"tls.issuer","subject":"example.nl","severity":"finding","dimension_hint":"juridisch","attributes":{"issuer_country":["US"]},"created_at":"2026-04-23T19:50:11Z"}
```

`evidence` is base64-encoded when present in JSONL so downstream
consumers can read it without double-decoding. A flag
`--include-evidence=false` (default true) drops it.

## External systems

None. Reads the local SQLite file.

## Failure modes

- **Very large export.** Writers stream row-by-row — they do not hold
  the full result set in memory.
- **Output path not writable.** Fail fast with a clear stderr message
  and exit 1.
- **No rows match the selectors.** Exit 0 with an empty CSV (header
  only) or empty JSONL (zero lines) — silence for an empty set is a
  bug.
- **Unexported probe attribute type.** `attributes_json` uses
  `encoding/json`; non-serialisable types are refused at `Finding.Validate`
  in the first place, so we inherit that guarantee.

## Clever valkuil

Tempting: add `--format parquet`, `--format arrow`, a gRPC streaming
export, a plugin system for custom exporters. All of that is a future
problem. CSV and JSONL together cover ≥95% of real consumers
(spreadsheets, ETL pipelines, grep). Anything more elaborate waits for
a concrete consumer to ask for it.

## Tests

- Unit tests per format: given fabricated records, assert byte-exact
  output.
- Golden-file test for a representative Findings export.
- Smoke test: run a scan (or use a fixture DB), run
  `wanderer export findings --format csv`, pipe through `csvtool` or
  similar to verify row count matches `SELECT COUNT(*)`.
