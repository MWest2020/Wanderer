# Tasks: TLS-chain geography

## 1. Direction decisions (resolve with Mark before building)

- [x] 1.1 Q1 — where the aggregate is synthesised. Confirmed (Mark,
  2026-06-24): scanner synthesis step alongside the existing `synthesise*`
  helpers, emitted as an observed Finding so the wand rule stays a pure
  annotation.
- [x] 1.2 Q2 — CA table: curated issuer org/CN substrings → CA brand,
  grown in-repo; raw issuer org/CN retained as fallback + evidence.
- [x] 1.3 Q3 — add a **Certificate** flow row mapped to `cert_issuer_eea`.
  Confirmed.
- [x] 1.4 Q4 — enrich `tls.chain` with per-intermediate organisation +
  country (passive, read from the already-parsed certs); keep
  `intermediates` (CNs) for backward compatibility. Confirmed "Both: row +
  probe enrich".

- [x] 2.1 CA naming: a table + helper (`certAuthority(issuerOrg, issuerCN)
  string`) mapping the issuer org/CN to a recognisable CA, raw values
  retained. Unit-tested per entry incl. CN-only and raw fallback.
  (`certchain.go`.)
- [x] 2.2 (Q4) Probe enrichment: `tls.chain` intermediates gain
  `organisation` + `country` from the parsed certs (passive); existing
  `intermediates` (CNs) + `length` unchanged. (No probe test file exists;
  covered via the scanner synthesis test + live smoke.)
- [x] 2.3 Scanner synthesis: a `synthesiseCertChain` step running after
  pass 2, reading the apex `tls.issuer` + `tls.chain` into one observed
  `tls.chain_geography` Finding ("the TLS certificate for {domain} is
  issued by {CA} ({country})" + chain). No-issuer, no-country handled.
  Persisted + appended in `scanner.go`.
- [x] 2.4 `wand.juridisch.cert_issuer_eea` leads its verdict with the
  named CA (`observedCertAuthority`, tolerant of the attr shape); EEA
  scoring unchanged.
- [x] 2.5 Rendering: added the **Certificate** flow row (`cert_issuer_eea`)
  to `flows.go`; the raw `tls.chain_geography` Finding shows in the
  findings-by-probe view.
- [x] 2.6 Tests: CA table (per-entry incl. CN-only, fallback), scanner
  synthesis (named CA + country + chain, no-country, unrecognised issuer,
  no-issuer), rule CA-led verdict, flow-row test repointed. `go vet` +
  full suite green. Passive (no new network — reuses
  `tls.issuer`/`tls.chain`). Live-smoke `wanderer scan w3.org` → "issued
  by Google Trust Services (US); chain ← …".
- [x] 2.7 docs/operator.md ("TLS-chain geography" section) + CHANGELOG.

## 3. Wrap-up

- [ ] 3.1 Commit + push (main).
- [ ] 3.2 Archive (tls-chain-geography capability spec merged); tick
  `research-high-signal-observability` task 2.6b — Wave 2 complete.
