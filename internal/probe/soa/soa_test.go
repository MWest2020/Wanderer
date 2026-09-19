package soa_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	soaprobe "github.com/MWest2020/wanderer/internal/probe/soa"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubResolver struct {
	mname, rname string
	soaErr       error

	hosts    []string
	hostsErr error
	mx       []*net.MX
	mxErr    error
}

func (s *stubResolver) LookupSOA(_ context.Context, _ string) (string, string, error) {
	return s.mname, s.rname, s.soaErr
}

func (s *stubResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return s.hosts, s.hostsErr
}

func (s *stubResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return s.mx, s.mxErr
}

func findingByProbeID(findings []models.Finding, id string) *models.Finding {
	for i := range findings {
		if findings[i].ProbeID == id {
			return &findings[i]
		}
	}
	return nil
}

func TestRun_NormalZone(t *testing.T) {
	r := &stubResolver{
		mname: "ns1.voorbeeld.nl",
		rname: "hostmaster.voorbeeld.nl",
		hosts: []string{"192.0.2.1"},
		mx:    []*net.MX{{Host: "mail.voorbeeld.nl.", Pref: 10}},
	}
	p := &soaprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "dns.soa")
	if f == nil {
		t.Fatalf("no dns.soa finding, got %+v", findings)
	}
	if f.DimensionHint != models.DimensionAccountability {
		t.Errorf("dimension = %q, want accountability", f.DimensionHint)
	}
	if f.Attributes["mname"] != "ns1.voorbeeld.nl" {
		t.Errorf("mname = %v", f.Attributes["mname"])
	}
	if f.Attributes["rname"] != "hostmaster.voorbeeld.nl" {
		t.Errorf("rname = %v", f.Attributes["rname"])
	}
	if f.Attributes["mailbox"] != "hostmaster@voorbeeld.nl" {
		t.Errorf("mailbox = %v", f.Attributes["mailbox"])
	}
	if f.Attributes["mailbox_domain"] != "voorbeeld.nl" {
		t.Errorf("mailbox_domain = %v", f.Attributes["mailbox_domain"])
	}
	if f.Attributes["mailbox_domain_resolves"] != true {
		t.Errorf("mailbox_domain_resolves = %v, want true", f.Attributes["mailbox_domain_resolves"])
	}
	if f.Attributes["mailbox_domain_has_mx"] != true {
		t.Errorf("mailbox_domain_has_mx = %v, want true", f.Attributes["mailbox_domain_has_mx"])
	}
}

func TestRun_EscapedDotRNAME(t *testing.T) {
	r := &stubResolver{
		mname:    "ns1.voorbeeld.nl",
		rname:    `john\.doe.voorbeeld.nl`,
		hostsErr: errors.New("no such host"),
		mxErr:    errors.New("no such host"),
	}
	p := &soaprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "dns.soa")
	if f == nil {
		t.Fatalf("no dns.soa finding, got %+v", findings)
	}
	if f.Attributes["mailbox"] != "john.doe@voorbeeld.nl" {
		t.Errorf("mailbox = %v, want john.doe@voorbeeld.nl", f.Attributes["mailbox"])
	}
	if f.Attributes["mailbox_domain"] != "voorbeeld.nl" {
		t.Errorf("mailbox_domain = %v", f.Attributes["mailbox_domain"])
	}
	if f.Attributes["mailbox_domain_resolves"] != false {
		t.Errorf("mailbox_domain_resolves = %v, want false", f.Attributes["mailbox_domain_resolves"])
	}
	if f.Attributes["mailbox_domain_has_mx"] != false {
		t.Errorf("mailbox_domain_has_mx = %v, want false", f.Attributes["mailbox_domain_has_mx"])
	}
}

func TestRun_LameDelegationTimeout(t *testing.T) {
	r := &stubResolver{soaErr: context.DeadlineExceeded}
	p := &soaprobe.Probe{Resolver: r}
	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(findings) != 1 || findings[0].ProbeID != "dns.soa.unavailable" {
		t.Fatalf("expected single dns.soa.unavailable finding, got %+v", findings)
	}
}

func TestSplitRNAME(t *testing.T) {
	// splitRNAME is unexported; exercise it through Run's mailbox
	// attribute rather than importing internals from _test package.
	cases := []struct {
		name        string
		rname       string
		wantErr     bool
		wantMailbox string
	}{
		{name: "normal", rname: "hostmaster.voorbeeld.nl", wantMailbox: "hostmaster@voorbeeld.nl"},
		{name: "escaped dot", rname: `john\.doe.voorbeeld.nl`, wantMailbox: "john.doe@voorbeeld.nl"},
		{name: "no dot at all", rname: "hostmaster", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &stubResolver{mname: "ns1.voorbeeld.nl", rname: tc.rname, hostsErr: errors.New("x"), mxErr: errors.New("x")}
			p := &soaprobe.Probe{Resolver: r}
			findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			f := findingByProbeID(findings, "dns.soa")
			if f == nil {
				t.Fatalf("no dns.soa finding, got %+v", findings)
			}
			if tc.wantErr {
				if f.Attributes["rname_parse_error"] == nil {
					t.Errorf("expected rname_parse_error to be set")
				}
				return
			}
			if f.Attributes["mailbox"] != tc.wantMailbox {
				t.Errorf("mailbox = %v, want %v", f.Attributes["mailbox"], tc.wantMailbox)
			}
		})
	}
}
