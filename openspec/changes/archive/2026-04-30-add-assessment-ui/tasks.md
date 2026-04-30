## 1. Rule.Rationale field

- [x] 1.1 Add `Rationale string` to `assessor.Rule` in `internal/assessor/rule.go`; document it in the godoc above the struct
- [x] 1.2 Populate `Rationale` on every rule in `internal/assessor/dictu/rules.go` (one paragraph each, plain language, focused on the consequence of the rule firing)
- [x] 1.3 Populate `Rationale` on every rule in `internal/assessor/eucsf/rules.go`
- [x] 1.4 Add `TestEveryRuleHasRationale` to `internal/assessor/dictu/rules_test.go` and `internal/assessor/eucsf/rules_test.go` that fails the build when any rule's `Rationale` is empty

## 2. Rule registry lookup

- [x] 2.1 Add `internal/ui/registry.go::lookupRule(framework, criteriumID)` returning the live `assessor.Rule` plus an ok flag
- [x] 2.2 Test the helper with a known DICTU rule, a known EUCSF rule, an unknown framework, and a retired CriteriumID

## 3. Assessment page handler

- [x] 3.1 Add `assessmentHandler(st, tmpl)` to `internal/ui/ui.go`, mounted at `r.Get("/scans/{id}/assessment", ...)`
- [x] 3.2 Handler reads `store.GetScan` (404 on miss) and `store.ListAssessmentsForScan` (empty slice → empty-state path)
- [x] 3.3 Build the view model: per framework (dictu first, then alphabetical), per dimension, per rationale row with rule lookup wired through
- [x] 3.4 Render via a new `assessment.tmpl` embedded alongside the existing templates
- [x] 3.5 CSS for dimension cards, expandable rationale details, and the empty-state block (extends the existing main.css)

## 4. Scan-detail link

- [x] 4.1 `scanView` gains `HasAssessment bool`, populated from `ListAssessmentsForScan`
- [x] 4.2 `scan.tmpl` renders an "Open assessment →" anchor only when `HasAssessment` is true

## 5. Tests

- [x] 5.1 Round-trip: seed a scan with two assessments (DICTU + EUCSF), open `/ui/scans/{id}/assessment`, assert status 200 and that both framework labels and at least one rationale row appear
- [x] 5.2 Empty-state test: scan with zero assessments returns 200 and shows the "wanderer assess" hint
- [x] 5.3 Retired-rule test: persist an assessment whose CriteriumID is not in `DefaultRules`, assert the row renders with "rule retired" prefix and the historical verdict survives
- [x] 5.4 Verify the existing `TestNoMutatingHandlersInPackage` static-analysis test still passes (no new mutation handlers were added)
- [x] 5.5 Scan-page link tests: assessment exists → link present; no assessment → no link

## 6. Docs + CHANGELOG

- [ ] 6.1 Update `docs/architecture.md` "Read-only operator UI" section to mention the new Analysis page route
- [ ] 6.2 Update `docs/operator.md` (or create one) with screenshots / curl examples of the three UI pages and the assessment one
- [x] 6.3 CHANGELOG entry under `### Added` (assessment UI page) and `### Changed (breaking)` (Rule.Rationale becomes required)

## Notes

- The architecture-doc + operator-doc walkthrough updates (6.1, 6.2)
  are paired with `add-posture-dashboard`'s docs work, since both
  proposals touch the same UI walkthrough section. They land
  together in the dashboard change rather than splitting the
  walkthrough across two commits with intermediate inconsistency.
  Tracked as a follow-up scoped to that change.
- Rule.Rationale is intentionally distinct from Description:
  Description is a single-sentence summary the markdown report
  uses; Rationale is a paragraph the UI surfaces in an
  `<details>` block. The TestEveryRuleHasRationale test fails the
  build when the two are equal so authors do not paper over
  Rationale by copy-pasting the Description.
- Side-by-side framework rendering uses CSS grid; on narrow
  viewports the columns stack. No JavaScript anywhere — the
  expandable detail uses `<details><summary>` so the read-only
  static-analysis contract holds.
