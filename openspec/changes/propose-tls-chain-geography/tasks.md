# Tasks: TLS-chain geography

## 1. Direction decisions (resolve with Mark before building)

- [ ] 1.1 Q1 — where the aggregate is synthesised. Recommendation:
  scanner synthesis step alongside the existing `synthesise*` helpers,
  emitted as an observed Finding so the wand rule stays a pure annotation.
- [ ] 1.2 Q2 — CA table: ~10–12 curated issuer org/CN substrings → CA
  brand, grown in-repo; raw issuer org/CN retained as fallback + evidence.
- [ ] 1.3 Q3 — add a **Certificate** flow row mapped to `cert_issuer_eea`
  (one-line `flowRules` addition, pure presentation).
- [ ] 1.4 Q4 — enrich `tls.chain` with per-intermediate organisation +
  country (passive, read from the already-parsed certs); keep
  `intermediates` (CNs) for backward compatibility.

## 2. Implementation

- [ ] 2.1 CA naming: a table + helper (`certAuthority(issuerOrg, issuerCN)
  string`) mapping the issuer org/CN to a recognisable CA, raw values
  retained. Unit-tested per entry incl. CN-only and raw fallback.
- [ ] 2.2 (Q4) Probe enrichment: `tls.chain` intermediates gain
  `organisation` + `country` from the parsed certs (passive); existing
  `intermediates` (CNs) + `length` unchanged. Probe test updated.
- [ ] 2.3 Scanner synthesis: a `synthesiseCertChain` step running after
  pass 2, reading the apex `tls.issuer` + `tls.chain` into one observed
  `tls.chain_geography` Finding ("the TLS certificate for {domain} is
  issued by {CA} ({country})" + chain). No-issuer, no-country handled.
  Persisted + appended in `scanner.go`.
- [ ] 2.4 `wand.juridisch.cert_issuer_eea` leads its verdict with the
  named CA (tolerant of the JSON-reloaded attr shape); EEA scoring
  unchanged.
- [ ] 2.5 Rendering: add the **Certificate** flow row (`cert_issuer_eea`)
  to `flows.go`; the raw `tls.chain_geography` Finding shows in the
  findings-by-probe view.
- [ ] 2.6 Tests: CA table (per-entry incl. CN-only, fallback), scanner
  synthesis (named CA + country, no-country, unrecognised issuer,
  no-issuer), rule CA-led verdict, flow row present. `go vet` + full suite
  green. Passive (no new network — reuses `tls.issuer`/`tls.chain`).
  Live-smoke `wanderer scan <domain>` → "issued by …".
- [ ] 2.7 docs/operator.md ("TLS-chain geography" section) + CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (tls-chain-geography capability spec merged); tick
  `research-high-signal-observability` task 2.6b — Wave 2 complete.
