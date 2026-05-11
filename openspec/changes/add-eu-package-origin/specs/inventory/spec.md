# Delta for inventory

## ADDED Requirements

### Requirement: Package inspectors emit vendor / maintainer attribute

The RPM inspector SHALL include the package's Vendor field in
the Finding's `vendor` attribute. The DPKG inspector SHALL
include the package's Maintainer field in the Finding's
`maintainer` attribute. Both attributes carry the raw upstream
value; classification happens in the assessor.

#### Scenario: RPM emits vendor

- **GIVEN** `rpm -qa --qf` output includes the line
  `bash 5.2.32-1.fc42 x86_64 Fedora Project`
- **WHEN** the agent inspects
- **THEN** the resulting `inventory.packages.rpm` Finding's
  Attributes include `vendor: "Fedora Project"`

#### Scenario: DPKG emits maintainer

- **GIVEN** `dpkg-query -W -f` output includes a line whose
  Maintainer is
  `Debian PostgreSQL Maintainers <team+pg@tracker.debian.org>`
- **WHEN** the agent inspects
- **THEN** the resulting `inventory.packages.dpkg` Finding's
  Attributes include `maintainer:
  "Debian PostgreSQL Maintainers <team+pg@tracker.debian.org>"`

---

### Requirement: wand.host.eu_package_origin classifies package vendor jurisdiction

The assessor SHALL register `wand.host.eu_package_origin`,
reading `inventory.packages.*` findings and classifying each
finding's vendor / maintainer against
`internal/assessor/package_vendors.yaml`. The rule SHALL emit:

- `afhankelijk` on any package classified to a US-tied vendor
  (Red Hat, Microsoft, Oracle, Canonical, etc.), with the
  matched packages cited in Verdict + Evidence
- `soeverein` when every classified package resolves to an
  EU-tied vendor (openSUSE / SUSE / etc.), with a negative-
  evidence sample
- `voldoende` when no US hits AND not every classified
  package is EU-tied (mixed / unclassified)
- `onbekend` without `inventory.packages.*` findings

#### Scenario: Fedora host scores afhankelijk

- **GIVEN** every `inventory.packages.rpm` Finding carries
  `vendor: "Fedora Project"`
- **WHEN** the assessor runs `wand.host.eu_package_origin`
- **THEN** the Score is `afhankelijk` and the Verdict names
  Fedora Project + Red Hat as the parent vendor

#### Scenario: openSUSE host scores soeverein

- **GIVEN** every `inventory.packages.rpm` Finding carries
  `vendor: "openSUSE"`
- **WHEN** the assessor runs `wand.host.eu_package_origin`
- **THEN** the Score is `soeverein`, the Verdict includes
  the inspected count, and Evidence cites a sample of
  Finding IDs

#### Scenario: Maintainer email domain classifies dpkg

- **GIVEN** a `inventory.packages.dpkg` Finding whose
  `maintainer` is
  `Debian Postgres <team+pg@tracker.debian.org>`
- **WHEN** the rule classifies the finding
- **THEN** the lookup uses `debian.org` as the key
