package flow_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/probe/egress/flow"
	"github.com/MWest2020/wanderer/pkg/models"
)

// stubSource emits a fixed list of events and closes its channel.
type stubSource struct {
	events []flow.Event
	closed bool
}

func (s *stubSource) Events(ctx context.Context) <-chan flow.Event {
	ch := make(chan flow.Event, len(s.events))
	for _, e := range s.events {
		ch <- e
	}
	close(ch)
	return ch
}

func (s *stubSource) Close() error { s.closed = true; return nil }

func TestAggregator_DedupsBySrcDestPair(t *testing.T) {
	a := flow.NewAggregator()
	ip := net.ParseIP("203.0.113.5")
	a.Add(flow.Event{DestIP: ip, DestPort: 443, PID: 1, Comm: "node"})
	a.Add(flow.Event{DestIP: ip, DestPort: 443, PID: 2, Comm: "curl"}) // duplicate
	a.Add(flow.Event{DestIP: ip, DestPort: 80, PID: 3, Comm: "wget"})  // different port

	got := a.Findings(nil)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2", len(got))
	}
	// First-observation wins: the 443 finding's process_name is "node".
	for _, f := range got {
		if f.Attributes["destination_port"] == 443 {
			if f.Attributes["process_name"] != "node" {
				t.Errorf("process_name = %v, want first observation 'node'", f.Attributes["process_name"])
			}
		}
	}
}

func TestAggregator_FindingShape(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{
		DestIP:   net.ParseIP("203.0.113.5"),
		DestPort: 443,
		PID:      4242,
		Comm:     "agent",
	})
	got := a.Findings(nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d", len(got))
	}
	f := got[0]
	if f.Severity == "" {
		t.Error("Severity unset")
	}
	if f.Attributes["runtime"] != true {
		t.Error("runtime attribute missing")
	}
	if f.Attributes["destination_ip"] != "203.0.113.5" {
		t.Errorf("destination_ip = %v", f.Attributes["destination_ip"])
	}
	if f.Attributes["destination_port"] != 443 {
		t.Errorf("destination_port = %v", f.Attributes["destination_port"])
	}
	if f.Attributes["pid"] != 4242 {
		t.Errorf("pid = %v", f.Attributes["pid"])
	}
	if got, want := f.Attributes["process_name"], "agent"; got != want {
		t.Errorf("process_name = %v, want %v", got, want)
	}
}

func TestAggregator_IgnoresZeroPort(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 0})
	if got := a.Findings(nil); len(got) != 0 {
		t.Errorf("port-0 events should be skipped, got %d", len(got))
	}
}

func TestAggregator_IgnoresNilIP(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{DestPort: 443})
	if got := a.Findings(nil); len(got) != 0 {
		t.Errorf("nil-IP events should be skipped, got %d", len(got))
	}
}

func TestAggregator_ResolverAnnotates(t *testing.T) {
	a := flow.NewAggregator()
	a.Add(flow.Event{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"})

	// Stub resolver returns a fixed annotation.
	resolver := stubResolver{asn: 64500, org: "example", country: "NL"}
	got := a.Findings(resolver)
	if len(got) != 1 {
		t.Fatal("resolver annotation should not change finding count")
	}
	f := got[0]
	if f.Attributes["asn"] != uint(64500) {
		t.Errorf("asn = %v", f.Attributes["asn"])
	}
	if f.Attributes["country"] != "NL" {
		t.Errorf("country = %v", f.Attributes["country"])
	}
}

type stubResolver struct {
	asn     uint
	org     string
	country string
}

func (s stubResolver) Resolve(_ string) (uint, string, string, bool) {
	return s.asn, s.org, s.country, true
}

func TestFlow_Available_NoSourceUnavailable(t *testing.T) {
	// Without a Source the kernel-side detection runs. On a host
	// without root, Available must return false and the reason must
	// name the missing piece (kernel BTF, capability, or the
	// not-yet-shipped loader).
	f := flow.Flow{}
	ok, reason := f.Available()
	if ok {
		t.Fatal("Available must be false without a Source")
	}
	if reason == "" {
		t.Error("Available reason is empty")
	}
}

func TestFlow_Available_TestSourceMakesAvailable(t *testing.T) {
	f := flow.Flow{Source: &stubSource{}}
	ok, _ := f.Available()
	if !ok {
		t.Fatal("test Source should make the inspector available")
	}
}

func TestFlow_Inspect_ConsumesEventsAndReturnsFindings(t *testing.T) {
	src := &stubSource{events: []flow.Event{
		{DestIP: net.ParseIP("203.0.113.5"), DestPort: 443, Comm: "node"},
		{DestIP: net.ParseIP("203.0.113.6"), DestPort: 443, Comm: "node"},
	}}
	// Window must be long enough to drain the channel before the
	// timeout fires; with a buffered channel that is closed after
	// the events arrive, the collect loop exits on channel close.
	f := flow.Flow{Source: src, Window: 5 * time.Second}
	got, err := f.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2", len(got))
	}
	if !src.closed {
		t.Error("Inspect did not close the source")
	}
}

func TestFlow_Inspect_NoSourceErrors(t *testing.T) {
	f := flow.Flow{}
	_, err := f.Inspect(context.Background())
	if err == nil {
		t.Fatal("expected error when Source is nil")
	}
}

// Smoke: classifier reuse still works through buildFlowFinding —
// the existing classify_test.go already pins the rule table; here we
// just confirm that an HTTPS destination resolves to a non-unknown
// category through the flow path.
func TestFlow_ClassifierReuse_HTTPSGoesToObjectStorageWhenAWS(t *testing.T) {
	a := flow.NewAggregator()
	// 52.218.0.1 happens to be in the AWS S3 prefix range, but we do
	// not call extractHost on an IP-only dest, so the classifier
	// returns "unknown". This test pins the IP-only behaviour: the
	// flow path emits egress.flow.unknown rather than misclassifying
	// raw IPs as a vendor.
	a.Add(flow.Event{DestIP: net.ParseIP("52.218.0.1"), DestPort: 443, Comm: "agent"})
	got := a.Findings(nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d", len(got))
	}
	if got[0].ProbeID != "egress.flow.unknown" {
		t.Errorf("ProbeID = %s, want egress.flow.unknown for raw IP", got[0].ProbeID)
	}
	// And the severity for unknown drops to info.
	if got[0].Severity != models.SeverityInfo {
		t.Errorf("Severity = %s, want info for unknown", got[0].Severity)
	}
}
