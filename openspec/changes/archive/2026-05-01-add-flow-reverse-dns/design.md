# Design: Reverse DNS annotation for flow Findings

## Resolver interface

```go
// ReverseDNSResolver maps a destination IP to a best-effort hostname.
// Implementations MUST honour a per-call deadline and MUST return
// ok=false (no error surfaced to the caller) on any failure mode —
// NXDOMAIN, timeout, refused, or transport error. Reverse DNS is
// strictly enrichment; failure is normal and not a Finding.
type ReverseDNSResolver interface {
    Reverse(ctx context.Context, ip string) (host string, ok bool)
}
```

A default implementation `NewReverseDNSResolver(timeout)` wraps
`net.DefaultResolver.LookupAddr`. The wrapper:

1. Applies the configured per-lookup timeout via `context.WithTimeout`.
2. Picks the first hostname returned (PTR can yield multiple; we
   keep one to keep the wire shape simple — operators who need the
   full list can run an out-of-band query).
3. Strips a single trailing dot.
4. Returns `ok=false` for empty results, errors, or context-cancelled
   lookups.

## Cache

The Aggregator already deduplicates by `(IP, port)`. Two ports to
the same IP produce two Findings, but should produce only **one**
PTR query. A per-IP cache sized to the unique IP set covers that
without crossing tick boundaries:

```go
type ptrCache struct {
    mu    sync.Mutex
    seen  map[string]ptrResult
}

type ptrResult struct {
    host string
    ok   bool
}
```

The cache lives on `Aggregator` and is consulted from
`buildFlowFinding`. Lifetime is one `Inspect` call (one sampling
window). Across ticks the cache resets — that bounds memory and
keeps stale labels from drifting across long-running agents.

## Wire format

On success the Finding's Attributes gain one key:

```
reverse_dns: "ec2-203-0-113-5.eu-west-1.compute.amazonaws.com"
```

On failure: nothing. No `reverse_dns_error`, no `reverse_dns: null`.
A consumer (UI, assessor, exporter) treats `reverse_dns` as
optional; absence is the default state.

## Config surface

```yaml
inventory:
  egress:
    flow:
      enabled: true
      window: 60s
      reverse_dns:
        enabled: false      # default; opt-in
        timeout: 500ms      # per-lookup
```

Two knobs is enough. `enabled` gates the entire feature; `timeout`
lets operators tune for a slow resolver. The cache is internal and
not configurable.

## Agent builder

`buildFlowProbe` in `cmd/wanderer/agent.go` constructs the
resolver only when `cfg.Egress.Flow.ReverseDNS.Enabled` is true.
When disabled, the `Flow.ReverseResolver` field stays `nil` and
`buildFlowFinding` skips the annotation path. The opt-in is hard:
no resolver constructed, no DNS traffic possible.

## Why a separate resolver and not extending HostResolver

`HostResolver` (existing) is the GeoLite2-backed ASN/country path,
fully offline. Folding PTR into the same interface would mix two
concepts: one queries a local mmdb (no network), the other queries
the system resolver (network). They have different failure modes,
different latency profiles, and very different privacy characters.
Keeping them separate matches the existing pattern and keeps the
"this call goes to the network" surface explicit.

## Test strategy

Unit tests in `internal/probe/egress/flow/`:

- **Stub annotates**: a stub resolver that always returns
  `("ptr.example.", true)` produces Findings with `reverse_dns:
  "ptr.example"` (trailing dot stripped).
- **Stub fails silently**: a resolver that returns `ok=false` leaves
  Attributes unchanged; no `reverse_dns` key appears.
- **Cache**: a counting stub records lookups; two events for the
  same IP on different ports yield one lookup.
- **Nil resolver**: when `Aggregator.Findings` is called with a nil
  `ReverseDNSResolver`, behaviour is byte-identical to today.
- **Timeout**: a stub that blocks past the per-call deadline returns
  `ok=false` from the default `NewReverseDNSResolver(50*time.Millisecond)`.

No integration test against real DNS. The unit-tested seam is the
contract we ship; the system resolver is the kernel's job.
