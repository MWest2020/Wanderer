# Proposal: arm64 BPF build target for the flow probe

## Intent

The egress flow probe ships a CO-RE BPF program built for x86_64
only:

```
//go:generate bpf2go ... -target amd64 Connect bpf/connect.bpf.c
```

ADR-0010 named "arm64 target — one-line change to `gen.go`; covered
by the same builder container" as deferred work. This change closes
that item.

After this change `bpf2go` emits a second pair of artefacts —
`connect_arm64_bpfel.{go,o}` — alongside the existing
`connect_x86_bpfel.{go,o}`. `cilium/ebpf`'s codegen selects the
right blob via Go build constraints (`//go:build amd64` versus
`//go:build arm64`), so a Go binary built for `GOARCH=arm64` ships
the arm64-targeted BPF object and one built for `GOARCH=amd64`
ships the x86 object. No runtime selection logic; it falls out of
`go build`.

## Scope

**In scope:**
- `internal/probe/egress/flow/gen.go`: `-target amd64` →
  `-target amd64,arm64`.
- `./scripts/bpf-build.sh` re-run inside the pinned bpf-builder
  container; the resulting `connect_arm64_bpfel.{go,o}` is
  committed alongside the existing x86 pair.
- A cross-arch sanity check: `GOARCH=arm64 go build
  ./internal/probe/egress/flow/...` compiles the arm64 binding
  cleanly on the developer host (x86_64), proving the artefact
  is wired into the package.
- ADR-0010 addendum recording the landing.
- `docs/egress.md` "Rebuilding the BPF object" updated to mention
  the second artefact pair.

**Out of scope:**
- Big-endian (`bpfeb`) variants. Linux on big-endian is a
  vanishingly small deployment surface for this probe; we ship
  little-endian only on both archs.
- arm32 / armv7. The agent's broader Go toolchain has not been
  certified there.
- A CI matrix that runs `GOARCH=arm64 go build` on every PR. The
  cross-arch build is a one-shot verification at change time;
  a CI matrix is a separate operational decision.
- Loader integration test on a real arm64 kernel — same constraint
  as the x86 loader: needs root + a kernel host, not in CI.

## Why opt-out / opt-in does not apply

This change does not introduce a new behaviour or a new toggle —
it widens the build matrix of an existing probe. An operator who
previously ran the agent on an arm64 host got an x86 artefact
that the kernel verifier rejected; after this change the same
operator gets a working probe with no config change. Strictly an
expansion, not a feature.

## Wand dimensions informed

Indirect — the flow probe's runtime evidence reaches more
deployment surface, so the assessor's *Juridisch* dimension stops
being silently blind on arm64 perimeter hosts.

## Passive / active boundary

Build-time only. Runtime behaviour is unchanged on x86; on arm64
the existing flow-probe contract now applies where it previously
emitted `egress.flow.unavailable` with a verifier error.

## Parallel-safe

Touches `internal/probe/egress/flow/gen.go` (one line), the two
generated artefacts (machine-written), one ADR addendum, and one
docs paragraph. No schema changes, no DB migration, no public-API
surface change.
