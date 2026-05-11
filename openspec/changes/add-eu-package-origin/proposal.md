# Proposal: eu_package_origin — score package vendor jurisdiction

> **Status:** Implementation. Decisions locked auto-mode +
> "boring + auditable + verkoopbaar" framing.

## Resolved decisions

1. **Agent change is the minimum diff**: RPM inspector adds
   `%{VENDOR}` to its query format string; DPKG inspector adds
   `${Maintainer}`. Both emit the raw value as a new Finding
   attribute (`vendor` for rpm, `maintainer` for dpkg). No
   parsing on the agent side — the rule does the
   classification.
2. **Match list is YAML, embedded via `go:embed`** — same
   pattern as host_telemetry.yaml and
   container_registries.yaml.
3. **Binary verdict, not threshold-based.** Any single
   US-vendored package → afhankelijk. Boring + auditable;
   sidesteps the "what percentage is the right cutoff"
   conversation.
4. **DPKG maintainer parsing**: extract the email domain from
   `"Name <user@host>"`. Domain → vendor lookup via the same
   YAML. Bare names without an email fall through as
   `other` (no jurisdiction call).

## Intent

Memory has called `eu_package_origin` a known scope-tightened
gap since add-host-side-scoring landed:

> wand.host.eu_package_origin — requires the RPM/DPKG
> inspectors to emit a vendor attribute, which they don't
> today. Adding it is a small agent change; the rule lands
> after.

Today the agent emits one inventory.packages.* Finding per
installed package with only `name`, `version`, `arch`. The
rule pack has no way to tell whether the package comes from
Red Hat (US) or openSUSE (Germany) or Debian (community
but SPI-incorporated US). This change closes the gap.

## Scope

**In scope:**

- **Agent change** (small):
  - `internal/probe/inventory/packages/rpm.go` — query format
    gains `%{VENDOR}`; parser stores it on the Finding's
    `vendor` attribute.
  - `internal/probe/inventory/packages/dpkg.go` — query format
    gains `${Maintainer}`; parser stores it on the Finding's
    `maintainer` attribute. The status check moves to a
    separate field to preserve maintainer's spacing.
- **Assessor change**:
  - New `internal/assessor/package_vendors.yaml` listing
    known package vendors mapped to `{ jurisdiction, parent_org }`.
    Buckets: `eu`, `us`, `other`. Case-insensitive substring
    match against the RPM `vendor` value OR the email domain
    extracted from the DPKG `maintainer` value.
  - New `internal/assessor/package_vendors.go` loader +
    `ClassifyPackageVendor(rpmVendor, dpkgMaintainer)
    (PackageVendor, bool)`.
  - New rule `wand.host.eu_package_origin`:
    - Walks `inventory.packages.*` findings
    - Classifies each finding (rpm: by `vendor` attr; dpkg: by
      `maintainer` email domain)
    - afhankelijk on any US-vendored hit (with subject + vendor
      + parent_org in verdict, Finding IDs in Evidence)
    - soeverein when every classified package is EU-vendored,
      with negative-evidence sample
    - voldoende when no US hits AND not all are EU (mixed /
      unclassified — the rule can't make a confident call but
      no red flag either)
    - onbekend without findings

**Out of scope:**

- A separate distro-detection probe reading
  `/etc/os-release` to attribute the platform vendor. The
  per-package vendor signal is enough for the first wave; a
  whole-host attribution rule lands later if needed.
- Vendor parent-org tracking across acquisitions (Red Hat → IBM,
  SUSE owned by EQT, etc.). The YAML names the immediate
  parent for verdict readability; jurisdiction is what the
  rule scores against.
- EUCSF analogue. The wand rule is a per-package observation;
  EUCSF supply-chain is already covered by sov6 rules
  (no_us_hyperscaler, host_no_us_telemetry,
  nextcloud_supply_chain, container_supply_chain). Adding a
  fifth supply-chain rule would create overlap. The wand
  pack carries this signal alone.

## Risks

- **Most distros are US-tied.** Fedora Project = Red Hat (US).
  Almost every Fedora host will score afhankelijk on this
  rule. The verdict text names the vendor so an operator
  sees "this is correctly identified as US-tied", not
  "Wanderer is broken".
- **Maintainer field is noisy.** DPKG Maintainer is often a
  team email (`team+pg@tracker.debian.org`) — useful — but
  sometimes an individual. The rule classifies via the email
  domain; absent an email it falls through to `other`.
- **Vendor list drift.** SUSE / Canonical / Red Hat
  ownership shifts happen. The YAML is small enough to fix
  by hand; the rule's verdict will surface a wrong
  classification visibly.

## Parallel-safe

Touches `internal/probe/inventory/packages/`,
`internal/assessor/`, `internal/assessor/wand/`,
`internal/fixtures/`, `tests/playwright/specs/`, docs. No
schema change, no UI change. The agent change is a one-line
edit to two query format strings.
