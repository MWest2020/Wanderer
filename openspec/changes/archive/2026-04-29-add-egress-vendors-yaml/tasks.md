# Tasks: Externalised vendor list

## 1. Schema + loader

- [x] 1.1 `internal/probe/egress/vendors.yaml` with current entries
- [x] 1.2 `vendors.go` loader with `//go:embed`, env + flag override
- [x] 1.3 Schema validation with named-key error messages

## 2. Classifier integration

- [x] 2.1 Pass the loaded `Vendors` struct into the rule table
- [x] 2.2 Existing tests continue to pass without changes

## 3. Tests

- [x] 3.1 Embedded-default round-trip
- [x] 3.2 Override-file path produces a different verdict
- [x] 3.3 Malformed YAML rejected with a clear message

## 4. Docs + CHANGELOG

- [x] 4.1 `docs/egress.md` "Customising the vendor list" section
- [x] 4.2 CHANGELOG entry under `### Added`

## Notes

- The proposal's design.md sketch included a
  `us_hyperscaler_organisation_substrings` block. That list is owned
  by the assessor (`internal/assessor/{dictu,eucsf}/rules.go`), not
  by the egress classifier, so externalising it is out of scope for
  this change. Adding it to vendors.yaml without wiring it into the
  assessor would be dead weight — moved to a follow-up if the
  assessor ever needs YAML-driven hyperscaler rules.
- The loaded `Vendors` table also externalises the `*_key_regex`
  fallbacks (previously hard-coded in `classify.go`). This was a
  natural extension of the proposal: an operator running self-hosted
  Loki at an internal hostname can now add the matching key regex
  without rebuilding.
- `Configure` is intentionally not concurrency-safe — agents call it
  once at startup before scans begin.
