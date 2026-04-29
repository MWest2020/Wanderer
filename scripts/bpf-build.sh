#!/usr/bin/env bash
# Wanderer eBPF builder.
#
# Builds the bpf-builder container (idempotent — only rebuilds on
# Dockerfile change) and runs `go generate ./internal/probe/egress/flow/...`
# inside it so the BPF objects + bpf2go-generated Go sources stay in
# sync with internal/probe/egress/flow/bpf/connect.bpf.c.
#
# Pre-generates `vmlinux.h` from the host's BTF mount so the container
# has the kernel type definitions the CO-RE program references.
#
# Run from the repo root:
#   ./scripts/bpf-build.sh
#
# Requires: docker (or podman wrapping docker), and either
# /sys/kernel/btf/vmlinux on the host (kernel 5.8+ with
# CONFIG_DEBUG_INFO_BTF=y) or a checked-in vmlinux.h.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

IMAGE_TAG="wanderer-bpf-builder"
BPF_DIR="internal/probe/egress/flow/bpf"

echo ">> Building bpf-builder image ($IMAGE_TAG)"
docker build -q -t "$IMAGE_TAG" build/bpf-builder/ >/dev/null

# Generate vmlinux.h on the host (needs /sys/kernel/btf/vmlinux). The
# container has bpftool too, but a kernel mount inside an unprivileged
# container is fragile; doing it on the host once is simpler.
if [[ ! -f "$BPF_DIR/vmlinux.h" ]] || [[ /sys/kernel/btf/vmlinux -nt "$BPF_DIR/vmlinux.h" ]]; then
    if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
        echo "ERROR: /sys/kernel/btf/vmlinux is not readable. Either:" >&2
        echo "  - Run on a kernel 5.8+ host with CONFIG_DEBUG_INFO_BTF=y, or" >&2
        echo "  - Check in a $BPF_DIR/vmlinux.h generated elsewhere." >&2
        exit 1
    fi
    echo ">> Regenerating $BPF_DIR/vmlinux.h from host BTF"
    bpftool btf dump file /sys/kernel/btf/vmlinux format c > "$BPF_DIR/vmlinux.h"
fi

echo ">> Running go generate in container"
# `:z` shared SELinux relabel — safe for repeated dev access; without
# it podman returns EACCES on the mount under SELinux-enforcing hosts
# (RHEL / Fedora / AlmaLinux). The label persists harmlessly for other
# Go tools.
docker run --rm \
    -v "$ROOT:/src:z" \
    -w /src \
    -e GOFLAGS=-mod=mod \
    "$IMAGE_TAG" \
    go generate ./internal/probe/egress/flow/...

echo ">> Done. Generated artifacts:"
find "$BPF_DIR" -maxdepth 1 -mindepth 1 -printf '    %p\n'
find internal/probe/egress/flow -maxdepth 1 -mindepth 1 -name '*_bpfel_*.go' -printf '    %p\n'
find internal/probe/egress/flow -maxdepth 1 -mindepth 1 -name '*_bpfel_*.o' -printf '    %p\n'
find internal/probe/egress/flow -maxdepth 1 -mindepth 1 \( -name '*_bpfeb_*.go' -o -name '*_bpfeb_*.o' \) -printf '    %p\n'
