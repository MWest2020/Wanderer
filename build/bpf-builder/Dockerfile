# Wanderer eBPF builder.
#
# Pinned image for compiling internal/probe/egress/flow/bpf/connect.bpf.c
# via bpf2go. Stays small and reproducible: an operator does not need
# to install clang / llvm / libbpf-devel on the host. The build is
# CGo-free; clang is invoked directly with `-target bpf` so the
# resulting .bpf.o is kernel-portable via CO-RE.
#
# Usage:
#   ./scripts/bpf-build.sh                  # builds image + runs go generate
#   docker run --rm -v $PWD:/src wanderer-bpf-builder go generate ./internal/probe/egress/flow/...
#
# Why Fedora 42: ships clang 18, llvm 18, libbpf-devel 1.4, and a
# matching bpftool in the base repos, so no EPEL pivot. Same eBPF
# ABI lineage as the AlmaLinux 10 host. The container does NOT need
# to match the host kernel — CO-RE handles the kernel-version skew
# at runtime, and vmlinux.h is generated from the host's BTF mount
# (see bpf-build.sh).

FROM fedora:42

RUN dnf install -y --setopt=install_weak_deps=False \
        clang \
        llvm \
        libbpf-devel \
        bpftool \
        kernel-headers \
        make \
        golang \
    && dnf clean all

# Install bpf2go at a pinned version. cilium/ebpf v0.16.0 is the
# stable line as of 2026-04; pin so reproducible regenerations
# stay byte-identical across hosts.
ENV GOFLAGS=-mod=mod
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.16.0
ENV PATH=$PATH:/root/go/bin

WORKDIR /src

# Default command prints the toolchain versions so an operator can
# eyeball that the container is wired correctly. The real entrypoint
# in scripts/bpf-build.sh overrides with `go generate`.
CMD echo "clang:" && clang --version | head -1 && \
    echo "llvm-strip:" && llvm-strip --version | head -1 && \
    echo "bpf2go:" && bpf2go -version 2>&1 | head -1 || true && \
    echo "go:" && go version
