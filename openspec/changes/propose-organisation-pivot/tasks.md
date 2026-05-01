# Tasks: Organisation pivot — design checkpoints (no code)

> This change ships the **proposal and design only**. Each task
> below is a design checkpoint Mark needs to confirm before an
> implementation PR is opened. Once confirmed, a sibling change
> (`add-organisation-pivot` or similar) carries the actual code.

## 1. Open questions to resolve

- [ ] 1.1 **Default-org migration**: magic `default` slug vs.
  prompt-at-upgrade. Recommendation: `default` slug; existing
  operators rename later if they want.
- [ ] 1.2 **Agent host modelling**: keep Target with Kind=host vs
  introduce separate `Host` entity. Recommendation: keep
  Target with Kind=host; rule packs assume Target shape today
  and the dashboard's external/internal split already works.
- [ ] 1.3 **Scan ergonomics**: explicit `--organisation` vs WHOIS
  inference. Recommendation: explicit only; inference is too
  flaky to build into the contract.
- [ ] 1.4 **Schedule migration**: existing scheduled scans
  attach to `default`; new schedules require an org field.
- [ ] 1.5 **Asset-discovery seed**: `wanderer org seed --from-amass`
  in v1 or v2? Recommendation: v2 — operator scripting suffices
  for v1.
- [ ] 1.6 **MCP exposure**: ship three new MCP methods
  (`org.list`, `org.show`, `org.targets`) in the same PR? Or
  follow-up? Recommendation: same PR — MCP needs to track the
  surface, separate PR risks drift.

## 2. Sign-off checkpoints

- [ ] 2.1 Mark approves the entity shape (slug rules, Name,
  Description, CreatedAt)
- [ ] 2.2 Mark approves the migration plan (steps 1–3, the
  expand-then-contract sequence)
- [ ] 2.3 Mark approves the CLI surface (`wanderer org add /
  list / show`, the YAML seed shape, no remove / rename in v1)
- [ ] 2.4 Mark approves the UI shape (`/ui/orgs/{slug}`, the
  instance-wide `/ui/` becoming a list of registered organisations
  + global rollup, Reporting query param)

## 3. Pre-implementation inventory

- [ ] 3.1 Walk every spot listed in design.md "Things to
  inventory before implementing" and confirm each is in the
  impl PR's scope
- [ ] 3.2 Verify no rule pack hard-codes "the only Target shape
  is a domain" — grep `internal/assessor/wand/`,
  `internal/assessor/eucsf/` for assumptions

## 4. Open the implementation change

- [ ] 4.1 When all of 1.x and 2.x are resolved, scaffold
  `add-organisation-pivot` (or rename of this change with the
  same number) and start coding
