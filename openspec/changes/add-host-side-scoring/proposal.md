# Proposal: host-side scoring — make agent findings actually score

> **Status:** Design proposal. The implementation lands after
> Mark signs off on the rule set and severity calls.

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

**In scope (the first wave):**

The agent currently emits two finding families:

- `inventory.packages.rpm` (and `.dpkg`) — one finding per
  installed package with name, version, vendor
- `inventory.systemd.service` — one finding per active service
  with unit name, exec start, environment redactions

The first wave of rules pulls a handful of high-signal checks
out of those:

### wand.host.no_us_telemetry_packages

Flag installed packages whose vendor or name matches a known
US-hosted telemetry / observability vendor. Soeverein when no
such packages are installed; voldoende when one or two are
present (operationally common — `chronyd` etc. are fine);
afhankelijk when several. The match list is small and pinned
in YAML (e.g. `datadog-agent`, `newrelic-*`, etc.).

### wand.host.eu_package_origin

Score the **vendor distribution** of installed packages: what
percentage of packages declare an EEA-region vendor field?
RHEL / AlmaLinux / Debian's `Vendor:` field is reliable for
distro-managed packages. Soeverein at >90% EU-vendor;
voldoende 70–90%; afhankelijk below 70%.

### wand.host.no_us_telemetry_services

Same idea as no_us_telemetry_packages but reading from
`inventory.systemd.service` ExecStart paths and unit names.
Specifically flags units that start known telemetry daemons
even if installed from a non-vendored binary.

### eucsf.sov-host.no_us_telemetry

Identical match-table approach but framed for the SEAL
sov-host dimension. Reuses the underlying classifier list so
the two packs stay in lock-step.

### eucsf.sov-host.eu_kernel

Reads `inventory.systemd.service` for the running kernel
release (a separate Finding the existing inspectors could emit
trivially) and the build vendor. Soeverein on a distro-built
mainline kernel; voldoende on a third-party module; onbekend
when the inspector did not emit a kernel finding.

**Also in scope:**

- A new dimension constant `models.DimensionHost` (or
  reuse `DimensionTechnologie` — see Open questions). The
  agent's existing findings already use
  `DimensionTechnologie`; if we add a new dimension, the
  agent's `dimension_hint` field needs updating.
- The egress static scanner's findings (`egress.s3`,
  `egress.slack`, etc.) get **one** wand rule:
  `wand.host.no_us_egress_targets` — same vendor classifier
  the perimeter rule already uses, just consuming the
  inventory-side findings. EUCSF analogue.

**Out of scope (deferred to a future wave):**

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

## Open questions

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
