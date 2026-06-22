# Tasks: web third-party origin map

## 1. Direction decisions (resolve with Mark before building)

- [x] 1.1 Q1 — where the aggregate is synthesised. Confirmed (Mark,
  2026-06-22): scanner synthesis step alongside the four Wave-1
  `synthesise*` helpers, emitted as an observed Finding so the wand rule
  stays a pure annotation.
- [x] 1.2 Q2 — vendor table: curated host suffixes + ASN-org fallback,
  reusing `operatorBySuffix`, grown in-repo; no new dependency.
- [x] 1.3 Q3 — group the map by vendor (union kinds + countries), retain
  per-host detail in evidence. Confirmed.
- [x] 1.4 Q4 — the rule leads with the non-EEA vendor names (the export
  surface); all-EEA pages keep a clean count with no scary lead. Confirmed
  "Implement as proposed".

- [x] 2.1 Vendor naming: a suffix→vendor table + helper
  (`thirdPartyVendor(host, asnOrg) string`) reusing `operatorBySuffix`,
  ASN-org fallback, raw host retained. Unit-tested per entry incl.
  label-boundary + fallback. (`originmap.go`.)
- [x] 2.2 Scanner synthesis: a `synthesiseOriginMap` step running after
  pass 2 (post-`Related`/`ip.asn`), correlating `http.third_party` (+
  `kinds`) × `ip.asn` into one observed `http.origin_map` Finding,
  vendor-grouped (union kinds + countries). Observation severity. The
  observed map records country only; the EEA judgment is the rule's.
  No-third-parties (gated on `http.response`), no-GeoIP, anycast handled.
  Persisted + appended in `scanner.go`.
- [x] 2.3 `wand.technologie.third_parties_eea` leads its verdict with the
  observed non-EEA vendor names (`observedNonEEAVendors`, tolerant of the
  JSON-reloaded attr shape); the in/out-EEA count is unchanged. All-EEA
  pages keep their clean count with no lead.
- [x] 2.4 Rendering: the Sovereignty overview's **Third parties** flow row
  reads the vendor-led verdict for free (flows.go maps
  `third_parties_eea` → Third parties); the raw `http.origin_map` Finding
  shows in the findings-by-probe view.
- [x] 2.5 Tests: vendor table (per-entry incl. label-boundary, fallback),
  scanner synthesis (vendor grouping + kinds union, multi-vendor,
  no-GeoIP, unrecognised host, page-fetched-no-third-parties,
  probe-not-run), rule vendor-led verdict (non-EEA named, EEA not
  flagged). `go vet` + full suite green. Passive (no new network — reuses
  `http.third_party`/`ip.asn`). Live-smoke `wanderer scan getbootstrap.com`
  → "loads styles/assets from …, scripts from …".
- [x] 2.6 docs/operator.md ("Web third-party origin map" section) +
  CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (third-party-origin capability spec merged); tick
  `research-high-signal-observability` task 2.5.
