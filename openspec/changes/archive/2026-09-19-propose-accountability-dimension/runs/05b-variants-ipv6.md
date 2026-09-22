# Habitat run 05b — variants: IPv6-eerlijkheid (tasks 6.4–6.5)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(design.md "Politeness (variants probe)", specs/scanner/spec.md
"Variants probe observes path convergence").

Both defects were found by running the merged probe against
rijksoverheid.nl from agent-lxc, which has a Tailscale ULA but no public
IPv6 route. Verified out of band: `curl -6 https://www.rijksoverheid.nl/`
fails there, yet the probe reported that path reachable.

## Scope — ONLY these tasks
- [x] 6.4 `internal/probe/variants/ipv6.go`: capability detection must
  mean "can reach the public internet over IPv6", not "an IPv6 address
  is configured". Unique-local (`fc00::/7`, so Tailscale's `fd7a::/16`)
  and CGNAT-style addresses do not count. Do one short-timeout dial to
  decide it, once per scan, injectable so tests can drive both cases.
  When the scanner has no IPv6, the four v6 paths are `not_tested` with
  reason `scanner_no_ipv6` — never `unreachable`.
- [x] 6.5 The "stop on an already-verified origin" optimisation SHALL
  NOT cross address families: a path may only be reported `reachable`
  over the family it was actually dialed on. Keep the optimisation
  within one family. Regression test: a stub where v4 succeeds and v6
  always fails must yield four reachable v4 paths and four v6 paths that
  are NOT reachable.

Tests use httptest / stub dialers, never the live network.

Done = these two checkboxes checked, and 6.4–6.5 ticked in tasks.md.

## Out of scope
Rules, copy, UI, other probes.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` green. Budget is $4.
