## 1. Rule.Rationale field

- [ ] 1.1 Add `Rationale string` to `assessor.Rule` in `internal/assessor/rule.go`; document it in the godoc above the struct
- [ ] 1.2 Populate `Rationale` on every rule in `internal/assessor/dictu/rules.go` (one paragraph each, plain language, focused on the consequence of the rule firing)
- [ ] 1.3 Populate `Rationale` on every rule in `internal/assessor/eucsf/rules.go`
- [ ] 1.4 Add `TestEveryRuleHasRationale` to `internal/assessor/dictu/rules_test.go` and `internal/assessor/eucsf/rules_test.go` that fails the build when any rule's `Rationale` is empty

## 2. Rule registry lookup

- [ ] 2.1 Add a small helper in `internal/ui` (or `internal/assessor`) that takes a `Framework` string and a `CriteriumID` and returns the live `assessor.Rule` (or a "rule retired" sentinel)
- [ ] 2.2 Test the helper with a known DICTU rule, a known EUCSF rule, and a CriteriumID that does not exist (retired-rule path)

## 3. Assessment page handler

- [ ] 3.1 Add `assessmentHandler(st, tmpl)` to `internal/ui/ui.go`, mounted at `r.Get("/scans/{id}/assessment", ...)`
- [ ] 3.2 Handler reads the scan via `store.GetScan` (404 on miss) and `store.ListAssessmentsForScan` (empty slice → "no assessment yet" path)
- [ ] 3.3 Build the view model: per framework, per dimension (sorted), per rationale row with rule lookup wired through
- [ ] 3.4 Render via a new `assessment.tmpl` embedded alongside the existing templates
- [ ] 3.5 CSS rules for dimension cards + score badges (extend the existing static stylesheet — same colour scheme as the targets index)

## 4. Scan-detail link

- [ ] 4.1 The `scanHandler` view model gains an `HasAssessment bool` flag, populated from `ListAssessmentsForScan`
- [ ] 4.2 The existing `scan.tmpl` template renders an "Open assessment" anchor when `HasAssessment` is true and nothing when false

## 5. Tests

- [ ] 5.1 Round-trip test in `internal/ui/ui_test.go`: seed a scan with two assessments (DICTU + EUCSF), open `/ui/scans/{id}/assessment`, assert status 200 and that both framework labels and at least one rationale row appear
- [ ] 5.2 Empty-state test: scan with zero assessments returns 200 and shows the "run wanderer assess" hint
- [ ] 5.3 Retired-rule test: persist an assessment whose CriteriumID is not in `DefaultRules`, assert the row renders with "rule retired" prefix
- [ ] 5.4 Verify the existing `internal/ui` static-analysis test still passes (no new mutation handlers were added)

## 6. Docs + CHANGELOG

- [ ] 6.1 Update `docs/architecture.md` "Read-only operator UI" section to mention the new Analysis page route
- [ ] 6.2 Update `docs/operator.md` (or create one) with screenshots / curl examples of the three UI pages and the assessment one
- [ ] 6.3 CHANGELOG entry under `### Added` (assessment UI page) and `### Changed (breaking)` (Rule.Rationale becomes required)
