# Proposal: host-side scoring — make agent findings actually score

> **Status:** Implementation. Mark approved on 2026-05-10
> ("allebei, playwrights eerst" — host scoring is the second
> half). Resolved decisions on the four open questions:
>
> 1. **Dimension:** reuse `DimensionTechnologie`. No schema
>    change; the agent already tags inventory findings with
>    operationeel-like hints, and a new dimension would mean
>    a wider migration for marginal modelling gain.
> 2. **Match list shape:** YAML embedded via `go:embed` at
>    `internal/assessor/host_telemetry.yaml`. Operator-visible
>    file, reviewable in one place, mirrors the egress
>    probe's `vendors.yaml` pattern.
> 3. **Severity calibration:** kept simple for the first wave.
>    Any single hit on the known-telemetry list →
>    `afhankelijk`. No threshold tuning until real-world data
>    arrives. The eu-package-origin rule is dropped from this
>    wave (agent doesn't emit vendor metadata yet — see
>    "Scope tightening" below).
> 4. **File layout:** flat in `internal/assessor/wand/` and
>    `internal/assessor/eucsf/`. No subpackage.

## Intent

Mark asked on 2026-05-10: *"dit is enkel nog van buiten naar
binnen. hoe staat het met binnen → buiten?"* The honest answer:
the agent collects 2021 findings on this dev host (1790 RPM
packages, 231 systemd units) but **none of them are scored**.
Every existing rule in both rule packs looks at perimeter
finding ProbeIDs (`tls.issuer`, `whois.registrant`, `ip.asn`,
`http.third_party`). When the assessor runs against an
agent-host scan, every rule returns `onbekend` because the
shapes don't match.

The agent modus is complete on the *collection* side and
empty on the *scoring* side. This change adds the first wave
of host-side rules so an agent scan actually produces a
verdict on Dashboard + Analysis.

## Scope

**In scope (the first wave, post-tightening):**

Inventory observation confirms the agent emits two finding
families:

- `inventory.packages.rpm` (and `.dpkg`) — one finding per
  installed package. Attributes today: `arch`, `version`. **No
  `vendor` field** — the inspector does not extract RPM's
  `Vendor:` tag yet, so the proposed `eu_package_origin` rule
  is dropped from the first wave. Adding the field is a
  separate (small) agent change.
- `inventory.systemd.service` — one finding per active unit.

Three rules in this wave (down from the original five):

### wand.host.no_us_telemetry_packages

Flag installed packages whose name matches a known
US-hosted telemetry / observability agent
(`datadog-agent`, `newrelic-*`, `amazon-cloudwatch-agent`,
`amazon-ssm-agent`, `google-cloud-ops-agent`,
`azure-monitor-agent`, `omsagent`, `splunk*`,
`splunkforwarder`, `nrinfragent`). Soeverein when zero match;
afhankelijk when one or more match. Distro-shipped open-source
agents (`collectd`) are not on the list — they're not US-tied.

### wand.host.no_us_telemetry_services

Same match list applied to `inventory.systemd.service`
findings — flags active units that *run* a known
US-telemetry agent even if it was installed from a tarball or
container. Re-uses the same YAML so the two rules stay
synchronised.

### eucsf.sov5.host_no_us_telemetry

EUCSF analogue covering both packages and services in one
rule (the SEAL framework rolls supply-chain / vendor
exposure into a single observation). Reuses the YAML list.

**Out of scope (deferred to a future wave):**

- `wand.host.eu_package_origin` — requires the RPM/DPKG
  inspectors to emit a `vendor` attribute, which they don't
  today. Adding it is a small agent change; the rule lands
  after.
- `wand.host.no_us_egress_targets` — requires the agent's
  egress probe to be enabled and to emit findings. The demo
  fixture's agent run had egress disabled (no need to scan
  config files / proc env for credentials in the smoke
  test). Rule lands when egress data is on tap.
- Container image scoring (Docker inspector). The data is
  there (`inventory.docker.container`); rules cost a bigger
  inventory of known-US registries (`gcr.io`, `quay.io`'s
  current owner, etc.) and that's its own scope.
- Per-process / per-PID rules from the flow probe. The flow
  probe lands findings under `egress.flow.*`; rules that
  evaluate them are a follow-up.
- nextcloud inspector findings. Empty package, no rules yet.
- Drift findings as a rule source (drift is its own modus;
  scoring drift is a separate question).

## Resolved decisions (see status block at top)

See the four resolved decisions in the status block at the
top of this proposal. The full options + pros/cons are
preserved below as the design record.

## Open questions (resolved — preserved as design record)

1. **New dimension or reuse `technologie`?** The agent's
   inventory + egress findings already carry
   `dimension_hint: technologie` by default. Adding a new
   `DimensionHost` is cleaner conceptually but means a
   schema-level constant change. Reusing `technologie` keeps
   the existing dashboard split (where eucsf.sov4 already
   speaks to technology) but conflates "what the host runs"
   with "what the perimeter exposes". Recommendation: reuse
   `technologie` for now; consider a split later when the
   rule set is larger and the UI starts to feel crowded.

2. **Hard list or YAML?** The "known US telemetry vendors"
   list is the heart of these rules. Same pattern as the
   egress probe's `vendors.yaml` — embed via `go:embed` so
   the binary builds without external files but an operator
   can override at runtime with a `WANDERER_HOST_VENDORS`
   path. Recommendation: YAML embedded; mirrors the existing
   classifier.

3. **Severity calibration?** What counts as "afhankelijk" for
   `eu_package_origin`? 70% is a starting threshold; with
   real-world data on multiple hosts the threshold may need
   tuning. Recommendation: ship at 70% and revisit after a
   month of operator feedback.

4. **Where do the rules live?** Inside the existing
   `internal/assessor/wand/` package or a new
   `internal/assessor/wand/host/` sub-package? Recommendation:
   stay flat in `internal/assessor/wand/`; the per-rule file
   shape is small and the package's `DefaultRules()` already
   lists every rule explicitly.

## Wand / EUCSF dimensions informed

- **Technologie** (primary): what software the host runs
  becomes a sovereignty signal alongside what the perimeter
  exposes.
- Indirectly **Operationeel** for the systemd-side rules.

## Passive / active boundary

Pure assess-time logic. Reads existing Findings the agent
already lands. No new probes, no new network calls, no schema
change beyond the (possible) new dimension constant.

## Parallel-safe

Touches `internal/assessor/wand/`, `internal/assessor/eucsf/`,
optionally `pkg/models/finding.go` (the dimension constant),
plus tests. No UI change required — the agent's host scan
already shows up on Dashboard / Analysis once it has a
non-onbekend Assessment.
