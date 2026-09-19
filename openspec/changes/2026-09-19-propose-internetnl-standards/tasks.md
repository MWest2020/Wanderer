# Tasks: standards dimension (Internet.nl via netnl)

## 1. Contract first
- [ ] 1.1 Review NETNL-CONTRACT.md; land it in the internetnl-cli
  repo with golden fixtures (web + mail).
- [ ] 1.2 netnl: `--format findings` exporter + determinism test +
  non-zero exit on incomplete batch.
- [ ] 1.3 Copy fixtures into Wanderer's testdata (same bytes).

## 2. Wanderer importer
- [ ] 2.1 `wanderer import internetnl <file>`: parse, target-match,
  persist under import-kind scan; WARN+skip for unknown domains and
  malformed entries; abort on schema-version mismatch; idempotent
  re-import (file hash + request ID).
- [ ] 2.2 Store: import-kind scans coexist with perimeter scans;
  assessor reads newest per kind.

## 3. Assessor
- [ ] 3.1 `standards` dimension registration + six rules (verdict
  mapping only); table-driven tests incl. not-measured, stale,
  mixed-verdict, and no-double-score (security.txt) paths.
- [ ] 3.2 `standards.max_age` config (default 30d).

## 4. Wrap-up
- [ ] 4.1 UI: verify standards renders via existing rule rendering;
  report URL clickable in evidence; "not measured" pill state.
- [ ] 4.2 docs/reference (assessor, findings) + how-to "Feed
  Internet.nl results from CI"; note the permanent non-goals list in
  docs/explanation.
- [ ] 4.3 CHANGELOG; commit + push; archive.
- [ ] 4.4 Draft v2 follow-up proposal stub (facade webhook ingestion)
  once v1 has run in CI for a few weeks.
