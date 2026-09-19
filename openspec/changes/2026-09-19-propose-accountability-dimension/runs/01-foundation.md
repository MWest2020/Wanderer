# Habitat run 01 — foundation (tasks 2.1–2.4)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(proposal.md, design.md "Reason codes" + "Design gate outcome",
specs/assessor/spec.md).

## Scope — ONLY these tasks
- 2.1 `models.DimensionHint`: add `accountability`; `Valid()` accepts it.
- 2.2 Rename `DICTUDimensions` → `WandDimensions` (append
  `accountability`); no consumer or test assumes a dimension count.
- 2.3 Reason codes: `Reason` on `RuleResult`, `reason,omitempty` on
  `models.Rationale`; one registry table (code → class
  `structural`|`gap` + subject `target`|`scanner`) seeded with
  `registry_redacted`, `not_published_by_registry` (structural,
  target), `scanner_no_ipv6` (structural, scanner), `probe_unavailable`
  (gap, target). Unknown
  code → test failure. Generic: no accountability-specific branches.
- 2.4 `scoreDimension`: structural rationales excluded from worst
  score and completeness denominator; all-structural dimension
  reported as not applicable. Table-driven tests, including loading
  an assessment JSON from before this change (no `reason`, five
  dimensions) without error.

Check off exactly 2.1–2.4 in tasks.md.

## Out of scope for this run
No rules, no probes, no YAML lists, no store migration, no UI. The
`accountability` dimension will have zero rules after this run and
must score as it does today for a rule-less dimension.

## Done means
`go build ./...`, `go vet ./...` and `go test ./...` green;
`openspec validate 2026-09-19-propose-accountability-dimension`
green. If the Go toolchain is unavailable in the worker, STOP and
report — do not edit code you cannot compile.
