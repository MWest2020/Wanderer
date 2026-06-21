# Tasks: DNS hosting

## 1. Direction decisions (resolve with Mark before building)

- [x] 1.1 Q1 — where the aggregate is synthesised. Confirmed (Mark,
  2026-06-21): scanner synthesis step alongside `synthesiseMailRouting`,
  emitted as an observed Finding so the wand rule stays a pure annotation.
- [x] 1.2 Q2 — operator table size/location. Confirmed: curated NS
  suffixes + ASN-org fallback, grown in-repo; sharing the operator-suffix
  helper (`operatorBySuffix`) with `mailrouting.go`.
- [x] 1.3 Q3 — hosting country = observed fact; operator HQ /
  jurisdiction nuance carried by the rule.

## 2. Implementation

- [x] 2.1 Operator naming: a suffix→operator table + helper
  (NS-host, ASN-org fallback) mapping an NS host to a recognisable
  managed-DNS operator. Unit-tested per entry; raw host + ASN-org
  retained in the route. (`dnshosting.go`: `dnsOperatorSuffixes` +
  `dnsOperator`; shared `operatorBySuffix`/`operatorSuffix` with
  `mailrouting.go`.)
- [x] 2.2 Scanner synthesis: a `synthesiseDNSHosting` step running after
  pass 2 (post-`Related`/`ip.asn`), correlating `dns.ns` × `ip.asn` into
  one observed `dns.ns_hosting` Finding ("DNS for {domain} is run by
  {operator} ({country})"), observation severity. No-NS and no-GeoIP and
  anycast handled. Persisted + appended in `scanner.go`.
- [x] 2.3 `wand.juridisch.ns_vendor_jurisdiction` leads its verdict with
  the observed operator (tolerant of the JSON-reloaded attr shape);
  scoring unchanged. (`observedDNSOperators` + `joinAnd`.)
- [x] 2.4 Rendering: the Sovereignty overview's **DNS** flow row reads
  the rule's operator-led verdict for free (flows.go maps
  `ns_vendor_jurisdiction` → DNS); the raw `dns.ns_hosting` Finding shows
  in the findings-by-probe view.
- [x] 2.5 Tests: operator table (per-entry incl. label-boundary),
  scanner synthesis (operator+country, no-GeoIP, anycast, multi-operator,
  no-NS), rule operator-led verdict. `go vet` + full suite green.
  Passive (no new network — reuses `dns.ns`/`ip.asn`). Live-smoke
  `wanderer scan cloudflare.com` → "DNS for cloudflare.com is run by
  Cloudflare"; `transip.nl` → "… run by TransIP".
- [x] 2.6 docs/operator.md ("DNS hosting" section) + CHANGELOG.

## 3. Wrap-up

- [x] 3.1 Commit + push (main).
- [x] 3.2 Archive (dns-hosting capability spec merged); tick
  `research-high-signal-observability` task 2.3.
