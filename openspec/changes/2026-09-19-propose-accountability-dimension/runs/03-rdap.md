# Habitat run 03 — RDAP (tasks 4.1–4.3)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(design.md "External systems and failure modes", specs/scanner/spec.md
requirements "whois probe emits identity, reseller, status, and expiry
findings" and "Scanner looks up NS holder transparency").

## Scope — ONLY these tasks
- [x] 4.1 `internal/probe/whois`: parse the RDAP response it already
  fetches into four additional findings — `whois.registrant_identity`
  (name, kind, privacy-proxy flag), `whois.reseller`,
  `whois.status`, `whois.expiry` (the expiration event date, or an
  explicit `absent` when the registry publishes none). Entities are
  parsed RECURSIVELY (`entities[].entities[]`): a reseller commonly sits
  under the registrar entity. Redacted or missing fields are emitted as
  explicit redacted/absent values, never silently omitted. An RFC 9537
  `redacted` array, when present, is recorded as evidence — it does not
  decide anything by itself (the TLD list in run 06 does).
- [x] 4.2 Fixtures: (a) the live capture at
  `openspec/changes/2026-09-19-propose-accountability-dimension/fixtures/rdap-rijksoverheid.nl-20260919.json`
  — copy it into the probe's testdata: registrant "REDACTED FOR
  PRIVACY", registrar "Rijksoverheid", NO reseller at any nesting level,
  events registration/last changed only (so `whois.expiry` = absent);
  (b) a synthetic response with a reseller nested under the registrar.
- [x] 4.3 `internal/scanner`: for each unique registrable domain among
  the target's `dns.ns` hosts, one RDAP lookup → `whois.ns_holder`
  (registrant present / proxied / absent). Cache per scan: N nameservers
  under one provider cost ONE lookup. A failure emits
  `whois.ns_holder.unavailable` for that domain and the scan continues.
  Fixture: two providers / three nameservers → exactly two lookups; and
  a TLD without RDAP (404 from the bootstrap) → unavailable, scan
  continues.

No assessor rules in this run — the findings are observations only.
Tests are table-driven and use httptest, never the live network.

Done = these three checkboxes checked, and 4.1–4.3 ticked in tasks.md.

## Out of scope for this run
No rules, no YAML lists, no soa/securitytxt/variants probes, no UI.
Follow-up 7.4 is not yours.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/, see openspec/config.yaml);
`openspec validate 2026-09-19-propose-accountability-dimension --strict`
green. Budget is $5 — leave room to write your run report.
