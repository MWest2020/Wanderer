# Proposal: Egress flow probe (eBPF)

## Intent

The egress probe that landed reads *static* configuration: env
vars, config files, systemd units. It cannot see URLs that an
application assembles at runtime — `https://${REGION}.${VENDOR}/api`
is invisible until the request actually leaves the host. The
egress-probe design called this out as a known gap and named the
follow-up: an eBPF-based flow probe.

This change adds `wanderer agent` flow observation: a short-lived
eBPF program attached to `kprobe/connect` (or the equivalent
tracepoint) that records destination IP+port for outbound
connections, post-process it into the same `egress.*` Finding
shape, and feed the IP probe for ASN/country annotation.

## Scope

**In scope:**

- A new agent inspector under `internal/probe/egress/flow/` that
  uses `github.com/cilium/ebpf` to load a single CO-RE program and
  read connection events for a configured sampling window.
- Findings carry `egress.flow.<category>` with the destination
  host, port, the originating PID name, and a `runtime: true`
  attribute.
- Reuse of the existing classifier and redactor — no parallel
  surface.
- A YAML toggle: `egress.flow: { enabled: false }` — opt-in only,
  never default-on.
- Privilege handling: the program needs `CAP_BPF` /
  `CAP_PERFMON` (kernel 5.8+) or root. The agent emits
  `egress.flow.unavailable` cleanly when those are absent.

**Out of scope:**

- Cross-process correlation (which container made the call).
  Resolution is by PID; container attribution is a follow-up.
- Persistent eBPF programs that survive agent restarts. The
  program loads on agent start and unloads on stop.
- Kernel-version compatibility below 5.8. The CO-RE program
  targets stable BTF.
- Parsing TLS SNI from packets. We see `connect()` destinations,
  not application data.

## DICTU dimensions informed

- **Juridisch** (primary): runtime egress destinations expose
  jurisdiction in a way static configs cannot.
- **Data & AI**: identity-federation calls, telemetry endpoints.

## Passive/active boundary

Strictly local kernel observation. No outbound calls beyond what
the OS already makes; the probe does not initiate any traffic.

## Parallel-safe

Touches `internal/probe/egress/flow/` (new) and one wire-up line
in `cmd/wanderer/agent.go`. No schema changes.
