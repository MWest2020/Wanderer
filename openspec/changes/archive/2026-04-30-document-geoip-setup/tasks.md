## 1. Documentation

- [x] 1.1 Add a "GeoLite2 setup" section to `docs/operator.md` covering: free-tier MaxMind account, license key, `--geoip` / `--geoip-country` flags + env equivalents, systemd-timer-based `geoipupdate` example, crontab alternative, opt-out for offline labs
- [x] 1.2 `docs/architecture.md` "External systems and their failure modes" table — link the GeoLite2 row to the operator doc section
- [x] 1.3 `docs/tutorial.md` first-run section — add a "before you scan" callout pointing at the GeoLite2 setup so a new contributor lands on the populated path the first time

## 2. CLI startup warning

- [x] 2.1 Add a small helper in `cmd/wanderer/` that resolves the GeoLite2 ASN path from flag → env → empty, and reports whether the opt-out is set
- [x] 2.2 Wire the helper into `cmd/wanderer/scan.go` and `cmd/wanderer/serve.go` to emit one warning line at startup when ASN path is empty and opt-out is not set; warning text references `--geoip` and the docs path
- [x] 2.3 Add `--no-geoip` boolean flag and the `WANDERER_GEOIP_OPTIONAL=1` env-var fallback to both commands
- [x] 2.4 Tests: `cmd/wanderer/geoip_test.go` asserts the warning fires on default invocation and is silenced by `--no-geoip` / the env var; existing scan-completion behaviour unchanged

## 3. Stub mmdb for tests

- [x] 3.1 Add `scripts/geoip-stub.sh` (driver) + `scripts/geoip-stub/main.go` (build-tag ignore'd) that uses `github.com/maxmind/mmdbwriter` to produce a minimal empty-but-valid mmdb at the path passed as `$1`
- [x] 3.2 Document the script's usage in `docs/operator.md` under the GeoLite2 setup section
- [x] 3.3 Manual smoke: `./scripts/geoip-stub.sh /tmp/stub.mmdb && /tmp/wanderer scan example.com --geoip /tmp/stub.mmdb` opens the stub and runs the IP probe in its populated-but-empty mode

## 4. Decision: vendor a stub vs scripted stub

- [x] 4.1 Decide whether `mmdbwriter` is acceptable as a build-time-only dep, or whether to check in a tiny pre-built stub mmdb under `testdata/` with a README. **Chose scripted stub**: the script + a `//go:build ignore`'d main package keeps the production build clean (no production package imports mmdbwriter), and the script remains reproducible

## 5. CHANGELOG

- [x] 5.1 CHANGELOG entry under `### Added` (startup warning, opt-out flag, stub-builder script, docs)

## Notes

- The mmdbwriter dep enters go.mod (and brings two transitive deps:
  `oschwald/maxminddb-golang/v2` and `go4.org/netipx`). None of
  these are imported by any non-`//go:build ignore` Go file in the
  module, so the production binary is unchanged. Verified via
  `go build ./...` producing the same binary size before and after.
- The warning is emitted via `fmt.Fprintln(os.Stderr, "warning: …")`
  rather than `slog.Warn` so the output is independent of the
  `--json-logs` toggle and the spec scenario "stderr contains a
  line beginning with `warning:`" holds in both modes.
- An integration check would normally exercise the full scan path
  with a real GeoLite2 file, but that requires a license key.
  `scripts/geoip-stub.sh` covers the populated-but-empty path; a
  future operator-driven smoke test can run the populated-with-data
  path on a host that has a real `GeoLite2-ASN.mmdb`.
