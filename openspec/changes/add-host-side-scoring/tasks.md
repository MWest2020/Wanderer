# Tasks: host-side scoring (proposal pending sign-off)

> Every task is a design checkpoint until Mark approves the
> rule set and severity calls. The implementation PR opens
> after sign-off.

## 1. Open questions to resolve

- [ ] 1.1 New `DimensionHost` or reuse `DimensionTechnologie`?
  Recommendation: reuse for now.
- [ ] 1.2 Hard-coded match list or YAML? Recommendation: YAML
  embedded via `go:embed`, mirroring the egress probe's
  `vendors.yaml`.
- [ ] 1.3 Severity thresholds (70% / 90% etc.)? Recommendation:
  ship at the listed values, revisit after a month.
- [ ] 1.4 Rule file layout — flat vs `host/` subpkg?
  Recommendation: flat in `internal/assessor/wand/`.

## 2. Sign-off checkpoints

- [ ] 2.1 Mark approves the rule set scope (the five wand
  rules + eucsf analogues)
- [ ] 2.2 Mark approves the dimension call (reuse
  `technologie`)
- [ ] 2.3 Mark approves the match-list shape (YAML)

## 3. Pre-implementation inventory

- [ ] 3.1 Walk the agent's existing inspectors to confirm the
  finding shapes the rules will read against
- [ ] 3.2 Confirm the rule packs' existing `DefaultRules()`
  doesn't need restructuring for the new entries

## 4. Open the implementation change

- [ ] 4.1 When 1.x + 2.x are resolved, scaffold
  `add-host-side-rules` and start coding
