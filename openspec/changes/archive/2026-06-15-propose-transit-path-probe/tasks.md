# Tasks: Transit-path probe

## 1. Direction decisions

- [x] 1.1 Q1 — **unprivileged first**: `ToolTracer` shells out to
  `tracepath` (preferred) / `traceroute -n`, streams stdout, parses
  tolerantly. Degrades to `transit.unavailable` when neither is on PATH.
- [x] 1.2 Q2 — **score the destination**: `wand.transit.eu_path` scores
  the destination hop's jurisdiction (EEA→soeverein, non-EEA→
  afhankelijk); non-EEA transit hops are named informationally and do
  not downgrade.
- [x] 1.3 Q3 — **reuse GeoLite2**: hops are enriched via the `ip`
  probe's `Lookup` (nil-safe) + reverse DNS.

## 2. Implementation

- [x] 2.1 `internal/probe/transit/` — trace to the target's primary IP;
  per-hop Finding (hop, ip, rdns, asn, org, country, rtt) + a
  `transit.path` aggregate; no-reply hops as gaps; bounded by maxHops
  (default 20) + the per-probe timeout; partial path survives a
  timeout (streamed stdout).
- [x] 2.2 Reuse GeoLite2 ASN/country + reverse-DNS per hop.
- [x] 2.3 `wand.transit.eu_path` rule (registered in DefaultRules).
- [x] 2.4 Wired into `buildProbes`.
- [x] 2.5 Tests: parser (tracepath + traceroute formats, no-reply,
  dedup), Run (unavailable/aggregate), rule (EEA/non-EEA/transit/
  no-geo). Code + security reviewed (clean). Live-smoked against real
  `tracepath` (rDNS revealed providers; gaps handled).
- [x] 2.6 docs/operator.md + CHANGELOG.
- [ ] 2.7 FOLLOW-UPS (separate increments): a dedicated scan-view
  **path rendering** (today the hops surface in the generic
  findings-by-probe view) with a Playwright spec; the **agent-modus
  on-host trace** (the stronger vantage for "where does *my* traffic
  go").

## 3. Wrap-up

- [x] 3.1 Commit + push.
- [x] 3.2 Archive (transit-probe capability spec merged).
