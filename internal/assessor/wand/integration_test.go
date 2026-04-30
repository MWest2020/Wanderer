package wand_test

// Integration test: pin the contract between probe IDs and assessor IDs.
// Bug 1 of the 2026-04-27 audit was a casing mismatch (rule looked for
// "dns.A" while probe emits "dns.a") that no per-package test could
// catch — both sides happily agreed on the wrong string. This test
// drives the real DNS probe and the real DICTU rules end-to-end so any
// future drift on either side breaks the build.

import (
	"context"
	"net"
	"testing"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/internal/probe"
	dnsprobe "github.com/MWest2020/wanderer/internal/probe/dns"
	"github.com/MWest2020/wanderer/pkg/models"
)

// fakeResolver is a duplicate of the one in the dns package's tests.
// We can't import _test.go files from another package, so we keep a
// minimal version here. Only methods the DNS probe calls are wired.
type fakeResolver struct {
	hosts []string
	mx    []*net.MX
	ns    []*net.NS
	txt   map[string][]string
	caa   []dnsprobe.CAA
}

func (f *fakeResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return f.hosts, nil
}
func (f *fakeResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return f.mx, nil
}
func (f *fakeResolver) LookupNS(_ context.Context, _ string) ([]*net.NS, error) {
	return f.ns, nil
}
func (f *fakeResolver) LookupCNAME(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (f *fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	return f.txt[name], nil
}
func (f *fakeResolver) LookupCAA(_ context.Context, _ string) ([]dnsprobe.CAA, error) {
	return f.caa, nil
}

// asnFinding constructs an ip.asn finding the same way the IP probe
// would. Done by hand so this test does not require a GeoLite2 mmdb on
// disk — the contract being asserted is probe-ID + Subject + attribute
// shape, not the GeoIP plumbing.
func asnFinding(host, country, org string) models.Finding {
	return models.Finding{
		ProbeID:       "ip.asn",
		DimensionHint: models.DimensionJuridisch,
		Subject:       host,
		Severity:      models.SeverityFinding,
		Attributes: map[string]any{
			"country":      country,
			"organisation": org,
		},
	}
}

func httpThirdParty(host string) models.Finding {
	return models.Finding{
		ProbeID:       "http.third_party",
		DimensionHint: models.DimensionTechnologie,
		Subject:       host,
		Severity:      models.SeverityObservation,
	}
}

// rationaleByID pulls a single Rationale out of an Assessment's
// dimensions by criterium ID, so the test can assert on a specific rule
// without depending on dimension/index ordering.
func rationaleByID(scores []models.DimensionScore, id string) (models.Rationale, bool) {
	for _, ds := range scores {
		for _, r := range ds.Rationale {
			if r.CriteriumID == id {
				return r, true
			}
		}
	}
	return models.Rationale{}, false
}

func TestDNSProbeFeedsApexIPEEARule(t *testing.T) {
	dp := &dnsprobe.Probe{Resolver: &fakeResolver{
		hosts: []string{"94.198.159.35"},
		ns:    []*net.NS{{Host: "ns1.transip.net."}, {Host: "ns2.transip.nl."}},
	}}
	dnsFindings, err := dp.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("dns run: %v", err)
	}
	all := append([]models.Finding{}, dnsFindings...)
	all = append(all, asnFinding("example.nl", "NL", "TransIP B.V."))

	scores := assessor.Assess(all, wand.DefaultRules())
	rat, ok := rationaleByID(scores, "wand.juridisch.apex_ip_eea")
	if !ok {
		t.Fatal("apex_ip_eea rationale missing from assessment")
	}
	if rat.Score != models.ScoreSoeverein {
		t.Errorf("apex_ip_eea score = %s, want soeverein (probe-ID drift?)", rat.Score)
	}
	if len(rat.Evidence) == 0 {
		t.Error("apex_ip_eea: no evidence — rule did not match real DNS probe output")
	}
}

func TestDNSProbeFeedsMXVendorJurisdictionRule(t *testing.T) {
	dp := &dnsprobe.Probe{Resolver: &fakeResolver{
		hosts: []string{"94.198.159.35"},
		mx: []*net.MX{
			{Host: "mail.protection.outlook.com.", Pref: 10},
		},
		ns: []*net.NS{{Host: "ns1.example.nl."}, {Host: "ns2.example.nl."}},
	}}
	dnsFindings, err := dp.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("dns run: %v", err)
	}
	all := append([]models.Finding{}, dnsFindings...)
	// Subject of the MX correlation is the MX host itself (lowercased,
	// trailing-dot stripped) — this mirrors how the scanner's Related
	// expansion feeds the IP probe.
	all = append(all, asnFinding("mail.protection.outlook.com", "IE", "Microsoft Ireland"))

	scores := assessor.Assess(all, wand.DefaultRules())
	rat, ok := rationaleByID(scores, "wand.juridisch.mx_vendor_jurisdiction")
	if !ok {
		t.Fatal("mx_vendor_jurisdiction rationale missing")
	}
	if rat.Score != models.ScoreSoeverein {
		t.Errorf("mx_vendor_jurisdiction score = %s, want soeverein", rat.Score)
	}
	if len(rat.Evidence) == 0 {
		t.Error("mx_vendor_jurisdiction: no evidence")
	}
}

func TestThirdPartiesEEARuleContract(t *testing.T) {
	// The third-parties rule depends on http.third_party Subject ==
	// ip.asn Subject. Pin that contract.
	all := []models.Finding{
		httpThirdParty("cdn.example.eu"),
		asnFinding("cdn.example.eu", "DE", "Hetzner Online GmbH"),
	}
	scores := assessor.Assess(all, wand.DefaultRules())
	rat, ok := rationaleByID(scores, "wand.technologie.third_parties_eea")
	if !ok {
		t.Fatal("third_parties_eea rationale missing")
	}
	if rat.Score != models.ScoreSoeverein {
		t.Errorf("third_parties_eea score = %s, want soeverein", rat.Score)
	}
	if len(rat.Evidence) == 0 {
		t.Error("third_parties_eea: no evidence")
	}
}
