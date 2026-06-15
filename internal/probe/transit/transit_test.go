package transit

import (
	"context"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestParseTrace_Tracepath(t *testing.T) {
	out := `` +
		" 1?: [LOCALHOST]                      pmtu 1500\n" +
		" 1:  192.168.1.1                                          2.399ms \n" +
		" 1:  192.168.1.1                                          1.173ms \n" +
		" 2:  no reply\n" +
		" 5:  213.46.182.254                                       12.461ms asymm  6 \n" +
		" 6:  129.250.2.1                                          10.922ms \n" +
		"10:  81.24.6.82                                           13.644ms \n" +
		"11:  no reply\n" +
		"     Resume: pmtu 1500 hops 10 back 54\n"
	hops := parseTrace(out)
	want := []Hop{
		{Num: 1, IP: "192.168.1.1", RTTms: 2.399},
		{Num: 2, NoReply: true},
		{Num: 5, IP: "213.46.182.254", RTTms: 12.461},
		{Num: 6, IP: "129.250.2.1", RTTms: 10.922},
		{Num: 10, IP: "81.24.6.82", RTTms: 13.644},
		{Num: 11, NoReply: true},
	}
	assertHops(t, hops, want)
}

func TestParseTrace_Traceroute(t *testing.T) {
	out := `traceroute to 1.2.3.4 (1.2.3.4), 30 hops max, 60 byte packets
 1  192.168.1.1  0.456 ms
 2  * * *
 5  213.46.182.254  12.400 ms
`
	hops := parseTrace(out)
	want := []Hop{
		{Num: 1, IP: "192.168.1.1", RTTms: 0.456},
		{Num: 2, NoReply: true},
		{Num: 5, IP: "213.46.182.254", RTTms: 12.4},
	}
	assertHops(t, hops, want)
}

func assertHops(t *testing.T, got, want []Hop) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("hop count = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Num != want[i].Num || got[i].IP != want[i].IP || got[i].NoReply != want[i].NoReply {
			t.Errorf("hop[%d] = %+v, want %+v", i, got[i], want[i])
		}
		if got[i].RTTms != want[i].RTTms {
			t.Errorf("hop[%d] rtt = %v, want %v", i, got[i].RTTms, want[i].RTTms)
		}
	}
}

// fakeTracer returns canned hops for the Run test.
type fakeTracer struct {
	hops []Hop
	up   bool
}

func (f fakeTracer) Available() bool { return f.up }
func (f fakeTracer) Trace(context.Context, string, int) ([]Hop, error) {
	return f.hops, nil
}

func findingByProbe(fs []models.Finding, id string) *models.Finding {
	for i := range fs {
		if fs[i].ProbeID == id {
			return &fs[i]
		}
	}
	return nil
}

func TestRun_Unavailable(t *testing.T) {
	p := &Probe{Tracer: fakeTracer{up: false}}
	fs, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if findingByProbe(fs, "transit.unavailable") == nil {
		t.Fatalf("expected transit.unavailable, got %+v", fs)
	}
}

func TestRun_EmitsHopsAndAggregate(t *testing.T) {
	// Resolver-free path: a target that is already an IP literal so
	// LookupHost returns it without DNS.
	p := &Probe{
		Tracer: fakeTracer{up: true, hops: []Hop{
			{Num: 1, IP: "192.168.1.1", RTTms: 1.2},
			{Num: 2, NoReply: true},
			{Num: 3, IP: "81.24.6.82", RTTms: 13.6},
		}},
		// Geo nil → no ASN/country enrichment; rDNS may or may not
		// resolve in the sandbox, which is fine (best-effort).
	}
	fs, err := p.Run(context.Background(), models.Target{Domain: "81.24.6.82"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	hops := 0
	for _, f := range fs {
		if f.ProbeID == "transit.hop" {
			hops++
		}
	}
	if hops != 3 {
		t.Errorf("transit.hop count = %d, want 3", hops)
	}
	path := findingByProbe(fs, "transit.path")
	if path == nil {
		t.Fatal("expected a transit.path aggregate finding")
	}
	if path.Attributes["dest_ip"] != "81.24.6.82" {
		t.Errorf("dest_ip = %v, want 81.24.6.82", path.Attributes["dest_ip"])
	}
	if path.Attributes["hops_responded"] != 2 {
		t.Errorf("hops_responded = %v, want 2", path.Attributes["hops_responded"])
	}
}
