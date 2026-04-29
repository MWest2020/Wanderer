//go:build linux && amd64

package flow

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
)

// kernelEvent mirrors `struct flow_event` in
// internal/probe/egress/flow/bpf/connect.bpf.c. Field order and
// padding must match exactly — the kernel writes raw struct bytes
// into the perf ring buffer and we decode them little-endian.
type kernelEvent struct {
	PID    uint32
	Family uint8
	Pad    [3]uint8
	DPort  uint16
	Daddr  [16]byte
	Comm   [16]byte
}

// KernelSource is an EventSource backed by the bpf2go-generated
// program. It loads the embedded BPF object, attaches it to the
// `syscalls/sys_enter_connect` tracepoint, and reads kernel events
// from a perf ring buffer until Close() is called.
type KernelSource struct {
	objs    ConnectObjects
	tp      link.Link
	reader  *perf.Reader
}

// NewKernelSource boots the eBPF program. The caller is responsible
// for invoking Close() to release kernel resources. Failure modes
// (missing BTF, insufficient capability, verifier rejection) are
// propagated as errors; the agent's Flow.Run path translates a
// construction error into an `egress.flow.unavailable` Finding.
func NewKernelSource() (*KernelSource, error) {
	// Loading a BPF program needs RLIMIT_MEMLOCK; cilium/ebpf's
	// helper bumps it to infinity once per process. Idempotent.
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("flow: rlimit memlock: %w", err)
	}
	s := &KernelSource{}
	if err := LoadConnectObjects(&s.objs, nil); err != nil {
		return nil, fmt.Errorf("flow: load BPF objects: %w", err)
	}
	tp, err := link.Tracepoint("syscalls", "sys_enter_connect", s.objs.TraceConnect, nil)
	if err != nil {
		_ = s.objs.Close()
		return nil, fmt.Errorf("flow: attach tracepoint: %w", err)
	}
	s.tp = tp
	rd, err := perf.NewReader(s.objs.Events, 16*1024)
	if err != nil {
		_ = tp.Close()
		_ = s.objs.Close()
		return nil, fmt.Errorf("flow: open perf reader: %w", err)
	}
	s.reader = rd
	return s, nil
}

// Events returns a channel that emits one Event per non-AF_UNIX
// connect() the kernel observed. The channel closes when ctx is
// cancelled. The KernelSource itself stays alive across calls so
// the BPF program is loaded once and reused per tick; Close()
// tears the whole thing down.
func (s *KernelSource) Events(ctx context.Context) <-chan Event {
	out := make(chan Event)
	go func() {
		defer close(out)
		// Watcher goroutine: when ctx is cancelled, set a deadline
		// in the past so the blocking Read() returns immediately.
		stop := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				s.reader.SetDeadline(time.Unix(1, 0))
			case <-stop:
			}
		}()
		defer close(stop)

		for {
			record, err := s.reader.Read()
			if err != nil {
				if errors.Is(err, perf.ErrClosed) || errors.Is(err, os.ErrDeadlineExceeded) {
					return
				}
				if ctx.Err() != nil {
					return
				}
				continue
			}
			if record.LostSamples != 0 {
				continue
			}
			ev := decodeKernelEvent(record.RawSample)
			if ev == nil {
				continue
			}
			select {
			case out <- *ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Close releases kernel resources. Calling Close before Events has
// returned causes the in-flight Read to fail with ErrClosed, which
// the goroutine handles cleanly. Safe to call exactly once.
func (s *KernelSource) Close() error {
	var errs []error
	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("perf reader close: %w", err))
		}
	}
	if s.tp != nil {
		if err := s.tp.Close(); err != nil {
			errs = append(errs, fmt.Errorf("tracepoint close: %w", err))
		}
	}
	if err := s.objs.Close(); err != nil {
		errs = append(errs, fmt.Errorf("BPF objects close: %w", err))
	}
	return errors.Join(errs...)
}

func decodeKernelEvent(raw []byte) *Event {
	if len(raw) < int(binary.Size(kernelEvent{})) {
		return nil
	}
	var ke kernelEvent
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &ke); err != nil {
		return nil
	}
	var ip net.IP
	switch ke.Family {
	case 2: // AF_INET
		ip = net.IP(append([]byte(nil), ke.Daddr[:4]...))
	case 10: // AF_INET6
		ip = net.IP(append([]byte(nil), ke.Daddr[:16]...))
	default:
		return nil
	}
	if ip == nil || ke.DPort == 0 {
		return nil
	}
	return &Event{
		DestIP:   ip,
		DestPort: ke.DPort,
		PID:      ke.PID,
		Comm:     trimNul(ke.Comm[:]),
	}
}

func trimNul(b []byte) string {
	return strings.TrimRight(string(b), "\x00")
}
