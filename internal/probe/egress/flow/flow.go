// Package flow is the eBPF-based egress flow probe. It records
// outbound connect() destinations during a configured sampling window
// and emits one egress.flow.<category> Finding per unique
// (destination_ip, destination_port) pair, reusing the existing
// classifier and redactor so the on-wire shape matches the static
// egress probe.
//
// The inspector is opt-in (see internal/agent/config.go::EgressFlow)
// and never default-on: an operator must consciously accept the
// kernel-level capability cost before any kernel attach happens.
//
// As of 2026-04-29 this package ships the userspace half of the
// design — the Aggregator, the Inspector surface, and the
// classifier-reuse plumbing — but does NOT yet ship a compiled eBPF
// object. The build environment for the agent must add a clang/llvm
// toolchain and a `go generate` step before the kernel attach lands;
// see docs/decisions/0010-ebpf-flow-probe.md (ADR) for the kernel-
// version contract and the deferred work. Until that lands,
// Available() returns false with a reason that names the missing
// piece, which satisfies the "graceful unavailability" scenario the
// egress-probe spec requires.
package flow

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MWest2020/wanderer/internal/probe/egress"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Event is one connect() observation as the userspace half of the
// probe sees it. The kernel side fills the same shape via a perf
// ring buffer; tests inject Events directly through an EventSource.
type Event struct {
	DestIP   net.IP
	DestPort uint16
	PID      uint32
	Comm     string
}

// EventSource is the seam between the kernel-attached eBPF program
// (which is not compiled into this build) and the userspace
// aggregator. A test source feeds synthetic events; the production
// source — once shipped — wraps a perf event reader from
// cilium/ebpf.
type EventSource interface {
	// Events returns a channel that emits one Event per observed
	// connect(). The channel closes when the source is done.
	Events(ctx context.Context) <-chan Event
	// Close releases any kernel resources (perf ring, attached
	// program). Safe to call exactly once.
	Close() error
}

// Flow is the inspector. It is registered only when the operator
// flips egress.flow.enabled in the agent config; otherwise the
// inventory orchestrator does not see it at all and the spec's "no
// egress.flow.* findings on default config" scenario holds.
type Flow struct {
	// Window is the sampling window. Defaults to 60 seconds.
	Window time.Duration
	// Source overrides the kernel event source. Used by tests; in
	// production this is left nil and the inspector reports
	// Available()=false until the eBPF object lands.
	Source EventSource
	// Resolver annotates each Finding with ASN/country attributes
	// when wired (the existing GeoLite2 path).
	Resolver egress.HostResolver
}

// ID is the inspector identifier used in ProbeID prefixes.
func (Flow) ID() string { return "egress.flow" }

// Available reports whether the flow probe can run on the current
// host. The check is layered: OS, kernel BTF, capability, then the
// presence of a usable EventSource. Each missing piece produces a
// distinct reason so an operator can act.
func (f Flow) Available() (bool, string) {
	// Test seam: if a synthetic source is wired, the inspector is
	// available regardless of kernel state.
	if f.Source != nil {
		return true, ""
	}
	if runtime.GOOS != "linux" {
		return false, "eBPF flow probe requires Linux"
	}
	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		return false, "eBPF flow probe requires kernel BTF (kernel 5.8+ with CONFIG_DEBUG_INFO_BTF=y)"
	}
	if os.Geteuid() != 0 && !hasBPFCap() {
		return false, "eBPF flow probe requires CAP_BPF + CAP_PERFMON (or root)"
	}
	// Kernel ready, capability ready, no event source. The eBPF
	// object is not yet shipped in this build; the inspector cannot
	// load a program.
	return false, "eBPF flow loader not yet shipped in this build — install the bpf2go toolchain and rebuild (see ADR-0010)"
}

// Run is the agent-loop integration point. It runs the
// Available-then-Inspect pattern and tags every resulting Finding
// with SourceModus = egress, so the agent can simply append the
// returned slice to its existing perimeter / inventory / static-
// egress findings. When the inspector is unavailable, Run emits
// exactly one egress.flow.unavailable Finding so the absence is
// auditable in the store.
func (f Flow) Run(ctx context.Context) []models.Finding {
	host := flowHostname()
	if ok, reason := f.Available(); !ok {
		return []models.Finding{{
			ProbeID:     "egress.flow.unavailable",
			SourceModus: models.SourceModusEgress,
			Subject:     host,
			Severity:    models.SeverityInfo,
			Attributes:  map[string]any{"reason": reason, "unavailable": true},
		}}
	}
	findings, err := f.Inspect(ctx)
	if err != nil {
		return []models.Finding{{
			ProbeID:     "egress.flow.error",
			SourceModus: models.SourceModusEgress,
			Subject:     host,
			Severity:    models.SeverityInfo,
			Attributes:  map[string]any{"error": err.Error()},
		}}
	}
	for i := range findings {
		findings[i].SourceModus = models.SourceModusEgress
		if findings[i].Subject == "" {
			findings[i].Subject = host
		}
	}
	return findings
}

func flowHostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}

// Inspect drains the EventSource for the configured window, hands
// each unique (destination_ip, destination_port) pair to the
// aggregator, and returns the resulting Findings. Available()
// guards the call site, so on a host without a Source / kernel /
// capability we never reach this method.
func (f Flow) Inspect(ctx context.Context) ([]models.Finding, error) {
	if f.Source == nil {
		return nil, errors.New("egress.flow: no event source — kernel attach not implemented in this build")
	}
	window := f.Window
	if window <= 0 {
		window = 60 * time.Second
	}
	subCtx, cancel := context.WithTimeout(ctx, window)
	defer cancel()

	agg := NewAggregator()
	events := f.Source.Events(subCtx)
collect:
	for {
		select {
		case <-subCtx.Done():
			break collect
		case ev, ok := <-events:
			if !ok {
				break collect
			}
			agg.Add(ev)
		}
	}
	if err := f.Source.Close(); err != nil {
		// Close error is informational, not fatal; surface it on the
		// agent's slog and continue with the findings we have.
		_ = err
	}
	return agg.Findings(f.Resolver), nil
}

// hasBPFCap checks for CAP_BPF + CAP_PERFMON on the current process.
// The implementation is a placeholder: the standard library does not
// expose linux capabilities, and pulling in a CGo capability library
// for a single check is out of scope for the current change. Until
// the eBPF loader lands (and brings a real capability check via
// cilium/ebpf), root remains the only signal we honour.
func hasBPFCap() bool { return false }

// ---- Aggregator ----

// Aggregator deduplicates events by (destination_ip,
// destination_port) within a window. The first observation per pair
// wins (so the recorded process_name is the first program seen
// connecting to that destination, which is more useful for
// jurisdictional review than the last).
type Aggregator struct {
	mu   sync.Mutex
	seen map[string]Event
}

// NewAggregator returns an empty Aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{seen: map[string]Event{}}
}

// Add records one event. If the (DestIP, DestPort) pair has already
// been observed in this window, Add is a no-op so a tight loop of
// connects to the same destination produces exactly one Finding.
func (a *Aggregator) Add(ev Event) {
	if ev.DestIP == nil || ev.DestPort == 0 {
		return
	}
	key := flowKey(ev)
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.seen[key]; !ok {
		a.seen[key] = ev
	}
}

// Findings emits one Finding per unique destination. The resolver,
// when non-nil, annotates each Finding with ASN/country attributes
// using the same GeoLite2 path the static egress probe uses.
func (a *Aggregator) Findings(resolver egress.HostResolver) []models.Finding {
	a.mu.Lock()
	uniques := make([]Event, 0, len(a.seen))
	for _, ev := range a.seen {
		uniques = append(uniques, ev)
	}
	a.mu.Unlock()
	sort.Slice(uniques, func(i, j int) bool {
		return flowKey(uniques[i]) < flowKey(uniques[j])
	})

	out := make([]models.Finding, 0, len(uniques))
	for _, ev := range uniques {
		out = append(out, buildFlowFinding(ev, resolver))
	}
	return out
}

func flowKey(ev Event) string {
	return fmt.Sprintf("%s:%d", ev.DestIP.String(), ev.DestPort)
}

// buildFlowFinding hands the destination off to the existing
// classifier so the wire format stays consistent with the static
// egress probe. We synthesise a fake "key" name (`flow.connect`) so
// classifiers that branch on key still get a stable input.
func buildFlowFinding(ev Event, resolver egress.HostResolver) models.Finding {
	target := ev.DestIP.String()
	cls := egress.Classify("flow.connect", target)
	probeID := "egress.flow." + cls.Category
	if cls.Category == "" || cls.Category == "unknown" {
		probeID = "egress.flow.unknown"
	}
	severity := models.SeverityObservation
	if cls.Category == "unknown" {
		severity = models.SeverityInfo
	}
	subject := target
	if cls.Host != "" {
		subject = cls.Host
	}
	attrs := map[string]any{
		"runtime":          true,
		"destination_ip":   ev.DestIP.String(),
		"destination_port": int(ev.DestPort),
		"classifier_rule":  cls.Rule,
		"confidence":       string(cls.Confidence),
	}
	if ev.PID != 0 {
		attrs["pid"] = int(ev.PID)
	}
	if ev.Comm != "" {
		attrs["process_name"] = strings.TrimRight(ev.Comm, "\x00")
	}
	if cls.Provider != "" {
		attrs["provider"] = cls.Provider
	}
	if cls.Region != "" {
		attrs["region"] = cls.Region
	}
	if resolver != nil {
		if asn, org, country, ok := resolver.Resolve(target); ok {
			attrs["asn"] = asn
			attrs["organisation"] = org
			attrs["country"] = country
		}
	}
	return models.Finding{
		ProbeID:       probeID,
		DimensionHint: cls.Dimension,
		Subject:       subject,
		Severity:      severity,
		Attributes:    attrs,
	}
}
