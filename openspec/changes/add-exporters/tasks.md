# Tasks: Exporters

## 1. Store query helpers

- [ ] 1.1 `store.ListFindings(ctx, Selectors)` returning a streaming iterator (rows.Next style)
- [ ] 1.2 `store.ListScans(ctx, Selectors)` similar
- [ ] 1.3 `store.ListAssessments(ctx, Selectors)` — only wired if assessor types exist at compile time (build tag or conditional import)
- [ ] 1.4 Tests covering each selector combination

## 2. Writers

- [ ] 2.1 `internal/export/csv_findings.go` — deterministic column order, JSON-encoded attributes column
- [ ] 2.2 `internal/export/csv_scans.go`
- [ ] 2.3 `internal/export/csv_assessments.go` (gated on assessor being present)
- [ ] 2.4 `internal/export/jsonl.go` — generic JSONL writer
- [ ] 2.5 Golden-file tests for each writer

## 3. CLI

- [ ] 3.1 `cmd/wanderer/export.go` — subcommand routing, flag parsing, selector construction
- [ ] 3.2 Register `export` in `cmd/wanderer/main.go`
- [ ] 3.3 Smoke test against a scratch DB: export → parse with stdlib `encoding/csv` → assert row count

## 4. Docs + CHANGELOG

- [ ] 4.1 `docs/exporters.md` — formats, selectors, examples (Excel, jq pipe, Grafana CSV datasource)
- [ ] 4.2 Update `docs/README.md` index
- [ ] 4.3 CHANGELOG entry under `Added`
