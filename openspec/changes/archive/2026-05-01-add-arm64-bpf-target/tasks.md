# Tasks: arm64 BPF build target

## 1. Build matrix

- [x] 1.1 `internal/probe/egress/flow/gen.go`: switch `-target amd64`
  to `-target amd64,arm64`
- [x] 1.2 Re-run `./scripts/bpf-build.sh` in the pinned bpf-builder
  container
- [x] 1.3 Commit the regenerated `connect_arm64_bpfel.{go,o}`
  alongside the unchanged `connect_x86_bpfel.{go,o}`

## 2. Verification

- [x] 2.1 `go build ./...` clean on the developer host (native amd64)
- [x] 2.2 `GOARCH=arm64 go build ./internal/probe/egress/flow/...`
  clean — proves the arm64 artefact is wired in
- [x] 2.3 `go test ./...` clean

## 3. Docs

- [x] 3.1 ADR-0010 addendum noting arm64 landed, points at the
  spec delta
- [x] 3.2 `docs/egress.md` rebuilding-the-bpf-object snippet
  updated to mention the second artefact pair
- [x] 3.3 `CHANGELOG.md` entry under `### Added`
