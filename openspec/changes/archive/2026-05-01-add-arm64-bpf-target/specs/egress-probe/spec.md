# Delta for egress-probe

## ADDED Requirements

### Requirement: Flow probe ships a BPF object for amd64 and arm64

The egress flow probe SHALL ship a compiled CO-RE BPF object for
both `amd64` and `arm64` Linux build targets, so a `wanderer
agent` binary built for either GOARCH carries a kernel-loadable
BPF program. Selection between the two artefacts SHALL be a
build-time concern resolved by Go's `//go:build` constraints; no
runtime selection logic is required.

#### Scenario: Native amd64 build links the x86 artefact

- **Given** a developer host with `GOARCH=amd64`
- **When** `go build ./...` runs
- **Then** the produced `wanderer agent` binary embeds
  `connect_x86_bpfel.o`

#### Scenario: Cross-built arm64 binary links the arm64 artefact

- **Given** a developer host with `GOARCH=amd64` running
  `GOARCH=arm64 go build ./internal/probe/egress/flow/...`
- **When** the build completes
- **Then** the produced object compiles cleanly, embedding
  `connect_arm64_bpfel.o`
- **And** the build succeeds without invoking the eBPF toolchain

#### Scenario: Both artefacts are reproducible from source

- **Given** the pinned bpf-builder container and an unchanged
  `bpf/connect.bpf.c`
- **When** `./scripts/bpf-build.sh` runs
- **Then** both `connect_x86_bpfel.{go,o}` and
  `connect_arm64_bpfel.{go,o}` are emitted
- **And** their content is byte-identical to the committed
  artefacts (clean `git diff` after the run)
