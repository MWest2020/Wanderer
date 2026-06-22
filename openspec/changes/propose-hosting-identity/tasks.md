# Tasks: hosting identity

## 1. Direction decisions (resolve with Mark before building)

- [x] 1.1 Q1 — where the aggregate is synthesised. Confirmed (Mark,
  2026-06-22): scanner synthesis step alongside `synthesiseMailRouting` /
  `synthesiseDNSHosting`, emitted as an observed Finding so the wand rule
  stays a pure annotation.
- [x] 1.2 Q2 — operator name source. Confirmed: an ASN-org normalisation
  table (friendly-name the common hosts) + raw-org fallback, grown
  in-repo; no rDNS/whois in v1.
- [x] 1.3 Q3 — hosting country = observed fact; operator HQ /
  jurisdiction nuance carried by the rule.
- [x] 1.4 Q4 — rDNS / whois enrichment deferred to a follow-up; v1 reuses
  `ip.asn` only (zero new network). Confirmed "Implement as proposed".

- [x] 2.1 Operator naming: an ASN-org normalisation table + helper
  (`hostingOperator(asnOrg) string`) mapping a raw ASN organisation to a
  recognisable host name, raw org retained as fallback + evidence.
  Unit-tested per entry incl. friendly-name, case-insensitive, raw
  fallback, empty. (`hostingidentity.go`.)
- [x] 2.2 Scanner synthesis: a `synthesiseHostingIdentity` step running
  after pass 2 (post-`ip.asn`), correlating the apex `dns.a`/`dns.aaaa`
  host × `ip.asn` into one observed `ip.hosting` Finding ("{domain} is
  hosted at {operator} ({country})"), observation severity. No-apex,
  no-GeoIP, and anycast handled. Persisted + appended in `scanner.go`.
- [x] 2.3 `wand.juridisch.apex_ip_eea` leads its verdict with the observed
  operator (tolerant of the JSON-reloaded attr shape); scoring unchanged.
  (`observedHostingOperators` + shared `joinAnd`.)
- [x] 2.4 Rendering: the Sovereignty overview's **Hosting** flow row reads
  the rule's operator-led verdict for free (flows.go maps `apex_ip_eea` →
  Hosting); the raw `ip.hosting` Finding shows in the findings-by-probe
  view.
- [x] 2.5 Tests: operator table (per-entry incl. friendly-name,
  case-insensitive, raw fallback), scanner synthesis (operator+country,
  unrecognised-org, no-GeoIP, anycast, no-apex), rule operator-led
  verdict. `go vet` + full suite green. Passive (no new network — reuses
  `dns.a`/`ip.asn`). Live-smoke `wanderer scan github.com` → "hosting
  operator … undetermined (no GeoIP)" in this GeoIP-less env.
- [x] 2.6 docs/operator.md ("Hosting identity" section) + CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (hosting-identity capability spec merged); tick
  `research-high-signal-observability` task 2.4.
