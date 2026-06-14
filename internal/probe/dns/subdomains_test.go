package dns

import (
	"context"
	"errors"
	"net"
	"testing"
)

// stubResolver returns canned responses keyed by exact host.
type stubResolver struct {
	hosts map[string][]string
}

func (s *stubResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	return nil, errors.New("not used here")
}

func (s *stubResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if v, ok := s.hosts[host]; ok {
		return v, nil
	}
	return nil, errors.New("no such host")
}

func (s *stubResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) { return nil, nil }
func (s *stubResolver) LookupNS(_ context.Context, _ string) ([]*net.NS, error) { return nil, nil }
func (s *stubResolver) LookupCNAME(_ context.Context, _ string) (string, error) { return "", nil }
func (s *stubResolver) LookupTXT(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (s *stubResolver) LookupCAA(_ context.Context, _ string) ([]CAA, error)    { return nil, nil }

func TestSubdomainSweep_Hits(t *testing.T) {
	res := &stubResolver{hosts: map[string][]string{
		"www.example.nl":  {"203.0.113.10"},
		"mail.example.nl": {"203.0.113.20"},
	}}
	p := &Probe{Resolver: res}
	got := p.subdomainSweep(context.Background(), "example.nl")
	if len(got) != 2 {
		t.Fatalf("want 2 dns.subdomain Findings, got %d (%+v)", len(got), got)
	}
	for _, f := range got {
		if f.ProbeID != "dns.subdomain" {
			t.Errorf("ProbeID = %s", f.ProbeID)
		}
	}
}

func TestSubdomainSweep_Wildcard(t *testing.T) {
	hosts := map[string][]string{}
	for _, p := range commonPrefixes {
		hosts[p+".example.nl"] = []string{"203.0.113.99"}
	}
	res := &stubResolver{hosts: hosts}
	p := &Probe{Resolver: res}
	got := p.subdomainSweep(context.Background(), "example.nl")
	if len(got) != 1 || got[0].ProbeID != "dns.subdomain.wildcard" {
		t.Fatalf("want exactly one wildcard Finding, got %+v", got)
	}
}

func TestSubdomainSweep_NothingResolves(t *testing.T) {
	p := &Probe{Resolver: &stubResolver{}}
	got := p.subdomainSweep(context.Background(), "example.nl")
	if len(got) != 0 {
		t.Errorf("no resolves should produce no Findings, got %+v", got)
	}
}
