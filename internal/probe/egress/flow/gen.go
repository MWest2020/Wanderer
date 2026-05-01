// Code generation seam for the eBPF flow probe. The directive below
// runs bpf2go (cilium/ebpf) which compiles bpf/connect.bpf.c with
// clang and emits matching Go bindings (`bpf_bpfel_*.go` plus
// `.o` blobs).
//
// Run via `./scripts/bpf-build.sh` so the toolchain (clang, llvm,
// libbpf-devel, bpf2go) lives in the pinned bpf-builder container —
// `go build ./...` on a developer host with no eBPF toolchain still
// works because the generated artifacts are committed.
//
// Re-run after every edit to bpf/connect.bpf.c, or after a host
// kernel update changes the BTF that vmlinux.h is generated from.

package flow

//go:generate bpf2go -cc clang -cflags "-O2 -g -Wall -I bpf" -target amd64,arm64 Connect bpf/connect.bpf.c
