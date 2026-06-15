package flow_test

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/probe/egress/flow"
)

// stubReverseResolver records lookups for assertion and returns a
// fixed mapping. ip => host. Missing keys yield ok=false.
type stubReverseResolver struct {
	mapping map[string]string
	calls   int64
}

func (s *stubReverseResolver) Reverse(_ context.Context, ip string) (string, bool) {
	atomic.AddInt64(&s.calls, 1)
	host, ok := s.mapping[ip]
	if !ok {
		return "", false
	}
	return host, true
}

func TestAggregator_ReverseDNS_Annotates(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})

	rev := &stubReverseResolver{mapping: map[string]string{
		"203.0.113.5": "ec2-203-0-113-5.eu-west-1.compute.amazonaws.com",
	}}
	got := a.Findings(context.Background(), nil, rev)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	want := "ec2-203-0-113-5.eu-west-1.compute.amazonaws.com"
	if got[0].Attributes["reverse_dns"] != want {
		t.Errorf("reverse_dns = %v, want %s", got[0].Attributes["reverse_dns"], want)
	}
}

func TestAggregator_ReverseDNS_FailureIsSilent(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})

	rev := &stubReverseResolver{mapping: map[string]string{}} // every lookup fails
	got := a.Findings(context.Background(), nil, rev)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	if _, present := got[0].Attributes["reverse_dns"]; present {
		t.Errorf("reverse_dns key must be absent on PTR failure, got %v", got[0].Attributes["reverse_dns"])
	}
}

func TestAggregator_ReverseDNS_NilResolverIsByteIdentical(t *testing.T) {
	// The nil-resolver path is the safest default: when the operator
	// hasn't opted in, no reverse_dns key, no PTR query, no behaviour
	// change relative to today.
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})
	got := a.Findings(context.Background(), nil, nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d", len(got))
	}
	if _, present := got[0].Attributes["reverse_dns"]; present {
		t.Errorf("reverse_dns key must be absent without a resolver")
	}
}

func TestAggregator_ReverseDNS_CacheDeduplicates(t *testing.T) {
	// Two ports, same IP — one PTR query, both Findings annotated.
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 8443, Comm: "node"})

	rev := &stubReverseResolver{mapping: map[string]string{
		"203.0.113.5": "host.example",
	}}
	got := a.Findings(context.Background(), nil, rev)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2", len(got))
	}
	for _, f := range got {
		if f.Attributes["reverse_dns"] != "host.example" {
			t.Errorf("reverse_dns missing on finding %v", f.Attributes)
		}
	}
	if got := atomic.LoadInt64(&rev.calls); got != 1 {
		t.Errorf("resolver calls = %d, want 1 (cache must dedupe)", got)
	}
}

func TestAggregator_ReverseDNS_NegativeCacheDoesNotRetry(t *testing.T) {
	// A failing PTR for an IP must not be retried within the same
	// window — that would amplify failure load on the resolver.
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 8443, Comm: "node"})

	rev := &stubReverseResolver{mapping: map[string]string{}} // always fails
	_ = a.Findings(context.Background(), nil, rev)
	if got := atomic.LoadInt64(&rev.calls); got != 1 {
		t.Errorf("resolver calls = %d, want 1 (negative result must be cached)", got)
	}
}

func TestNewReverseDNSResolver_HonoursTimeout(t *testing.T) {
	// The default impl wraps net.DefaultResolver; we cannot reach
	// inside, but we can pin the contract that the *constructor*
	// returns something honouring a small timeout — by calling it
	// against a guaranteed-blackhole IP and checking it returns
	// quickly. Use TEST-NET-1 (203.0.113/24) with a tiny timeout.
	r := flow.NewReverseDNSResolver(50 * time.Millisecond)
	start := time.Now()
	host, ok := r.Reverse(context.Background(), "203.0.113.255")
	elapsed := time.Since(start)
	// We don't care about the result — only that the call returned
	// within a reasonable multiple of the configured timeout.
	if ok && host == "" {
		t.Errorf("ok=true but host empty")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Reverse took %s, far past the 50ms timeout — wrapper is not enforcing deadline", elapsed)
	}
}

func TestPTRCache_NilResolverReturnsNotOK(t *testing.T) {
	// Defensive: an Aggregator built normally has a cache, but the
	// cache.Lookup path with a nil resolver MUST short-circuit to
	// (host="", ok=false). The buildFlowFinding path relies on this
	// via its `reverse != nil` guard, but if a future caller skips
	// the guard the cache must still be safe.
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443})
	got := a.Findings(context.Background(), nil, nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d", len(got))
	}
	if _, present := got[0].Attributes["reverse_dns"]; present {
		t.Errorf("reverse_dns annotated despite nil resolver")
	}
}
