# Tasks: Rules ignore meta Findings

## 1. Helper

- [ ] 1.1 Add `IsEvidenceLike(f models.Finding) bool` to `internal/assessor/rule.go`
- [ ] 1.2 Unit test the helper with one positive and three negative cases (error / no_answer / unavailable)

## 2. Rule fixes

- [ ] 2.1 `mxPresent` consults `IsEvidenceLike` before counting
- [ ] 2.2 Audit every other rule in `internal/assessor/dictu/rules.go` that does bare-ProbeID counting and apply the helper where it matters
- [ ] 2.3 Regression test: NXDOMAIN-style finding set produces Onbekend on `mx_present`

## 3. Docs

- [ ] 3.1 Document the meta-finding convention in `docs/findings.md`
- [ ] 3.2 CHANGELOG entry under `### Fixed`
