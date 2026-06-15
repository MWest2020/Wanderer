# Proposal: Transit-path probe — trace where a target's traffic actually goes

> **Status:** Design pass — awaiting Mark's scope call. No code lands
> until the Q1–Q3 decisions below are made.

## Why

Wanderer's current output leans on framework scores that feel abstract
("weinigzeggend") — they tell you a verdict without showing the
observed reality behind it. A `traceroute` to a target is the opposite:
concrete and self-evident. Tracing `buren.commonground.nu` shows the
path lands at **CYSO in NL** (`fuga-vmx2.network.cyso.net`) via
`as9143` (NL) and `ntt.net` (a JP-headquartered carrier):

```
 5: nl-srk03a-ri1-ae51-0.core.as9143.net
 7: xe-3-4-1-1.a00.amstnl07.nl.ce.gin.ntt.net
 8: CYSO-HOSTIN.ear3.Amsterdam1.Level3.net
 9: fuga-vmx2.network.cyso.net
10: 81.24.6.82
```

That single trace answers "where is this Nextcloud hosted, and what
jurisdictions does the traffic cross?" — the core sovereignty question.
The organisation/host is the spil in the web; the operator must see
**what goes where, and what is used (or misused) where**. Wanderer
already resolves IP → ASN → country (the `ip` probe + GeoLite2) but
throws away the *path*. This change adds the transit dimension and lets
the existing ASN/country machinery + the wand rule pack speak.

## What Changes

- A new `transit` probe traces the network path to each target IP and
  emits one Finding per hop (hop number, IP, reverse DNS, ASN, org,
  country, RTT) plus an aggregate Finding ("path crosses N countries /
  M ASNs; non-EU carriers: NTT, Level3").
- A new wand rule `wand.transit.eu_path` scores the path: `afhankelijk`
  when it transits a non-EU jurisdiction or a US-headquartered transit
  provider (CLOUD Act reach), reusing the existing vendor-jurisdiction
  rule pattern (ADR-0012).
- The UI renders the path as a hop list (the host as hub, the chain
  laid out) on the scan view.

## Intent

Make Wanderer's most sovereignty-relevant fact — the actual transit
path and hosting of a target — a first-class, observed, rendered
signal, not a derived score. This is the first concrete instance of the
"high-signal observability" direction (see the sibling
`research-high-signal-observability` proposal).

## Scope

- **Perimeter first** (`wanderer scan` / serve): trace from the scanner
  to each resolved target IP. The **destination-side hops** (the last
  few — the hosting provider, e.g. CYSO/NL/the IP) are robust regardless
  of the scanner's vantage; the *middle* transit hops are
  vantage-flavoured (see Risks).
- **Agent modus is the stronger follow-up**: tracing from the host
  itself answers "where does *my* traffic go", which is the real egress
  question. Out of scope for the first wave; noted as the natural next.

## Open questions

1. **Privileged vs unprivileged.** Classic ICMP traceroute needs
   `CAP_NET_RAW`/root. The example used `tracepath` (UDP, unprivileged).
   Recommendation: **unprivileged first** — shell out to
   `tracepath`/`traceroute` when present (optional external tool, like
   Amass), parse the hops; fall back to a pure-Go UDP/ICMP
   implementation (`golang.org/x/net`) with the capability documented.
   Keeps the container non-root.
2. **Destination vs transit scoring.** The destination hop (hosting
   provider) is a strong, robust signal; transit hops are
   vantage-dependent. Recommendation: **score the destination strongly,
   surface transit as informational** (named, not hard-failed) until an
   agent-modus vantage makes transit authoritative.
3. **Enrichment source.** Reuse the existing GeoLite2 ASN/country mmdb
   loader (the `ip` probe path) for hop ASN/country, and add reverse
   DNS for the provider-revealing hostname. Recommendation: **reuse**;
   no new data dependency.

## Risks

- **Vantage dependence.** A traceroute shows the path *from the
  scanner*. Mitigation: lean on destination-side hops for scoring;
  document that transit is vantage-flavoured; agent modus later.
- **`no reply` hops.** Firewalled hops appear as gaps (the example is
  full of them). The probe must record gaps without failing.
- **ICMP rate-limiting / slow traces.** Bound by `maxHops` +
  per-probe-timeout (the existing scanner budget model).
- **No-GeoIP installs.** Without the mmdb, hops degrade to IP + rDNS
  only (same graceful path as the `ip` probe today).

## Not in scope

- Agent-modus (on-host) tracing — the stronger vantage, a follow-up.
- Active path manipulation, BGP/RPKI route-origin checks — separate
  research leads.
- Replacing the `ip` probe; this complements it.

## Parallel-safe

New `internal/probe/transit/` package alongside the existing probes;
one new wand rule in `internal/assessor/wand/`; a scan-view UI section;
a new `transit-probe` capability spec. No schema change. The probe is
opt-in to the probe set and degrades gracefully without GeoIP or a
traceroute tool.
