# Tasks: Reverse DNS annotation for flow Findings

## 1. Resolver interface + default implementation

- [x] 1.1 `ReverseDNSResolver` interface in
  `internal/probe/egress/flow/reverse.go`
- [x] 1.2 `NewReverseDNSResolver(timeout)` default impl wrapping
  `net.DefaultResolver.LookupAddr`
- [x] 1.3 Trailing-dot strip + first-hostname-wins normalisation

## 2. Aggregator integration

- [x] 2.1 `Aggregator` carries a per-IP `ptrCache`
- [x] 2.2 `Aggregator.Findings` accepts a `ReverseDNSResolver` (nil-safe)
- [x] 2.3 `buildFlowFinding` consults the cache, populates
  `attrs["reverse_dns"]` on success, leaves it absent on failure

## 3. Config + agent wiring

- [x] 3.1 `EgressFlowReverseDNS{Enabled, Timeout}` struct in
  `internal/agent/config.go`, embedded under `EgressFlow`
- [x] 3.2 Default timeout `500ms` when unset and enabled
- [x] 3.3 `cmd/wanderer/agent.go::buildFlowProbe` constructs the
  resolver only when enabled; otherwise leaves the field nil

## 4. Tests

- [x] 4.1 Stub resolver annotates Findings (success path)
- [x] 4.2 Stub returning `ok=false` leaves Attributes untouched
- [x] 4.3 Cache: same IP across multiple ports → one lookup
- [x] 4.4 Nil resolver produces today's wire format byte-for-byte
- [x] 4.5 Default impl honours per-call timeout

## 5. Docs + changelog

- [x] 5.1 `docs/egress.md` — Reverse DNS section under the flow
  probe docs (config, default-off rationale, what it adds)
- [x] 5.2 ADR-0010 addendum on the privacy tradeoff
- [x] 5.3 `CHANGELOG.md` entry under `### Added`
