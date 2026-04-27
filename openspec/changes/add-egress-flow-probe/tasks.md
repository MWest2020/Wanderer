# Tasks: Egress flow probe

## 1. eBPF source + build

- [ ] 1.1 `internal/probe/egress/flow/bpf/connect.bpf.c` — tracepoint program
- [ ] 1.2 `go generate` integration via `bpf2go`
- [ ] 1.3 CI matrix entry to ensure generated files stay in sync

## 2. Userspace loader

- [ ] 2.1 `flow.go` Inspector implementation, opens / attaches / reads perf events
- [ ] 2.2 Aggregator dedups by (daddr, dport) within the window
- [ ] 2.3 Privilege detection emits `egress.flow.unavailable` cleanly

## 3. Classifier reuse

- [ ] 3.1 Hand each unique destination to existing `egress.classify.Classify`
- [ ] 3.2 Reverse DNS best-effort, IP-only Finding when it fails
- [ ] 3.3 Optional GeoLite2 annotation via the existing IP probe resolver

## 4. Config + agent wiring

- [ ] 4.1 Extend `wanderer-agent.yaml` with `egress.flow: { enabled, window, capture_pids? }`
- [ ] 4.2 Register the inspector in `cmd/wanderer/agent.go`

## 5. Tests + docs

- [ ] 5.1 Aggregator unit tests with synthetic events
- [ ] 5.2 `docs/egress.md` flow section + capability/kernel matrix
- [ ] 5.3 ADR-XXXX recording the eBPF choice and the kernel-version contract
- [ ] 5.4 CHANGELOG entry under `### Added`
