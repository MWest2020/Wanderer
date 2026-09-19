# Habitat run 05 — variants probe (tasks 6.1–6.3)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(design.md "Politeness (variants probe)", specs/scanner/spec.md
"Variants probe observes path convergence").

## Scope — ONLY these tasks
- [x] 6.1 New variants probe (`internal/probe/variants`, wired into the
  scanner's probe list like the existing probes): the 8 paths
  apex/www × IPv4/IPv6 × http/https, redirect depth at most 5 per path,
  a HARD budget of 24 connections per target across all paths, no
  retries, inside the scan's global timeout. Emit `http.variants` with,
  per path: status (`reachable`, `unreachable`, `refused`,
  `not_followed_budget`, `not_tested`), the redirect chain, and the
  final origin. A chain MAY stop as soon as it reaches an origin already
  verified in this run. On total failure emit `http.variants.unavailable`.
- [x] 6.2 Every redirect hop passes the EXISTING SSRF guard
  (`internal/probe/ssrf.go`) — reuse it, do not write a second one. A
  refused hop is recorded as `refused` and never followed. Fixture: a
  redirect chain pointing at 10.0.0.5.
- [x] 6.3 When the scanner host has no IPv6 route, the four v6 paths are
  recorded as `not_tested` with reason `scanner_no_ipv6` (structural,
  subject scanner — see internal/assessor's reason table). A dead v6
  family on the target is only claimed when the scanner could have
  reached it. Detect the local capability once per scan, not per path,
  and make it injectable so a test can drive both cases.

No assessor rules in this run — findings are observations only. Tests
are table-driven and use httptest, never the live network; assert the
connection budget is actually enforced (a chain of 5 redirects per path
must not exceed 24 connections).

Done = these three checkboxes checked, and 6.1–6.3 ticked in tasks.md.

## Out of scope for this run
No rules, no YAML lists, no UI, no changes to the whois/soa/http probes
(runs 03 and 04). Follow-up 7.4 is not yours.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/, see openspec/config.yaml);
`openspec validate 2026-09-19-propose-accountability-dimension --strict`
green. Budget is $5 — leave room to write your run report.
