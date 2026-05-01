# Proposal: Reverse DNS annotation for flow Findings

## Intent

The flow probe (ADR-0010) emits one Finding per unique runtime
`(destination_ip, destination_port)`. When the vendor classifier
knows the destination, the Finding carries `provider`, `region`,
and (with GeoLite2 wired) `asn` / `organisation` / `country`. When
the classifier does **not** know the destination — generic cloud
ranges, customer infrastructure, anything not in
`vendors.yaml` — the Finding is `egress.flow.unknown` with only the
raw IP. Accurate, but lossy: the operator reviewing the dashboard
sees `203.0.113.5:443` and has no quick handle on whose IP that is.

Reverse DNS (PTR) lookups recover useful labels for many of those
addresses: `ec2-203-0-113-5.eu-west-1.compute.amazonaws.com`,
`lb-foo.gcp.example.com`, customer hostnames in private resolvers.
This change adds optional PTR annotation so a flow Finding's
Attributes can carry `reverse_dns: "<hostname>"` next to the
existing IP.

## Scope

**In scope:**

- A `ReverseDNSResolver` interface in `internal/probe/egress/flow/`,
  with a default implementation that wraps `net.Resolver.LookupAddr`
  with a per-lookup timeout.
- A per-IP cache inside the Aggregator window so multiple ports to
  the same IP produce exactly one PTR query.
- Config: `egress.flow.reverse_dns: { enabled, timeout }`.
  Default `enabled: false`. Default timeout `500ms`.
- Annotation: on success, the Finding's Attributes get
  `reverse_dns: "<hostname>"`. Failure (NXDOMAIN, timeout, refused)
  is **silent** — no annotation, no error Finding. Reverse DNS is
  best-effort enrichment, not a probe in its own right.
- Documentation: a Reverse DNS section in `docs/egress.md`, a
  privacy-tradeoff addendum to ADR-0010, and a CHANGELOG entry.

**Out of scope:**

- Reverse DNS for the static egress probe. Static egress already
  works from hostnames; the flow probe is the IP-only case.
- DNSSEC / FCrDNS / forward-confirm. PTR records are advisory
  labels, not trust anchors. Operators who need FCrDNS can do that
  out-of-band; the wire format does not change.
- A custom resolver path (DoH, alternate servers). The system
  resolver is what the host already uses; bringing a parallel
  resolver only widens the surface.
- Caching across agent ticks. The window-scoped cache is enough for
  dedupe within one sample; bookkeeping for cross-tick caching adds
  state for marginal benefit.

## Privacy: why the default is off

A PTR query leaks the observation back through the host's DNS path
— the resolver, every cache between, and the authoritative server
for the reverse zone all learn that *this host just saw IP X*. In
a sovereignty monitor whose explicit purpose is reducing data
flight, that side-channel is non-trivial.

Therefore:

1. Reverse DNS is off by default. Operators must opt in
   consciously, the same way they opt in to the flow probe itself.
2. ADR-0010 gets a "Reverse DNS addendum" recording the tradeoff
   so the choice is auditable.

This sits inside the existing "safest, most defensible default"
posture. Operators who run the agent in a closed lab (where leaking
to the local resolver is acceptable) flip the toggle; operators in
a tight sovereignty context leave it off and accept IP-only
Findings.

## Wand dimensions informed

- **Juridisch** (primary): hostname hints often resolve directly to
  provider/region patterns the classifier missed, raising the
  evidence quality of jurisdictional Findings.
- **Data & AI**: identity-federation and telemetry endpoints in
  customer DNS sometimes only show up as PTR labels.

## Passive / active boundary

The agent issues DNS queries via the resolver it already uses.
No new outbound channel; no application traffic initiated.

## Parallel-safe

Touches `internal/probe/egress/flow/` (extension) and one config
struct + one builder line in `cmd/wanderer/agent.go`. No schema
changes, no DB migration.
