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
- [ ] 2.2 Store: import-kind scans coexist with perimeter scans;
  assessor reads newest per kind — done 2026-09-22. Findings.SourceModus
  gained `import` (no new Scan.Kind column, per design decision);
  `Store.LatestScanByModus` gives a future assessor its "newest scan
  for this modus" primitive without this run reaching into
  assessor/rules territory.

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
- [ ] 4.1 UI: verify standards renders via existing rule rendering;
  report URL clickable in evidence; "not measured" pill state.
- [ ] 4.2 docs/reference (assessor, findings) + how-to "Feed
  Internet.nl results from CI"; note the permanent non-goals list in
  docs/explanation.
- [ ] 4.3 CHANGELOG; commit + push; archive.
- [ ] 4.4 Draft v2 follow-up proposal stub (facade webhook ingestion)
  once v1 has run in CI for a few weeks.
