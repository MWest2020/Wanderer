# Tasks: Egress flow probe

## 1. eBPF source + build

- [x] 1.1 `internal/probe/egress/flow/bpf/connect.bpf.c` — tracepoint program
- [ ] 1.2 `go generate` integration via `bpf2go`
- [ ] 1.3 CI matrix entry to ensure generated files stay in sync

## 2. Userspace loader

- [x] 2.1 `flow.go` Inspector implementation, opens / attaches / reads perf events
- [x] 2.2 Aggregator dedups by (daddr, dport) within the window
- [x] 2.3 Privilege detection emits `egress.flow.unavailable` cleanly

## 3. Classifier reuse

- [x] 3.1 Hand each unique destination to existing `egress.classify.Classify`
- [ ] 3.2 Reverse DNS best-effort, IP-only Finding when it fails
- [x] 3.3 Optional GeoLite2 annotation via the existing IP probe resolver

## 4. Config + agent wiring

- [x] 4.1 Extend `wanderer-agent.yaml` with `egress.flow: { enabled, window }`
  (capture_pids dropped from scope — see Notes)
- [x] 4.2 Register the inspector in `cmd/wanderer/agent.go`

## 5. Tests + docs

- [x] 5.1 Aggregator unit tests with synthetic events
- [x] 5.2 `docs/egress.md` flow section + capability/kernel matrix
- [x] 5.3 ADR-0010 recording the eBPF choice and the kernel-version contract
- [x] 5.4 CHANGELOG entry under `### Added`

## Notes — deferred kernel-attach work

The userspace half of this change is fully landed (Inspector,
Aggregator, classifier reuse, agent wiring, tests, ADR, docs). The
**kernel-attach half is explicitly deferred**:

- 1.2 / 1.3 (bpf2go integration + CI matrix): the dev environment
  for this change does not have clang or llvm-strip installed
  (`which clang` returns nothing, `clang --version` errors). bpf2go
  shells out to clang to generate the embedded `.go` file, so we
  cannot run it here. The CO-RE source is shipped at
  `internal/probe/egress/flow/bpf/connect.bpf.c` for review and
  for the operator who continues the work; ADR-0010 documents the
  exact build command. Once a build host with the toolchain runs
  `go generate ./internal/probe/egress/flow/...`, the generated
  `bpf_bpfel_x86.go` lands and the kernel attach lights up
  without any further code changes to the inspector surface.
- 3.2 (reverse DNS): the userspace half emits IP-only Findings
  through the existing classifier, which already handles raw IPs
  by falling back to `egress.flow.unknown`. Reverse-DNS annotation
  is a follow-up improvement that needs the kernel attach in place
  to be testable end-to-end; it adds attribute keys (`hostname`,
  `dns_resolved`) without changing the Finding shape.
- Loader unit test (originally implied by 1.2): until the BPF
  object is generated, the loader has nothing to load. The
  Aggregator and Run paths are exercised via synthetic events in
  `flow_test.go`; the Available() branches are exercised by
  `TestFlow_Available_*`.

The privilege / opt-in scenarios in the egress-probe spec
(`Privilege detection emits egress.flow.unavailable cleanly`,
`Default config does not load the program`) hold today: the
inspector returns Available=false with a reason that names the
missing kernel/capability/loader piece, and the agent does not
construct the Inspector at all when `egress.flow.enabled: false`.

The capture-time `capture_pids` filter from the original task
list (4.1) is dropped from this iteration: the in-kernel program
already filters AF_UNIX, and per-PID filtering in userspace is
trivial to add later (`Aggregator.Add` is a single-line guard) so
deferring it does not lock in a contract that future PID filtering
would need to break.
