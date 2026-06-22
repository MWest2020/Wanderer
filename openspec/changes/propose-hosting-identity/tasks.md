# Tasks: hosting identity

## 1. Direction decisions (resolve with Mark before building)

- [ ] 1.1 Q1 — where the aggregate is synthesised. Recommendation:
  scanner synthesis step alongside `synthesiseMailRouting` /
  `synthesiseDNSHosting`, emitted as an observed Finding so the wand rule
  stays a pure annotation.
- [ ] 1.2 Q2 — operator name source. Recommendation: an ASN-org
  normalisation table (friendly-name the common hosts, strip `-AS`/`,
  Inc.` noise) + raw-org fallback, grown in-repo; no rDNS/whois in v1.
- [ ] 1.3 Q3 — hosting country = observed fact; operator HQ /
  jurisdiction nuance carried by the rule.
- [ ] 1.4 Q4 — rDNS / whois enrichment deferred to a follow-up; v1 reuses
  `ip.asn` only (zero new network).

## 2. Implementation

- [ ] 2.1 Operator naming: an ASN-org normalisation table + helper
  (`hostingOperator(asnOrg) string`) mapping a raw ASN organisation to a
  recognisable host name, raw org retained. Unit-tested per entry incl.
  the strip/fallback paths.
- [ ] 2.2 Scanner synthesis: a `synthesiseHostingIdentity` step running
  after pass 2 (post-`ip.asn`), correlating the apex `dns.a`/`dns.aaaa`
  host × `ip.asn` into one observed `ip.hosting` Finding ("{domain} is
  hosted at {operator} ({country})"), observation severity. No-apex,
  no-GeoIP, and anycast handled. Persisted + appended in `scanner.go`.
- [ ] 2.3 `wand.juridisch.apex_ip_eea` leads its verdict with the observed
  operator (tolerant of the JSON-reloaded attr shape); scoring unchanged.
- [ ] 2.4 Rendering: confirm the Sovereignty overview's **Hosting** flow
  row now reads the operator-led verdict; the raw `ip.hosting` Finding
  shows in the findings-by-probe view.
- [ ] 2.5 Tests: operator table (per-entry incl. friendly-name, strip,
  raw fallback), scanner synthesis (operator+country, unrecognised-org,
  no-GeoIP, anycast, no-apex), rule operator-led verdict. `go vet` + full
  suite green. Passive (no new network — reuses `dns.a`/`ip.asn`).
  Live-smoke `wanderer scan <domain>` → "hosted at …".
- [ ] 2.6 docs/operator.md ("Hosting identity" section) + CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (hosting-identity capability spec merged); tick
  `research-high-signal-observability` task 2.4.
