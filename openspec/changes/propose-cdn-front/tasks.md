# Tasks: CDN / front detection

## 1. Direction decisions (resolve with Mark before building)

- [ ] 1.1 Q1 — where the aggregate is synthesised. Recommendation:
  scanner synthesis step alongside the existing `synthesise*` helpers,
  emitted as an observed Finding so the wand rule stays a pure annotation.
- [ ] 1.2 Q2 — CDN signature table: ~10–12 curated edges keyed on ASN-org
  + server-header substrings, grown in-repo; conservative entries only.
- [ ] 1.3 Q3 — "fronted" when the apex ASN org OR the server header
  matches a signature; record which signal(s) fired as evidence.
- [ ] 1.4 Q4 — ship CDN/front detection alone; split TLS-chain geography
  polish into its own follow-up lead (cert issuer used here only as
  corroborating evidence, not scored).

## 2. Implementation

- [ ] 2.1 CDN detection: a signature table + helper
  (`cdnFront(asnOrg, serverHeader) (edge string, signals []string)`)
  matching ASN-org/server substrings to a recognisable edge, raw values
  retained. Unit-tested per entry incl. header-only, org-only, and
  no-match.
- [ ] 2.2 Scanner synthesis: a `synthesiseCDNFront` step running after
  pass 2, correlating the apex `ip.asn` × `http.response` (server header)
  × `tls.issuer` into one observed `http.cdn_front` Finding ("{domain}'s
  apex is fronted by {edge} ({country})" or "no CDN/edge front detected").
  No-apex-evidence, no-GeoIP (header-only), and anycast handled. Persisted
  + appended in `scanner.go`.
- [ ] 2.3 `wand.technologie.no_us_hyperscaler` leads its verdict with the
  named apex front when fronted (tolerant of the JSON-reloaded attr
  shape); the US-hyperscaler-in-path scoring is unchanged.
- [ ] 2.4 Rendering: confirm the Sovereignty overview's **CDN /
  hyperscaler** flow row now reads the front-led verdict; the raw
  `http.cdn_front` Finding shows in the findings-by-probe view.
- [ ] 2.5 Tests: signature table (per-entry incl. header-only, org-only,
  no-match), scanner synthesis (fronted+country, server-header-only,
  anycast, directly-served, no-apex-evidence), rule front-led verdict.
  `go vet` + full suite green. Passive (no new network — reuses
  `ip.asn`/`http.response`/`tls.issuer`). Live-smoke `wanderer scan
  <cloudflare-fronted domain>` → "apex is fronted by …".
- [ ] 2.6 docs/operator.md ("CDN / front detection" section) + CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (cdn-front capability spec merged); tick
  `research-high-signal-observability` task 2.6 (and note the TLS-chain
  geography split as a new backlog sibling).
