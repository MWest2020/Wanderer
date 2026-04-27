# Tasks: Externalised vendor list

## 1. Schema + loader

- [ ] 1.1 `internal/probe/egress/vendors.yaml` with current entries
- [ ] 1.2 `vendors.go` loader with `//go:embed`, env + flag override
- [ ] 1.3 Schema validation with named-key error messages

## 2. Classifier integration

- [ ] 2.1 Pass the loaded `Vendors` struct into the rule table
- [ ] 2.2 Existing tests continue to pass without changes

## 3. Tests

- [ ] 3.1 Embedded-default round-trip
- [ ] 3.2 Override-file path produces a different verdict
- [ ] 3.3 Malformed YAML rejected with a clear message

## 4. Docs + CHANGELOG

- [ ] 4.1 `docs/egress.md` "Customising the vendor list" section
- [ ] 4.2 CHANGELOG entry under `### Added`
