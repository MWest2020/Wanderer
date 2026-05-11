# Tasks: Nextcloud as a target

> Mark picked direction (1) on 2026-05-11. Tasks below are
> the implementation plan. Q2-Q4 decisions recorded in the
> proposal's status block.

## 1. Resolved decisions (recorded)

- [x] 1.1 Direction = (1) Nextcloud as a sovereignty target.
- [x] 1.2 Extend `inventory.nextcloud.*` ProbeID family.
- [x] 1.3 Agent-side only; perimeter angle deferred.
- [x] 1.4 OIDC: probe `user_oidc` first; graceful unavailable
  for `oidc_login` / `social_login`.

## 2. Probe additions

- [ ] 2.1 `inventory.nextcloud.version` Finding — one row per
  scan, populated from `occ status --output=json`.
- [ ] 2.2 `inventory.nextcloud.trusted_domain` Findings — one
  row per entry from
  `occ config:list system trusted_domains`.
- [ ] 2.3 `inventory.nextcloud.objectstore` Findings — one
  row per S3-style backend from
  `occ config:list system objectstore`, carrying `endpoint`
  + `bucket` + (geoip-enriched) `asn` / `country`.
- [ ] 2.4 `inventory.nextcloud.oidc_provider` Findings — one
  row per provider from `occ user_oidc:provider list`,
  carrying `issuer_url`. Fallback emits
  `inventory.nextcloud.oidc.unavailable` with the
  discovered alternative app name when `user_oidc` is
  absent.
- [ ] 2.5 Per-probe unit tests covering happy path + the
  `occ` version contract (parser declares which Nextcloud
  majors it supports).

## 3. Rules

- [ ] 3.1 `wand.nextcloud.objectstore_eu` — afhankelijk on
  any objectstore Finding whose `country` is non-EEA.
  Soeverein when zero non-EEA backends are configured;
  onbekend without objectstore findings.
- [ ] 3.2 `wand.nextcloud.oidc_provider_eu` — same shape on
  `inventory.nextcloud.oidc_provider` (jurisdiction call on
  the issuer URL via the existing `tls.issuer` machinery).
- [ ] 3.3 `eucsf.sov6.nextcloud_supply_chain` — single SEAL
  rule covering both probe families.
- [ ] 3.4 Each rule cites the offending Finding ID(s) as
  Evidence on the afhankelijk branch and a sampled set on
  the soeverein branch (negative-evidence pattern from
  add-host-side-scoring).

## 4. Wiring

- [ ] 4.1 Register the rules in `wand/rules.go` +
  `eucsf/rules.go` `DefaultRules()`.
- [ ] 4.2 Extend the agent fixture scenario
  (`internal/fixtures/agent_host.go`) with a synthetic
  Nextcloud sample (one US objectstore backend + one
  US issuer URL) so Playwright can pin the rule rows.
- [ ] 4.3 Playwright spec
  `tests/playwright/specs/nextcloud-as-target.spec.ts`
  asserts the three new rules surface on
  `/ui/reporting/...` with non-onbekend rows.

## 5. Verification

- [ ] 5.1 `go test ./internal/probe/inventory/nextcloud/...`
  green.
- [ ] 5.2 `go test ./internal/assessor/...` green.
- [ ] 5.3 `go test ./...` clean across the repo.
- [ ] 5.4 `make playwright` clean (uses the hermetic fixtures
  from add-playwright-fixtures).
- [ ] 5.5 `openspec validate add-nextcloud-as-target
  --strict` passes.

## 6. Wrap-up

- [ ] 6.1 Update `docs/operator.md` with the Nextcloud
  inspector setup (which `occ` commands the agent runs,
  which apps are required, what each rule scores).
- [ ] 6.2 Commit + push (push needs explicit user OK).
- [ ] 6.3 Archive under
  `openspec/changes/archive/<YYYY-MM-DD>-add-nextcloud-as-target/`.
