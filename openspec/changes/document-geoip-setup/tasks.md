## 1. Documentation

- [ ] 1.1 Add a "GeoLite2 setup" section to `docs/operator.md` covering: free-tier MaxMind account, license key, `--geoip` / `--geoip-country` flags + env equivalents, systemd-timer-based `geoipupdate` example, crontab alternative, opt-out for offline labs
- [ ] 1.2 `docs/architecture.md` "External systems and their failure modes" table — link the GeoLite2 row to the operator doc section
- [ ] 1.3 `docs/tutorial.md` first-run section — add a "before you scan" callout pointing at the GeoLite2 setup so a new contributor lands on the populated path the first time

## 2. CLI startup warning

- [ ] 2.1 Add a small helper in `cmd/wanderer/` that resolves the GeoLite2 ASN path from flag → env → empty, and reports whether the opt-out is set
- [ ] 2.2 Wire the helper into `cmd/wanderer/scan.go` and `cmd/wanderer/serve.go` to emit one `slog.Warn` line at startup when ASN path is empty and opt-out is not set; warning text references `--geoip` and the docs path
- [ ] 2.3 Add `--no-geoip` boolean flag and the `WANDERER_GEOIP_OPTIONAL=1` env-var fallback to both commands
- [ ] 2.4 Tests: `cmd/wanderer/scan_test.go` (or a new test file) asserts the warning fires on default invocation and is silenced by `--no-geoip` / the env var; existing scan-completion behaviour unchanged

## 3. Stub mmdb for tests

- [ ] 3.1 Add `scripts/geoip-stub.sh` that uses `github.com/maxmind/mmdbwriter` to produce a minimal empty-but-valid mmdb at the path passed as `$1`
- [ ] 3.2 Document the script's usage in `docs/operator.md` "Running tests" subsection
- [ ] 3.3 Update `internal/probe/ip` test setup so a CI run can opt into the stub-mmdb path (build tag, helper function, or environment variable — implementation choice)

## 4. Decision: vendor a stub vs scripted stub

- [ ] 4.1 Decide whether `mmdbwriter` is acceptable as a build-time-only dep, or whether to check in a tiny pre-built stub mmdb under `testdata/` with a README. Default to the script path; only fall back to vendoring if the dep is rejected in review

## 5. CHANGELOG

- [ ] 5.1 CHANGELOG entry under `### Added` (startup warning, opt-out flag, stub-builder script) and `### Changed` (operator docs explicitly cover GeoLite2 setup)
