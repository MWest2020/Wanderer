# Tasks: Email routing

## 1. Direction decisions (resolve with Mark before building)

- [x] 1.1 Q1 — RESOLVED (Mark, 2026-06-17): synthesise the aggregate in
  the **scanner** (post-`Related` enrichment), emitted as an observed
  Finding, so the wand rule stays a pure annotation.
- [x] 1.2 Q2 — RESOLVED (Mark, 2026-06-17): small curated **operator
  table** (~10 suffixes + ASN-org fallback), grown in-repo like the
  egress vendor list; no new dependency.
- [x] 1.3 Q3 — RESOLVED (Mark, 2026-06-17): **hosting country =
  observed fact**; operator HQ / jurisdiction nuance carried by the
  rule.

## 2. Implementation

- [x] 2.1 Operator naming: a suffix→operator table + helper
  (`internal/scanner/mailrouting.go`: `mailOperatorSuffixes`,
  `mailOperator`) maps an MX host (ASN org as fallback) to a
  recognisable operator. Unit-tested per entry; raw host + ASN-org
  retained in the route.
- [x] 2.2 Scanner synthesis: `synthesiseMailRouting` runs after pass 2
  (post-`Related`/`ip.asn`), correlates `dns.mx` × `ip.asn` into one
  observed `dns.mx_routing` Finding ("inbound mail for {domain} lands
  at {operator} ({country})"), preference-ordered routes, observation
  severity. No-MX/null-MX and no-GeoIP handled. Persisted + appended in
  `scanner.go`.
- [x] 2.3 `wand.juridisch.mx_vendor_jurisdiction` leads its verdict with
  the observed operator (`observedMailOperators`/`mailLandsAt`, tolerant
  of the JSON-reloaded attr shape); scoring unchanged.
- [x] 2.4 Rendered via the Sovereignty overview's **Mail** flow, which
  reads the now operator-led rule verdict (same surface the transit
  path uses; the raw `dns.mx_routing` Finding also shows in the
  findings-by-probe view).
- [x] 2.5 Tests: operator table (7 cases incl. label-boundary), scanner
  synthesis (operator+country, no-GeoIP, anycast, multi-operator join,
  no-MX, no-dns.mx), rule operator-led verdict. `go vet` + full suite
  green. Passive (no new network — reuses `dns.mx`/`ip.asn`; summary is
  observed data rendered through auto-escaping `html/template`).
  Live-smoked `wanderer scan google.com` → "lands at Google Workspace".
- [x] 2.6 docs/operator.md ("Mail routing" section) + CHANGELOG.

## 3. Wrap-up

- [x] 3.1 Commit + push.
- [x] 3.2 Archive (email-routing capability spec merged); tick
  `research-high-signal-observability` task 2.2.
