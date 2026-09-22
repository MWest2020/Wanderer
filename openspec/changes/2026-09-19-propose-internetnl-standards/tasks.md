# Tasks: standards dimension (Internet.nl via netnl)

## 1. Contract first
- [x] 1.1 Review NETNL-CONTRACT.md — gedaan 2026-09-22 tegen een
  ECHTE meting (api.westerweel.work, batch v2.7.0, westerweel.work),
  niet tegen de documentatie. Vijf correcties, zie design.md
  "Design gate outcome": `detail` per variant bestaat niet in de API
  (eis 4 vervalt), `error` ontbrak in de verdictverzameling, de
  categorienamen dragen een `web_`-voorvoegsel, `measured_at` bestaat
  alleen per batch, en de rapport-URL is ondoorzichtig (zelf-gehost).
  Het antwoord ligt als fixture in `fixtures/`.
- [x] 1.1b Het gecorrigeerde contract staat als
  `docs/netnl-findings-v1.md` in de internetnl-cli-repo, met de gouden
  fixtures voor web én mail uit echte metingen.
- [x] 1.2 netnl: `internetnl results <id> --format findings` bestaat,
  met determinisme-test en niet-nul exit bij een onvolledige batch
  (change `2026-09-22-findings-export` in die repo, runs 01–04, 676
  tests groen). De mail-batch is gedaan en bevestigde dezelfde platte
  vorm; beide fixtures staan aan weerszijden byte-identiek.
- [x] 1.3 Copy fixtures into Wanderer's testdata (same bytes) — done
  2026-09-22, `internal/scanner/testdata/findings-v1-{web,mail}-20260922.json`,
  sha256-verified identical to the change's `fixtures/` copies.

## 2. Wanderer importer
- [x] 2.1 `wanderer import internetnl <file>`: parse, target-match,
  persist under import-kind scan; WARN+skip for unknown domains and
  malformed entries; abort on schema-version mismatch; idempotent
  re-import — done 2026-09-22. Idempotency keys on the file's sha256:
  the measured netnl-findings/v1 schema (design.md "Design gate
  outcome") carries no request ID anywhere, per-domain or per-batch,
  so "file hash + request ID" as originally phrased isn't buildable
  against the real fixture. **Corrected by habitat run 02b
  (2026-09-22):** file hash *alone* was wrong — a domain skipped for
  lack of a matching target still marked the whole file as imported,
  so creating the target afterwards and re-importing silently did
  nothing. The key is now **(file hash, domain)**: see
  `runs/02b-import-zonder-doel.md`.
- [x] 2.2 Store: import-kind scans coexist with perimeter scans;
  assessor reads newest per kind — reopened by habitat run 03b
  (2026-09-22): `Store.LatestScanByModus` (run 03) only gave a
  primitive, and no caller used it, so every `assessor.Assess` call
  site scored `scan.Findings` alone — a perimeter scan saw zero
  `internetnl.*` evidence, and each import scan saw only its own
  type (web or mail), scoring some `standards` rules (dnssec, ipv6,
  rpki, which have tests on both sides) on half the evidence.
  **Fixed 2026-09-22:** `Store.LatestImportFindings` (new,
  `internal/store/netnlimport.go`) groups a target's import-modus
  Findings by their `type` attribute (web/mail) and keeps only the
  Findings from each type's most-recently-started scan, so a fresh
  import of one type supersedes only that type. `Store.
  FindingsForAssessment(ctx, scan)` combines that with `scan`'s own
  non-import Findings (perimeter/inventory/egress/drift pass through
  unchanged — only `standards`-feeding import Findings are
  correlated) and replaces every `scan.Findings` argument to
  `assessor.Assess` across the six call sites (`internal/api/api.go`,
  `internal/mcp/tools.go`, `internal/scheduler/assess.go` +
  `assessmentPersister` interface, `internal/ui/ui.go` ×2). No change
  to any standards rule itself — `scoreStandardsCategory` already
  worked correctly given the right findings, it just never received
  them. `internal/fixtures/seed.go` untouched: it seeds no netnl
  import scenario. **Gap found and fixed 2026-09-22 (task-ref
  `tasks/2026-09-22-cli-assess-correlatie.md`):** the enumeration
  above missed a sixth caller, `cmd/wanderer/assess.go` — the CLI's
  `wanderer assess` command, which is the documented route for this
  whole feature (`internetnl results` → `wanderer import internetnl`
  → `wanderer assess`). It still scored `scan.Findings` directly, so
  a perimeter scan assessed via the CLI reported every `standards`
  rule as "not measured" even with both imports present in the same
  database. Now calls `st.FindingsForAssessment(ctx, scan)` once per
  scan, same as the other five call sites.
  `internal/fixtures/seed.go`'s direct use of `assessor.Assess`
  (`persisted`, not `FindingsForAssessment`) was checked too: it
  seeds single-scan demo data with no separate import scans to
  correlate against, so it's unaffected and correctly left as is.
- [x] 2.3 Tests pinning the corrected behaviour (habitat run 03b) —
  `internal/store/standards_correlation_test.go`:
  `TestFindingsForAssessment_CorrelatesWebAndMailAcrossScans` persists
  a perimeter scan plus separate web- and mail-import scans (both
  golden fixtures) and asserts `wand.standards.ipv6` sees all 9 tests
  (voldoende) and `wand.standards.rpki` all 10 (soeverein) regardless
  of which of the three scans is assessed — including the perimeter
  scan, previously blind to import evidence entirely.
  `TestFindingsForAssessment_FreshMailImportReplacesOnlyMail` imports
  web once, mail twice (the second with every `mail_ipv6` test
  flipped to passed), and asserts `wand.standards.ipv6` moves to
  soeverein without doubling its evidence count, while
  `wand.standards.dnssec` (untouched by mail) still sees all 6 tests
  unchanged — the fresh import replaces only its own type. Verified
  with the reparatie eruit: temporarily reverting
  `FindingsForAssessment` to `return scan.Findings, nil` fails both
  new tests exactly as expected (perimeter: 0 evidence for
  ipv6/rpki; web/mail scans: half the evidence each; the fresh-mail
  test: old and new mail_ipv6 results co-existing instead of the old
  being replaced) — restored before committing. **Extended 2026-09-22**
  with a CLI-level test for the sixth caller found above:
  `cmd/wanderer/assess_correlation_test.go`
  `TestRunAssess_CorrelatesImportedStandardsFindings` imports web and
  mail via `runImportInternetnl`, assesses a perimeter scan via
  `runAssess`, and asserts the persisted assessment shows
  `wand.standards.rpki` at 10 tests and `wand.standards.ipv6` at 9.
  Verified with the reparatie eruit (temporarily reverting
  `assess.go` to `scan.Findings`): fails exactly as expected (0
  evidence for both rules, "not measured" rendered) — restored before
  committing.

## 3. Assessor
- [x] 3.1 `standards` dimension registration + six rules (verdict
  mapping only); table-driven tests incl. not-measured, stale,
  mixed-verdict, and no-double-score (security.txt) paths — done
  2026-09-22. New `models.DimensionStandards`, registered in
  `assessor.WandDimensions`; six rules in
  `internal/assessor/wand/standards_rules.go`, wired into
  `wand.DefaultRules()`. Category→rule mapping is the one measured in
  runs/03-assessor.md (the API already resolves `web_ns_rpki_*` /
  `mail_ns_rpki_*` / `mail_mx_ns_rpki_*` onto `web_rpki`/`mail_rpki`
  via its own metadata hierarchy — no prefix-matching needed on
  Wanderer's side). `error` status scores onbekend ("meting mislukt"),
  never afhankelijk; a category whose every test is
  `not_tested`/`error` scores onbekend, never soeverein. Two new
  reason codes (`not_measured`, `measurement_stale`) registered in
  reason.go. Golden-fixture counts (76 tests, six rules + unscored
  `web_appsecpriv`) pinned in `standards_rules_test.go` against the
  task-1.3 testdata copies. Fixed two ADDED requirements in
  `specs/assessor/spec.md` that failed `openspec validate --strict`
  (SHALL only appeared past the parser's first line of a soft-wrapped
  paragraph) — wording only, no semantic change.
- [x] 3.2 `standards.max_age` config (default 30d) — done 2026-09-22.
  `StandardsMaxAgeDays`/`StandardsMaxAge` in
  `internal/assessor/wand/standards_rules.go`, surfaced as an
  `assessor.Threshold` on every standards rule (same pattern as
  `domainExpirySafeDays`/`certValidityExpiringSoonDays`) so the
  boundary is documented on each rule page, not just in code. Full
  YAML/CLI wiring (an operator-facing `standards.max_age` override in
  `serveconfig.Config`, threaded through every `wand.DefaultRules()`
  call site) is out of this run's scope — see the run report.

## 4. Wrap-up
- [x] 4.1 UI: verify standards renders via existing rule rendering;
  report URL clickable in evidence; "not measured" pill state — done
  2026-09-22 (habitat run 04). The generic rationale table
  (`assessment.tmpl`/`reporting_rule.tmpl`) already rendered standards
  rules correctly for two of the three checks — "not measured" text
  and the onbekend badge came for free from the existing engine/UI
  contract, and the vlootscherm's x/n is scoped to the seven
  sovereignty flows only (`internal/ui/flows.go`), so a standards
  rule never enters that count at all; `WorstScoreCovering`/
  `WorstScore` (used by the trends "covers: ..." pill) already
  exclude onbekend dimensions, so a not-measured target never sinks.
  Chose to lock both behaviours down with new tests
  (`TestWorstScoreCovering_StandardsNotMeasuredNeverCounts`,
  `TestWorstScoreCovering_StandardsWithEvidenceCounts`,
  `TestBuildFleetScore_StandardsNeverEntersTheFlowCount`) rather than
  leave them implicit. The one real gap: the Internet.nl report URL
  was shown as inert text inside the Verdict sentence, not a link.
  Fixed with a new `linkify` template func
  (`internal/ui/linkify.go`) applied to `.Verdict` in both templates —
  finds `http(s)://` substrings, validates scheme+host before
  linking, HTML-escapes everything else; unsafe/non-http(s) matches
  stay plain text. No assessor/engine changes.
- [x] 4.2 docs/reference (assessor, findings) + how-to "Feed
  Internet.nl results from CI"; note the permanent non-goals list in
  docs/explanation — done 2026-09-22. Added "The `standards`
  dimension" to `docs/reference/assessor.md` (rule↔category mapping,
  verdict rules, `not_measured`/`measurement_stale` reason codes,
  cross-scan correlation), an "Internet.nl import Findings" section to
  `docs/reference/findings.md` (`internetnl.<type>.<test>` shape,
  `SourceModusImport`), `docs/how-to/internetnl-ci.md` (the three real
  commands, plus the `--format findings` non-zero-on-incomplete-batch
  exit code a CI step must gate on), and
  `docs/explanation/internetnl-non-goals.md` (the permanent
  non-reimplementation list from proposal.md, why it's permanent not
  deferred, and the deferral policy for future probes). Linked from
  `docs/index.md`.
- [x] 4.3 CHANGELOG; commit + push; archive — CHANGELOG entry added
  under `[Unreleased]` 2026-09-22. Commit/push/archive left to Mark
  per the builder role (never merges, never pushes).
- [x] 4.4 Draft v2 follow-up proposal stub (facade webhook ingestion)
  once v1 has run in CI for a few weeks — done 2026-09-22,
  `openspec/changes/2026-09-22-internetnl-facade-v2-stub.md`: proposal
  heading, why it waits, no specs/tasks (a stub, explicitly not a
  change).
