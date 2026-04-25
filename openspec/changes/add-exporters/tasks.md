# Tasks: Exporters

## 1. Store query helpers

- [x] 1.1 `store.ListFindings(ctx, Selectors)` returning a streaming iterator (rows.Next style)
- [x] 1.2 `store.ListScans(ctx, Selectors)` similar
- [x] 1.3 `store.ListAssessments(ctx, Selectors)` — only wired if assessor types exist at compile time (build tag or conditional import)
- [x] 1.4 Tests covering each selector combination

## 2. Writers

- [x] 2.1 `internal/export/csv_findings.go` — deterministic column order, JSON-encoded attributes column
- [x] 2.2 `internal/export/csv_scans.go`
- [x] 2.3 `internal/export/csv_assessments.go` (gated on assessor being present)
- [x] 2.4 `internal/export/jsonl.go` — generic JSONL writer
- [x] 2.5 Golden-file tests for each writer

## 3. CLI

- [x] 3.1 `cmd/wanderer/export.go` — subcommand routing, flag parsing, selector construction
- [x] 3.2 Register `export` in `cmd/wanderer/main.go`
- [x] 3.3 Smoke test against a scratch DB: export → parse with stdlib `encoding/csv` → assert row count

## 4. Docs + CHANGELOG

- [x] 4.1 `docs/exporters.md` — formats, selectors, examples (Excel, jq pipe, Grafana CSV datasource)
- [x] 4.2 Update `docs/README.md` index
- [x] 4.3 CHANGELOG entry under `Added`
