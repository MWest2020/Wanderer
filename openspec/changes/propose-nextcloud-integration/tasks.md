# Tasks: Nextcloud integration (direction-finding pending)

> Every task is a design checkpoint until Mark picks Q1
> (the direction). Implementation tasks are written assuming
> direction = (1) Nextcloud as a target; if a different
> direction wins, this tasks file is rewritten or a sibling
> proposal supersedes this one.

## 1. Direction decision

- [ ] 1.1 Mark picks Q1 (1 / 2 / 3 / 4 / combo). No
  implementation work starts until this is answered.

## 2. Scoping decisions (assumes direction = 1)

- [ ] 2.1 Resolve Q2 — extend `inventory.nextcloud.*` family
  vs introduce `config.nextcloud.*`. Recommendation:
  extend the existing family.
- [ ] 2.2 Resolve Q3 — agent-side only, or add a
  perimeter-side probe in a sibling change. Recommendation:
  agent-side only; sibling change for perimeter later.
- [ ] 2.3 Resolve Q4 — `user_oidc`-first with a graceful
  unavailable path for `oidc_login` / `social_login`.

## 3. Implementation — after sign-off

- [ ] 3.1 Add Nextcloud version probe
  (`occ status --output=json` → `inventory.nextcloud.version`).
- [ ] 3.2 Add trusted-domains probe
  (`occ config:list system trusted_domains`).
- [ ] 3.3 Add objectstore probe with geoip annotation.
- [ ] 3.4 Add OIDC-provider probe with the `user_oidc`
  fallback path.
- [ ] 3.5 Three rules: `wand.nextcloud.objectstore_eu`,
  `wand.nextcloud.oidc_provider_eu`,
  `eucsf.sov6.nextcloud_supply_chain`.
- [ ] 3.6 Unit tests for each probe + rule, plus a parser
  contract test that pins the supported Nextcloud major
  versions.
- [ ] 3.7 `docs/operator.md` walkthrough section for
  enabling the inspector with each probe family on.

## 4. Wrap-up

- [ ] 4.1 Playwright spec covering the Nextcloud-as-a-target
  rule rows (gated on the hermetic-fixtures change landing).
- [ ] 4.2 Commit + push (push needs explicit user OK).
- [ ] 4.3 Archive under
  `openspec/changes/archive/<YYYY-MM-DD>-add-nextcloud-as-target/`.
