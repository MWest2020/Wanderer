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
	hosts     []string
	mx        []*net.MX
	ns        []*net.NS
	cname     string
	txt       map[string][]string
	caaByName map[string][]dnsprobe.CAA
	caaErr    error
	caaCalls  []string
	hostsErr  error
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

func (f *fakeResolver) LookupCAA(_ context.Context, name string) ([]dnsprobe.CAA, error) {
	f.caaCalls = append(f.caaCalls, name)
	if f.caaErr != nil {
		return nil, f.caaErr
	}
	return f.caaByName[name], nil
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
		caaByName: map[string][]dnsprobe.CAA{
			"example.nl": {{Flag: 0, Tag: "issue", Value: "letsencrypt.org"}},
		},
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

func caaFinding(findings []models.Finding) (models.Finding, bool) {
	for _, f := range findings {
		if f.ProbeID == "dns.caa" {
			return f, true
		}
	}
	return models.Finding{}, false
}

// TestCAAOwnRecords covers a subdomain with its own CAA records: no
// climb, no inherited_from attribute.
func TestCAAOwnRecords(t *testing.T) {
	r := &fakeResolver{
		caaByName: map[string][]dnsprobe.CAA{
			"iam.voorbeeld.nl": {{Flag: 0, Tag: "issue", Value: "digicert.com"}},
		},
	}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "iam.voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f, ok := caaFinding(findings)
	if !ok {
		t.Fatal("no dns.caa finding")
	}
	if f.Attributes["value"] != "digicert.com" {
		t.Errorf("value = %v, want digicert.com", f.Attributes["value"])
	}
	if _, inherited := f.Attributes["inherited_from"]; inherited {
		t.Errorf("own records should not carry inherited_from, got %v", f.Attributes["inherited_from"])
	}
}

// TestCAAInherited covers a subdomain with no CAA of its own but an
// apex that has records — the case the original bug (LookupCAA always
// returning nil, nil) made indistinguishable from "nowhere at all".
func TestCAAInherited(t *testing.T) {
	r := &fakeResolver{
		caaByName: map[string][]dnsprobe.CAA{
			"voorbeeld.nl": {{Flag: 0, Tag: "issue", Value: "certsign.ro"}},
		},
	}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "iam.voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f, ok := caaFinding(findings)
	if !ok {
		t.Fatal("no dns.caa finding")
	}
	if f.Subject != "iam.voorbeeld.nl" {
		t.Errorf("subject = %q, want iam.voorbeeld.nl (the queried name, not the origin)", f.Subject)
	}
	if f.Attributes["inherited_from"] != "voorbeeld.nl" {
		t.Errorf("inherited_from = %v, want voorbeeld.nl", f.Attributes["inherited_from"])
	}
}

// TestCAANowhere covers a zone with no CAA at any level up to the
// registrable domain: "no CAA records" is only correct once the whole
// chain has been checked, not just the queried name.
func TestCAANowhere(t *testing.T) {
	r := &fakeResolver{caaByName: map[string][]dnsprobe.CAA{}}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "iam.voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f, ok := caaFinding(findings)
	if !ok {
		t.Fatal("no dns.caa finding")
	}
	if f.Attributes["no_answer"] != true {
		t.Errorf("expected no_answer finding, got %v", f.Attributes)
	}
}

// TestCAAAlwaysEmptyResolver plugs in a resolver whose LookupCAA
// silently returns nil, nil for every name — the exact shape of the
// original bug, now used as a stub instead of the live implementation.
// It pins two things the old code got wrong at once: the probe must
// still report "no CAA records" (not fabricate a positive result), and
// it must have actually queried every level up to the registrable
// domain before concluding that — not just the name it was handed.
// The original bug queried once and stopped, which this test would
// have caught via the call count alone.
func TestCAAAlwaysEmptyResolver(t *testing.T) {
	r := &fakeResolver{} // caaByName is nil: every LookupCAA call returns nil, nil
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "a.iam.voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f, ok := caaFinding(findings)
	if !ok {
		t.Fatal("no dns.caa finding")
	}
	if f.Attributes["no_answer"] != true {
		t.Errorf("expected no_answer finding, got %v", f.Attributes)
	}
	want := []string{"a.iam.voorbeeld.nl", "iam.voorbeeld.nl", "voorbeeld.nl"}
	if len(r.caaCalls) != len(want) {
		t.Fatalf("caaCalls = %v, want %v", r.caaCalls, want)
	}
	for i := range want {
		if r.caaCalls[i] != want[i] {
			t.Errorf("caaCalls[%d] = %q, want %q", i, r.caaCalls[i], want[i])
		}
	}
}

func TestCAALookupError(t *testing.T) {
	r := &fakeResolver{caaErr: &net.DNSError{Err: "timeout", IsTimeout: true}}
	p := &dnsprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f, ok := caaFinding(findings)
	if !ok {
		t.Fatal("no dns.caa finding")
	}
	if f.Attributes["kind"] != "timeout" {
		t.Errorf("kind = %v, want timeout", f.Attributes["kind"])
	}
}
