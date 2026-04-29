//go:build !(linux && amd64)

package flow

import (
	"context"
	"errors"
)

// KernelSource is the no-op variant on platforms where the eBPF
// loader is not built. The bpf2go-generated bindings only target
// linux/amd64 today; ports to arm64 will land alongside an
// additional `bpf2go -target arm64` directive.
type KernelSource struct{}

// NewKernelSource always returns ErrNotSupported on non-Linux /
// non-amd64 builds.
func NewKernelSource() (*KernelSource, error) {
	return nil, errors.New("flow: eBPF kernel source requires linux/amd64")
}

// Events returns a closed channel.
func (s *KernelSource) Events(_ context.Context) <-chan Event {
	ch := make(chan Event)
	close(ch)
	return ch
}

// Close is a no-op.
func (s *KernelSource) Close() error { return nil }
