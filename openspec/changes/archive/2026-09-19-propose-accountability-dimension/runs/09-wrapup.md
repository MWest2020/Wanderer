# Habitat run 09 — docs + wrap-up (tasks 9.1–9.2)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`.

## Scope — ONLY these tasks
- [ ] 9.1 `docs/reference/assessor.md` and the findings reference: the
  new `accountability` dimension, the seven rules, the new findings
  (`whois.registrant_identity`, `whois.reseller`, `whois.status`,
  `whois.expiry`, `whois.ns_holder`, `dns.soa`, `http.securitytxt`,
  `http.variants`, `config.expected_registrant`) and the reason-code
  mechanism (class structural/gap, subject target/scanner). CHANGELOG
  entry under `[Unreleased]`.
- [ ] 9.2 `docs/explanation/`: a short note on the RDAP fields we wish
  existed (actor/escalation relationships, ISO2-prefixed subject
  identifiers) and why they are out of scope, plus the `.nl` passive
  ceiling — SIDN redacts registrant data and publishes no expiry, so
  two of five questions are structurally n.v.t. there and Wanderer does
  not work around it (no registrar APIs, no scraping, no KVK lookups).

Follow the docs contract already in this repo (front matter, the
diátaxis folders under `docs/`), and do not touch task 9.3 (the
attribution text) — Mark supplies that.

## Out of scope for this run
Task 9.3 (attribution wording) and 9.4 (archiving the change).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` green. Budget is $4.
