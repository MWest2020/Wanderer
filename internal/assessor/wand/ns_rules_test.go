package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func nsFinding(id, host string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "dns.ns", Subject: "example.nl",
		Attributes: map[string]any{"host": host},
	}
}

func ipASN(id, host, country string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "ip.asn", Subject: host,
		Attributes: map[string]any{"country": country, "organisation": "Org"},
	}
}

func TestNSVendorJurisdiction_EEASoeverein(t *testing.T) {
	r := nsVendorJurisdiction()
	got := r.Match([]models.Finding{
		nsFinding("n1", "ns1.transip.nl."),
		nsFinding("n2", "ns2.transip.nl."),
		ipASN("a1", "ns1.transip.nl", "NL"),
		ipASN("a2", "ns2.transip.nl", "NL"),
	})
	if got.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Error("soeverein must cite evidence")
	}
	// Evidence must cite both sides of the correlation: the dns.ns IDs
	// as well as the ip.asn IDs.
	if !contains(got.Evidence, "n1") {
		t.Errorf("evidence should cite the dns.ns finding ID, got %v", got.Evidence)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestNSVendorJurisdiction_NonEEAAfhankelijk(t *testing.T) {
	r := nsVendorJurisdiction()
	got := r.Match([]models.Finding{
		nsFinding("n1", "alice.ns.cloudflare.com."),
		ipASN("a1", "alice.ns.cloudflare.com", "US"),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "outside EEA") {
		t.Errorf("verdict = %q", got.Verdict)
	}
}

func TestNSVendorJurisdiction_SplitVoldoende(t *testing.T) {
	r := nsVendorJurisdiction()
	got := r.Match([]models.Finding{
		nsFinding("n1", "ns1.transip.nl."),
		nsFinding("n2", "alice.ns.cloudflare.com."),
		ipASN("a1", "ns1.transip.nl", "NL"),
		ipASN("a2", "alice.ns.cloudflare.com", "US"),
	})
	if got.Score != models.ScoreVoldoende {
		t.Fatalf("score = %s, want voldoende", got.Score)
	}
}

func TestNSVendorJurisdiction_NoNS_Onbekend(t *testing.T) {
	r := nsVendorJurisdiction()
	if got := r.Match([]models.Finding{ipASN("a1", "x", "NL")}); got.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend", got.Score)
	}
}

func TestNSVendorJurisdiction_NSButNoGeo_Onbekend(t *testing.T) {
	r := nsVendorJurisdiction()
	got := r.Match([]models.Finding{nsFinding("n1", "ns1.example.nl.")})
	if got.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend", got.Score)
	}
}
