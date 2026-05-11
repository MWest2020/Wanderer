# Tasks: host-side scoring

> Mark approved on 2026-05-10 ("allebei, playwrights eerst" —
> host scoring is the second half). The four open questions
> from the design pass are resolved in `proposal.md`'s status
> block. This file tracks the implementation steps.

## 1. Resolved design decisions (recorded)

- [x] 1.1 Reuse `DimensionTechnologie` — no new dimension constant.
- [x] 1.2 Match list ships as `internal/assessor/host_telemetry.yaml`
  embedded via `go:embed`, mirroring the egress probe's vendors.yaml.
- [x] 1.3 Severity stays simple: any single hit → `afhankelijk`,
  zero matches → `soeverein`, missing finding shape → `onbekend`.
  No threshold tuning until real-world data arrives.
- [x] 1.4 Rule files flat in `internal/assessor/wand/host_rules.go`
  and `internal/assessor/eucsf/host_rules.go` — no subpackage.

## 2. Scope tightening

- [x] 2.1 Drop `wand.host.eu_package_origin` from this wave —
  the RPM/DPKG inspectors do not extract the `Vendor:` tag yet.
  Adding the field is a separate (small) agent change.
- [x] 2.2 Drop `wand.host.no_us_egress_targets` — the demo
  fixture's agent run had egress disabled; rule lands when
  egress data is on tap.
- [x] 2.3 Defer container-image, per-PID flow, and nextcloud
  rules to follow-up waves.

## 3. Match list

- [x] 3.1 Author `internal/assessor/host_telemetry.yaml` with the
  US-tied agents (Datadog, New Relic, AWS CloudWatch, AWS SSM,
  Google Cloud Ops, Azure Monitor / OMS, Splunk, Dynatrace).
  Open-source agents like `collectd` stay off the list.
- [x] 3.2 Add `internal/assessor/host_telemetry.go` with
  `HostTelemetryMatch(subject)` (case-insensitive prefix) and
  `HostTelemetryEntries()` for tests + UI rule descriptions.
- [x] 3.3 Loader test (`host_telemetry_test.go`): YAML parses,
  every entry has prefix + vendor, match cases cover known
  vendors, case-insensitive variants, and the `collectd`
  negative case.

## 4. Wand rules

- [x] 4.1 `wand.host.no_us_telemetry_packages` — reads
  `inventory.packages.*` findings, returns `onbekend` when no
  package findings exist (agent did not run / inspector
  disabled), `soeverein` on a clean host, `afhankelijk` on any
  hit. Verdict names matched packages alphabetically and
  cites their vendor of record.
- [x] 4.2 `wand.host.no_us_telemetry_services` — same shape
  against `inventory.systemd.service`.
- [x] 4.3 Register both in `wand/rules.go` `DefaultRules()`.
- [x] 4.4 Unit tests in `wand/host_rules_test.go`: clean host →
  soeverein, hit → afhankelijk + verdict + evidence,
  deterministic alphabetical order across multiple hits, no
  findings → onbekend, only meta (`unavailable: true`) → onbekend,
  packages/services rules ignore the wrong probe family.

## 5. EUCSF rule

- [x] 5.1 `eucsf.sov5.host_no_us_telemetry` — single rule
  walking both `inventory.packages.*` and
  `inventory.systemd.service`. SEAL framing in verdict text.
- [x] 5.2 Register in `eucsf/rules.go` `DefaultRules()`.
- [x] 5.3 Unit tests in `eucsf/host_rules_test.go`: clean host →
  soeverein, package hit → afhankelijk, service hit →
  afhankelijk, both shapes missing → onbekend, deterministic
  ordering of mixed package/service hits.
- [x] 5.4 Update existing `TestDefaultRules_FiveRules` →
  `TestDefaultRules_HasSixRules` and assert the new rule ID is
  registered.

## 6. Verification

- [x] 6.1 `go build ./...` clean.
- [x] 6.2 `go test ./...` clean across the repo (not just the
  assessor packages).
- [x] 6.3 `openspec validate add-host-side-scoring --strict` passes.
- [x] 6.4 Playwright smoke: catalogue assertions + a deep-dive
  spec that asserts a non-onbekend row when an agent host is
  seeded. Manual seed flow documented in
  `tests/playwright/fixtures/agent-host.yaml`; the spec is
  gated (skips cleanly without seed) until the hermetic
  fixture loader follow-up lands.
- [x] 6.5 Manual smoke: ran `wanderer agent --once` against this
  laptop into `/tmp/wanderer-demo.db`, scored under wand +
  eucsf, technologie dimension flipped from all-onbekend to
  `soeverein` (`inspected 1790 packages` / `231 systemd units`,
  no US telemetry vendors). Surfaces on `/ui/reporting/wand/...`
  per-rule pages.

## 7. Wrap-up

- [ ] 7.1 Update `docs/architecture.md` rules-extension section
  to include the host-rule pattern (one short paragraph).
- [ ] 7.2 Commit + push (push needs explicit user OK per global rule).
- [ ] 7.3 Archive change under
  `openspec/changes/archive/<YYYY-MM-DD>-add-host-side-scoring/`.

## 8. Discovered during implementation

- [x] 8.1 The assessor engine forces verdicts without
  `Evidence` to `onbekend`. Both rule packs gained a
  `sampleEvidence`/`sampleInspectedEvidence` helper that cites
  up to 10 inspected-finding IDs on the soeverein branch.
  Verdict text now reads `"inspected N packages — no US-…"`
  so an operator sees the scale of negative evidence.
- [x] 8.2 The egress-side `eu_package_origin` follow-up needs
  the RPM/DPKG inspectors to emit a `vendor` attribute first;
  this is the natural follow-up wave once `Vendor:` tag
  extraction lands.
