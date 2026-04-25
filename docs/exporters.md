# Exporters

Wanderer's findings, scans, and assessments are valuable outside the
binary: analysts want them in Excel, operators want them in Grafana,
governance tooling wants them in JSONL. The `wanderer export`
subcommand reads from the local SQLite store and writes to stdout or
a file.

## Synopsis

```sh
wanderer export <resource> [flags]

  <resource>    findings | scans | assessments
  --format      csv (default) | jsonl
  -o <path>     write to file (default stdout)
  --db <path>   SQLite DSN (default ./wanderer.db)
  --scan <id>   limit to one scan
  --probe <p>   limit findings to probe_id starting with <p> (e.g. tls)
  --dimension   limit findings/assessment rows to one DICTU dimension
  --since <ts>  RFC 3339 lower bound on created_at
  --until <ts>  RFC 3339 upper bound on created_at
  --include-evidence=false
                drop the evidence field from JSONL output
```

Selectors are AND-combined and pushed down to the SQL query — Wanderer
never reads more rows than the filters select.

## CSV column schemas

All timestamps are RFC 3339 UTC. Column order is fixed; two runs over
the same data produce byte-identical output.

### Findings

```
id,scan_id,probe_id,dimension_hint,criterium_hint,subject,severity,created_at,attributes_json
```

`attributes_json` is the probe-specific `Attributes` map serialised as
a single JSON string. Flattening keys into columns would break each
time a probe added a new attribute; a single JSON column is the
boring choice that keeps the column set stable.

`evidence` is deliberately not in CSV — it is often binary or large.
Use JSONL (or query the API/DB directly) when you need it.

### Scans

```
id,target_id,domain,started_at,ended_at,status,error,finding_count
```

`finding_count` is computed in the query; one row per scan.

### Assessments

```
id,scan_id,framework,created_at,dimension,score,completeness
```

One row per (assessment × dimension). The detailed rationale list is
not flattened — use JSONL when the rationale matters.

## JSONL

One JSON object per line, no wrapping array, no trailing comma. The
findings line shape:

```json
{"id":"f_abc","scan_id":"s_abc","probe_id":"tls.issuer","dimension_hint":"juridisch","subject":"example.nl","severity":"finding","attributes":{"issuer_country":["US"]},"evidence":"<base64>","created_at":"2026-04-24T19:50:11Z"}
```

`evidence` is base64-encoded by default. Pass `--include-evidence=false`
to drop the field entirely.

Assessments JSONL emits a full `Assessment` per line, including the
markdown report under `report` if it was persisted.

## Recipes

### Open in Excel

```sh
wanderer export findings --scan s_abc -o findings.csv
open findings.csv
```

### Pipe to jq

```sh
wanderer export findings --probe tls --format jsonl |
  jq 'select(.dimension_hint == "juridisch")'
```

### Grafana CSV datasource

Point a CSV datasource at a path served by `wanderer export scans -o
scans.csv` rendered on a cron. The fixed column order keeps the
panel definitions stable.

### Diff two exports

```sh
wanderer export findings --scan s_old -o /tmp/old.csv
wanderer export findings --scan s_new -o /tmp/new.csv
diff /tmp/old.csv /tmp/new.csv
```

The CSV is byte-stable for stable input, so `diff` is meaningful
without sorting first.
