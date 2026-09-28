# Habitat run 01 — landwissel per adres (tasks 1.1–1.2)

Contract: `openspec/changes/2026-09-28-drift-land-per-adres/` (proposal: the
measurement and the cause; specs/scheduling/spec.md: three scenarios).

## Where
`internal/drift/rules.go`, `ipCountryChanged` (~line 232). `ip.asn` findings
carry `address`, `asn`, `country`, `organisation` in `Attributes`; `Subject` is
the host name. Tests: `internal/drift/rules_test.go` (`TestIPCountryChanged`,
`makeScan`).

## Decisions already made
1. Key = (subject, address). Emit when the same key has a different non-empty
   country. Findings without an `address` attribute: fall back to today's
   host-level behaviour only if the host has exactly one `ip.asn` finding in
   both scans; otherwise skip (do not guess).
2. Add `address` to the emitted attributes next to `prev_country` /
   `curr_country`.
3. Nothing else in the drift engine changes.

## Tests
Use the real shape from prod for the first scenario: six addresses for
`lennox.ns.cloudflare.com` — 108.162.195.214 CA, 162.159.44.214 US,
172.64.35.214 CA, 2606:4700:58::a29f:2cd6 CA, 2803:f800:50::6ca2:c3d6 CR,
2a06:98c1:50::ac40:23d6 GB — identical in both scans. Expect no
country_changed and one drift.no_changes via `Diff`. Check it red once against
the old code; say so. CHANGELOG `### Fixed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-28-drift-land-per-adres --strict` green. Budget $3.
