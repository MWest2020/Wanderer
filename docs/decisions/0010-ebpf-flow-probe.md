# 0010 — eBPF for the egress flow probe

**Status:** Accepted. Userspace landed 2026-04-29 morning; kernel
attach via cilium/ebpf + bpf2go-generated bindings landed
2026-04-29 afternoon through a pinned Fedora-42 builder container
(see `build/bpf-builder/Dockerfile`). The compiled BPF object is
committed at `internal/probe/egress/flow/connect_x86_bpfel.o` so
`go build ./...` works on developer hosts without the eBPF
toolchain.

**Context.** The static egress probe (`internal/probe/egress`) reads
config files, `/proc/<pid>/environ`, and systemd unit files. It can
identify env vars and configuration that point at S3 / Slack /
SMTP relays / OIDC issuers, but it cannot see destinations an
application *assembles at runtime* — `https://${REGION}.${VENDOR}/api`
is invisible until the request actually leaves the host.

For the assessor to score sovereign-vs-foreign egress with
production accuracy, we need runtime evidence: the actual
`connect()` destinations the host issues during a sampling window.

**Considered options.**

1. **eBPF tracepoint on `sys_enter_connect`.** Tiny in-kernel program
   reads the sockaddr, filters AF_UNIX, emits a perf event per
   non-local connect. Userspace aggregates by `(daddr, dport)`,
   classifies via the existing `egress.classify.Classify`, and
   emits Findings.
2. **Custom kernel module.** Same outcome, more attack surface,
   loaded code outside the verifier. Rejected.
3. **Packet capture (libpcap/AF_PACKET).** Sees full packets, not
   just connect events. Crosses the "passive observation"
   boundary the project takes seriously: we explicitly do not want
   to read application data. Rejected.
4. **Proxy injection (transparent SOCKS / HTTP CONNECT).** Active,
   per-vendor SDK rewriting needed, defeats jurisdictional clarity.
   Rejected.

**Decision.** Option 1. eBPF is the boring, supportable choice on
modern Linux: the verifier bounds the program, perf events are a
stable kernel ABI, and `cilium/ebpf` ships pure-Go loaders so the
agent stays CGo-free. We attach a `tracepoint/syscalls/sys_enter_connect`
program (CO-RE), filter AF_UNIX, and emit
`{pid, comm, family, daddr, dport}` to a perf ring buffer.

**Kernel-version contract.** The probe targets kernel 5.8+ with
`CONFIG_DEBUG_INFO_BTF=y` (CO-RE). Older kernels return
`egress.flow.unavailable` cleanly. The `/sys/kernel/btf/vmlinux`
file is the runtime signal.

**Capability contract.** `CAP_BPF` + `CAP_PERFMON` (introduced in
5.8) are sufficient — the agent does not need full root.
`/proc/self/status` exposes the bounding set; absent the
capabilities, the inspector emits `egress.flow.unavailable` exactly
once per tick rather than crashing.

**Opt-in default.** `egress.flow.enabled: false` in
`wanderer-agent.yaml`. An operator must consciously accept the
kernel-level capability cost before any kernel attach happens. With
the default config the inspector is not even constructed, so the
agent emits no `egress.flow.*` Findings (not even `unavailable`).

**Userspace + kernel surface.**

```
internal/probe/egress/flow/
  flow.go                   # Inspector: Available, Inspect, Run; Aggregator
  flow_test.go              # Aggregator + classifier-reuse tests
  gen.go                    # //go:generate bpf2go directive
  kernel_linux.go           # KernelSource: load + attach + perf read (build-tagged)
  kernel_stub.go            # NewKernelSource stub on non-linux/amd64
  connect_x86_bpfel.go      # bpf2go-generated bindings (committed)
  connect_x86_bpfel.o       # compiled BPF blob, embedded via //go:embed (committed)
  bpf/connect.bpf.c         # CO-RE tracepoint source (auditable)
  bpf/vmlinux.h             # generated from host BTF; gitignored

build/bpf-builder/Dockerfile  # pinned Fedora 42 + clang 20 + libbpf 1.5 + bpf2go
scripts/bpf-build.sh          # runs `go generate` inside the builder container
```

The Aggregator dedups by `(destination_ip, destination_port)`; the
first observation wins (so `process_name` records the first program
seen connecting to that destination, which is more useful for
jurisdictional review than a churn of repeats).

**Build path.** A developer who edits `bpf/connect.bpf.c` runs
`./scripts/bpf-build.sh` from the repo root. The script:

1. Builds (or reuses) the `wanderer-bpf-builder` image
   (~600 MB, one-time cost).
2. Regenerates `bpf/vmlinux.h` from the host's
   `/sys/kernel/btf/vmlinux` if the host BTF is newer.
3. Runs `go generate ./internal/probe/egress/flow/...` inside the
   container, which invokes bpf2go → clang → embedded `.o` plus
   matching `.go` bindings.

The resulting `connect_x86_bpfel.{go,o}` artefacts are committed to
the repo. CI (and any developer host without the eBPF toolchain)
runs plain `go build ./...` and gets a working binary.

**Deferred follow-up work.**

- arm64 target (`bpf2go -target amd64,arm64`) — adds
  `connect_arm64_bpfel.{go,o}` so the agent runs on ARM
  perimeter hosts. One-line change to `gen.go`; covered by the
  same builder container.
- Loader unit test that opens, attaches, and detaches against a
  real kernel — must skip cleanly on hosts without
  `/sys/kernel/btf/vmlinux`. The current build cannot exercise
  the kernel attach (no root in CI), so the loader is covered by
  the host integration test instead (run as root: see
  `docs/egress.md` "runtime flow probe" section).

## 2026-05-01 addendum — reverse DNS annotation (opt-in)

The original deferred-work list named "reverse DNS annotation for
IP-only destinations" without weighing the privacy cost. Landing
the feature forced that decision: a PTR query leaks the
observation back through the host's resolver, which a sovereignty
monitor cannot default-on without consent.

**Decision.** Reverse DNS annotation is opt-in via
`egress.flow.reverse_dns.enabled: true`, mirroring the flow probe's
own opt-in toggle. The default-off posture aligns with the
"safest, most defensible default" principle that already governs
the flow probe.

**Wire shape.** On success the Finding's Attributes gain
`reverse_dns: "<hostname>"`. On failure (NXDOMAIN, timeout,
refused, transport error) nothing is added — no error Finding, no
`reverse_dns: null`. PTR records are advisory labels, not trust
anchors; FCrDNS / DNSSEC are out of scope.

**Cache lifetime.** One sampling window. A per-IP cache inside the
Aggregator guarantees one PTR query per unique destination IP per
tick; the cache resets across ticks to bound memory and prevent
stale labels from drifting across long-running agents.

**References.**

- Proposal:
  `openspec/changes/archive/2026-05-01-add-flow-reverse-dns/proposal.md`
- Spec delta applied to:
  `openspec/specs/egress-probe/spec.md`

**Why now.** The static egress probe shipped with a known gap;
landing the userspace half + the auditable C source makes the
deferred work a self-contained build-environment task rather than a
greenfield design. ADR-0009 (dual-framework assessor) is the
companion entry: both ADRs converge on the assessor having more
authoritative evidence.

**References.**

- Original proposal:
  `openspec/changes/archive/2026-04-29-add-egress-flow-probe/proposal.md`
- Design:
  `openspec/changes/archive/2026-04-29-add-egress-flow-probe/design.md`
- Spec delta applied to:
  `openspec/specs/egress-probe/spec.md`
