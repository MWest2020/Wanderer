# Tasks: YAML config file for `wanderer serve`

## 1. New package internal/serveconfig

- [x] 1.1 `internal/serveconfig/config.go` — Config struct,
  Load, Parse (strict), Validate
- [x] 1.2 `internal/serveconfig/resolve.go` — resolveString,
  resolveBool, resolveDuration helpers
- [x] 1.3 `internal/serveconfig/config_test.go` — strict-parse,
  empty-file, typo-rejected scenarios
- [x] 1.4 `internal/serveconfig/resolve_test.go` — every layer of
  every resolver in isolation

## 2. Wire into cmd/wanderer/serve.go

- [x] 2.1 Add `--config` flag and `WANDERER_CONFIG` env var
- [x] 2.2 Parse flags first, capture set flags via `flag.Visit`
- [x] 2.3 Load YAML if `--config` non-empty; fail-fast on error
- [x] 2.4 Replace each existing `envOr(...)` flag default with
  `resolveX(setFlags, flagName, ...)` after parsing
- [x] 2.5 Confirm no behaviour change when `--config` is unset
  (existing serve_test.go should still pass)

## 3. Docs + changelog

- [x] 3.1 `docs/operator.md` — new "Config file" section showing
  serve.yaml schema, a minimal example, and a sample systemd
  unit using `--config`
- [x] 3.2 `CHANGELOG.md` entry under `### Added`

## 4. Spec

- [x] 4.1 Spec delta in `openspec/specs/scanner/spec.md` capturing
  the precedence order and the strict-parse contract
