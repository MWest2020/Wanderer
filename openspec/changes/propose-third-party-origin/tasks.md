# Tasks: web third-party origin map

## 1. Direction decisions (resolve with Mark before building)

- [ ] 1.1 Q1 — where the aggregate is synthesised. Recommendation:
  scanner synthesis step alongside the four Wave-1 `synthesise*` helpers,
  emitted as an observed Finding so the wand rule stays a pure annotation.
- [ ] 1.2 Q2 — vendor table size/location. Recommendation: ~15 curated
  host suffixes + ASN-org fallback, reusing `operatorBySuffix`, grown
  in-repo; no new dependency.
- [ ] 1.3 Q3 — group the map by vendor (union kinds + countries), retain
  per-host detail in evidence.
- [ ] 1.4 Q4 — the rule leads with the non-EEA vendor names (the export
  surface); all-EEA pages keep a clean count with no scary lead.

## 2. Implementation

- [ ] 2.1 Vendor naming: a suffix→vendor table + helper
  (`thirdPartyVendor(host, asnOrg) string`) reusing `operatorBySuffix`,
  ASN-org fallback, raw host retained. Unit-tested per entry incl.
  label-boundary + fallback.
- [ ] 2.2 Scanner synthesis: a `synthesiseOriginMap` step running after
  pass 2 (post-`Related`/`ip.asn`), correlating `http.third_party` (+
  `kinds`) × `ip.asn` into one observed `http.origin_map` Finding,
  vendor-grouped (union kinds + countries), non-EEA vendors flagged.
  Observation severity. No-third-parties, no-GeoIP, anycast handled.
  Persisted + appended in `scanner.go`.
- [ ] 2.3 `wand.technologie.third_parties_eea` leads its verdict with the
  observed non-EEA vendor names (tolerant of the JSON-reloaded attr
  shape); the in/out-EEA count is unchanged.
- [ ] 2.4 Rendering: confirm the Sovereignty overview's **Third parties**
  flow row now reads the vendor-led verdict; the raw `http.origin_map`
  Finding shows in the findings-by-probe view.
- [ ] 2.5 Tests: vendor table (per-entry incl. label-boundary, fallback),
  scanner synthesis (vendor grouping + kinds union, all-EEA, no-GeoIP,
  anycast, unrecognised host, no-third-parties), rule vendor-led verdict.
  `go vet` + full suite green. Passive (no new network — reuses
  `http.third_party`/`ip.asn`). Live-smoke `wanderer scan <domain>` →
  "loads from …".
- [ ] 2.6 docs/operator.md ("Web third-party origin map" section) +
  CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (third-party-origin capability spec merged); tick
  `research-high-signal-observability` task 2.5.
