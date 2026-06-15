package dns_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	dnsprobe "github.com/MWest2020/wanderer/internal/probe/dns"
	"github.com/MWest2020/wanderer/pkg/models"
)

type fakeResolver struct {
	hosts    []string
	mx       []*net.MX
	ns       []*net.NS
	cname    string
	txt      map[string][]string
	caa      []dnsprobe.CAA
	hostsErr error
}

func (f *fakeResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	if f.hostsErr != nil {
		return nil, f.hostsErr
	}
	return f.hosts, nil
}

func (f *fakeResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return f.mx, nil
}

func (f *fakeResolver) LookupNS(_ context.Context, _ string) ([]*net.NS, error) {
	return f.ns, nil
}

func (f *fakeResolver) LookupCNAME(_ context.Context, _ string) (string, error) {
	return f.cname, nil
}

func (f *fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	return f.txt[name], nil
}

func (f *fakeResolver) LookupCAA(_ context.Context, _ string) ([]dnsprobe.CAA, error) {
	return f.caa, nil
}

func TestHappyPath(t *testing.T) {
	r := &fakeResolver{
		hosts: []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"},
		mx: []*net.MX{
			{Host: "mail.example.nl.", Pref: 10},
			{Host: "alt.example.nl.", Pref: 20},
		},
		ns: []*net.NS{{Host: "ns1.example.nl."}, {Host: "ns2.example.nl."}},
		txt: map[string][]string{
			"example.nl":        {"v=spf1 include:_spf.example.net ~all"},
			"_dmarc.example.nl": {"v=DMARC1; p=reject; rua=mailto:dmarc@example.nl"},
		},
		caa: []dnsprobe.CAA{{Flag: 0, Tag: "issue", Value: "letsencrypt.org"}},
	}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	count := map[string]int{}
	for _, f := range findings {
		count[f.ProbeID]++
	}
	if count["dns.a"] == 0 {
		t.Error("no A finding")
	}
	if count["dns.aaaa"] == 0 {
		t.Error("no AAAA finding")
	}
	if count["dns.mx"] != 2 {
		t.Errorf("mx findings = %d, want 2", count["dns.mx"])
	}
	if count["dns.ns"] != 2 {
		t.Errorf("ns findings = %d, want 2", count["dns.ns"])
	}
	if count["dns.txt.spf"] != 1 {
		t.Errorf("spf findings = %d, want 1", count["dns.txt.spf"])
	}
	if count["dns.txt.dmarc"] != 1 {
		t.Errorf("dmarc findings = %d, want 1", count["dns.txt.dmarc"])
	}
	if count["dns.caa"] != 1 {
		t.Errorf("caa findings = %d, want 1", count["dns.caa"])
	}
}

func TestNXDOMAIN(t *testing.T) {
	r := &fakeResolver{
		hostsErr: &net.DNSError{Err: "no such host", IsNotFound: true},
	}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "nope.invalid"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.ProbeID == "dns.a" {
			if f.Attributes["kind"] != "nxdomain" {
				t.Errorf("kind = %v, want nxdomain", f.Attributes["kind"])
			}
			found = true
		}
	}
	if !found {
		t.Error("no dns.a finding on NXDOMAIN")
	}
}

func TestTimeout(t *testing.T) {
	r := &fakeResolver{hostsErr: &net.DNSError{Err: "timeout", IsTimeout: true}}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "slow.example"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var seen bool
	for _, f := range findings {
		if f.ProbeID == "dns.a" && f.Attributes["kind"] == "timeout" {
			seen = true
		}
	}
	if !seen {
		t.Error("expected timeout kind on A lookup")
	}
}

func TestResolverNil(t *testing.T) {
	p := &dnsprobe.Probe{}
	_, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err == nil || !errors.Is(err, err) {
		t.Errorf("expected error for nil resolver, got %v", err)
	}
}
