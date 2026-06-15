# Tasks: Transit-path probe (design pending)

Every task is a design checkpoint until the Q1–Q3 calls are made.

## 1. Direction decisions

- [ ] 1.1 Q1 — privileged vs unprivileged. Recommendation: unprivileged
  first (shell out to `tracepath`/`traceroute`, fall back to pure-Go
  UDP/ICMP with documented capability).
- [ ] 1.2 Q2 — destination vs transit scoring. Recommendation: score
  the destination hop; surface transit as informational until agent
  modus.
- [ ] 1.3 Q3 — enrichment source. Recommendation: reuse the GeoLite2
  ASN/country loader + add reverse DNS.

## 2. Implementation skeleton (after sign-off)

- [ ] 2.1 `internal/probe/transit/` — trace to each target IP; per-hop
  Finding (hop#, ip, rdns, asn, org, country, rtt) + an aggregate
  Finding (countries/ASNs crossed, non-EU carriers). Handle `no reply`
  hops as gaps; bound by maxHops + per-probe-timeout.
- [ ] 2.2 Reuse the GeoLite2 ASN/country enrichment; reverse-DNS each hop.
- [ ] 2.3 `wand.transit.eu_path` rule (ADR-0012 vendor-jurisdiction
  pattern): afhankelijk on non-EU / US-carrier transit; cite hop IDs as
  evidence.
- [ ] 2.4 Scan-view UI: render the path as a hop list.
- [ ] 2.5 Tests: probe parser (incl. no-reply gaps) against captured
  `tracepath`/`traceroute` output; rule unit test; UI Playwright spec.
- [ ] 2.6 docs/operator.md: the probe, the tracepath/traceroute
  dependency, the vantage caveat.

## 3. Wrap-up

- [ ] 3.1 Commit + push.
- [ ] 3.2 Archive.
