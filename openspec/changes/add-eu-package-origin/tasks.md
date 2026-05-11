# Tasks: eu_package_origin

> Auto-mode straight-through.

## 1. Agent — RPM inspector

- [ ] 1.1 Change `realRpmQuery` to include `%{VENDOR}`.
- [ ] 1.2 Update `parseRpm` to extract the vendor field and
  store it on the Finding's `vendor` attribute.
- [ ] 1.3 Unit test the parser end-to-end with sample
  `rpm -qa --qf` output including Vendor.

## 2. Agent — DPKG inspector

- [ ] 2.1 Change `realDpkgQuery` to include `${Maintainer}`.
  Maintainer values can contain spaces; field order needs to
  be unambiguous.
- [ ] 2.2 Update `parseDpkg` to split off the maintainer field
  before splitting status. Store the raw maintainer string on
  the Finding's `maintainer` attribute.
- [ ] 2.3 Unit test the parser with realistic dpkg-query
  output (team emails + individual maintainers).

## 3. Vendor classifier

- [ ] 3.1 Author
  `internal/assessor/package_vendors.yaml` with the
  jurisdiction map.
- [ ] 3.2 Author `internal/assessor/package_vendors.go` with
  `ClassifyPackageVendor(rpmVendor, dpkgMaintainer)
  (PackageVendor, bool)`. Case-insensitive substring match
  for RPM; email-domain extraction + substring match for DPKG.
- [ ] 3.3 Loader + classifier tests cover the happy paths
  (Fedora, Red Hat, Microsoft, openSUSE, Debian, Ubuntu) +
  edge cases (empty vendor, bareword maintainer).

## 4. Rule

- [ ] 4.1 `wand.host.eu_package_origin` in
  `wand/host_rules.go` (lives alongside the existing
  telemetry rules). Reads `inventory.packages.*` findings.
- [ ] 4.2 Register in `DefaultRules()`.
- [ ] 4.3 Unit tests mirror the existing host-rule pattern:
  clean EU host → soeverein with evidence, US hit →
  afhankelijk, mixed → voldoende, no findings → onbekend.

## 5. Fixture + Playwright

- [ ] 5.1 Update `internal/fixtures/agent_host.go` so the
  seeded RPM packages carry a `vendor` attribute. Most
  packages claim "Fedora Project" so the rule fires
  afhankelijk on the seeded scan.
- [ ] 5.2 Add Playwright spec asserting the rule row shows
  afhankelijk with "Fedora Project" + Red Hat in the verdict.

## 6. Docs

- [ ] 6.1 `docs/operator.md` — packages-inspector section
  gains a note on the new vendor attribute + the rule.

## 7. Verification

- [ ] 7.1 `go test ./...` clean.
- [ ] 7.2 `make playwright` clean.
- [ ] 7.3 `openspec validate add-eu-package-origin --strict`.

## 8. Wrap-up

- [ ] 8.1 Commit.
- [ ] 8.2 Merge spec delta into inventory-probe + assessor specs.
- [ ] 8.3 Archive.
