# Habitat run 07 — operationeel rules + Dutch copy (tasks 7.2 rest, 7.3, 7.4)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(specs/assessor/spec.md, design.md "Copy: one Dutch string table" and
"Reason codes").

## Scope — ONLY these tasks
- [ ] 7.2b The two operationeel rules:
  - `wand.operationeel.domain_expiry` — >90 days → soeverein; ≤90 →
    voldoende; ≤30 or past → afhankelijk; RDAP answered but the registry
    publishes no expiry → onbekend + reason `not_published_by_registry`
    (structural: it must NOT lower the dimension's completeness); RDAP
    unavailable → onbekend + reason `probe_unavailable`.
  - `wand.operationeel.variant_convergence` — every observed live path
    converges → soeverein; paths diverge, or a family is dead while its
    DNS record exists AND the scanner could reach that family →
    voldoende; a plain-HTTP path serves content without redirecting →
    afhankelijk; probe failed → onbekend + `probe_unavailable`. Paths
    marked `not_followed_budget` or `not_tested` never count as dead;
    the verdict says how many of 8 paths were observed. A run with
    `scanner_no_ipv6` carries that reason.
- [ ] 7.3 `internal/assessor/wand/accountability_nl.yaml`: per rule (all
  seven) the question, the verdict text per outcome
  (ja/nee/onbekend/n.v.t.) and one remediation line per failing outcome,
  in Dutch, embedded. Rules supply named parameters only
  (`{registrant}`, `{date}`, `{path}`, …). A load-time test fails when a
  rule has no entry, an outcome is missing, or an entry names a
  parameter the rule does not supply. No i18n machinery.
- [ ] 7.4 Follow-up from the run 01 review: the engine forces a
  rationale's score to onbekend whenever it carries a reason (spec: "A
  rationale with a reason SHALL score onbekend"), and a dimension whose
  every rationale is structural is marked not applicable explicitly
  rather than only being derivable from its rationale list. Test both,
  and keep old assessments (no `reason`) loading unchanged.

Also tick 7.2 in tasks.md now that both halves exist.

## Out of scope for this run
No UI work (run 08), no probe changes.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` green. Budget is $5 — leave room for your run report.
