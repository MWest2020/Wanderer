# Design: arm64 BPF build target

## How bpf2go selects the artefact

`cilium/ebpf`'s `bpf2go` accepts a comma-separated `-target` list
and emits one `<name>_<arch>_bpfel.{go,o}` pair per target,
gated by `//go:build` constraints in the generated `.go` files:

- `connect_x86_bpfel.go` carries `//go:build 386 || amd64`
- `connect_arm64_bpfel.go` carries `//go:build arm64`

A Go build for any given GOARCH includes exactly one of those
files; the matching `.o` is embedded via `//go:embed`. Selection
is a build-time concern, not a runtime one — there is no
multi-blob loader and no per-host detection logic to add.

## Why we keep both blobs in the repo

ADR-0010 already documents the choice: the compiled BPF object
ships in version control so `go build ./...` works on developer
hosts with no eBPF toolchain. That rationale carries forward —
both blobs ship, both stay reproducible by re-running
`./scripts/bpf-build.sh` inside the pinned Fedora-42 container,
and CI continues to do plain `go build` on a vanilla runner.

## Reproducibility

The Fedora-42 builder pins `clang 18`, `llvm 18`, and
`bpf2go@v0.16.0`. Adding a target does not change those pins.
Re-running the script on the same host produces the same byte
output for both architectures — the verification step is a `git
diff` against the regenerated artefacts; if that is empty after
re-run, the build is reproducible.

## What can go wrong, and how we catch it

- **Artefact missing the build constraint.** If `bpf2go` emitted a
  `.go` file without the right `//go:build` line, both
  architectures would compile both blobs and the linker would
  blow up. Caught by `go build ./...` (native) plus
  `GOARCH=arm64 go build ./internal/probe/egress/flow/...`
  (cross). Either failure is a hard stop.
- **Missing CO-RE relocation.** `bpf2go` invokes clang against
  the same `connect.bpf.c` source for both arches; the BPF
  bytecode is target-independent above the verifier. CO-RE
  relocations are resolved at load time using the running
  kernel's BTF, regardless of which `bpfel` blob shipped. So a
  CO-RE bug shows up the same way on both arches; this change
  does not introduce a new failure mode.
- **arm64-specific verifier rejections.** Kernel verifier rules
  are largely arch-independent for tracepoint programs of this
  shape (read sockaddr, emit perf event). Real arm64 verification
  happens on a real arm64 kernel — out of scope here, captured
  in the existing "loader integration test" deferred follow-up.

## Why no CI matrix in this change

Adding a `GOARCH=arm64` job to CI is an operational decision with
its own ownership: it widens the runner footprint, may want a
qemu fallback, and intersects with the broader CI strategy for
the agent. We treat it as a separate proposal so this change
remains scoped to the build matrix the developer host produces.

The cross-arch sanity check at change-landing time (run by hand
once) is enough to prove the artefact compiles.
