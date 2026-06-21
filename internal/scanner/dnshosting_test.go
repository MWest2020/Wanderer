package scanner

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestDNSOperator(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		asnOrg string
		want   string
	}{
		{"cloudflare", "ns1.cloudflare.com", "CLOUDFLARENET", "Cloudflare"},
		{"cloudflare trailing dot", "ns1.cloudflare.com.", "CLOUDFLARENET", "Cloudflare"},
		{"aws route53 family", "ns-1234.awsdns-56.org", "AMAZON-02", "AWS Route 53"},
		{"azure", "ns1-01.azure-dns.com", "MICROSOFT-CORP", "Azure DNS"},
		{"transip", "ns0.transip.net", "TransIP B.V.", "TransIP"},
		{"unknown falls back to asn org", "ns1.acme-dns.example", "ACME HOSTING BV", "ACME HOSTING BV"},
		{"unknown no org", "ns1.acme-dns.example", "", ""},
		{"suffix must be a label boundary", "notcloudflare.com", "EVILCORP", "EVILCORP"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dnsOperator(c.host, c.asnOrg); got != c.want {
				t.Fatalf("dnsOperator(%q, %q) = %q, want %q", c.host, c.asnOrg, got, c.want)
			}
		})
	}
}

func nsFinding(domain, host string) models.Finding {
	return models.Finding{
		ProbeID:    "dns.ns",
		Subject:    domain,
		Attributes: map[string]any{"host": host},
	}
}

func TestSynthesiseDNSHosting(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("names operator and country", func(t *testing.T) {
		findings := []models.Finding{
			nsFinding("example.com", "ns1.cloudflare.com."),
			asnFinding("ns1.cloudflare.com", "US", "CLOUDFLARENET", 13335),
		}
		f, ok := synthesiseDNSHosting(target, findings)
		if !ok {
			t.Fatal("expected a dns-hosting finding")
		}
		if f.ProbeID != "dns.ns_hosting" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		want := "DNS for example.com is run by Cloudflare (US)"
		if got := f.Attributes["summary"].(string); got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
		routes := f.Attributes["routes"].([]map[string]any)
		if len(routes) != 1 || routes[0]["operator"] != "Cloudflare" || routes[0]["country"] != "US" {
			t.Fatalf("unexpected routes: %+v", routes)
		}
	})

	t.Run("no geoip degrades to operator only", func(t *testing.T) {
		findings := []models.Finding{
			nsFinding("example.com", "ns1.cloudflare.com."),
			// no ip.asn — the suffix table still names the operator.
		}
		f, ok := synthesiseDNSHosting(target, findings)
		if !ok {
			t.Fatal("expected a dns-hosting finding")
		}
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "Cloudflare") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want operator with undetermined country", got)
		}
	})

	t.Run("anycast operator without country", func(t *testing.T) {
		findings := []models.Finding{
			nsFinding("example.com", "ns1.acme-dns.example."),
			asnFinding("ns1.acme-dns.example", "", "ACME HOSTING BV", 64500),
		}
		f, _ := synthesiseDNSHosting(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "ACME HOSTING BV") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want org name with undetermined country", got)
		}
	})

	t.Run("multiple operators are joined and deduped", func(t *testing.T) {
		findings := []models.Finding{
			nsFinding("example.com", "ns0.transip.net."),
			asnFinding("ns0.transip.net", "NL", "TransIP B.V.", 20857),
			nsFinding("example.com", "ns1.cloudflare.com."),
			asnFinding("ns1.cloudflare.com", "US", "CLOUDFLARENET", 13335),
			// a second Cloudflare NS collapses into one "Cloudflare (US)".
			nsFinding("example.com", "ns2.cloudflare.com."),
			asnFinding("ns2.cloudflare.com", "US", "CLOUDFLARENET", 13335),
		}
		f, _ := synthesiseDNSHosting(target, findings)
		got := f.Attributes["summary"].(string)
		// Host order: ns0.transip < ns1.cloudflare < ns2.cloudflare.
		want := "DNS for example.com is run by TransIP (NL) and Cloudflare (US)"
		if got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
	})

	t.Run("no resolvable authoritative DNS states so", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "dns.ns", Subject: "example.com", Attributes: map[string]any{"no_answer": true, "reason": "no NS records"}},
		}
		f, ok := synthesiseDNSHosting(target, findings)
		if !ok {
			t.Fatal("expected a finding even with no resolvable NS")
		}
		if f.Attributes["no_authoritative_dns"] != true {
			t.Fatalf("expected no_authoritative_dns=true, got %+v", f.Attributes)
		}
	})

	t.Run("no dns.ns finding yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "tls.issuer", Subject: "example.com"},
		}
		if _, ok := synthesiseDNSHosting(target, findings); ok {
			t.Fatal("expected no finding when the dns probe did not run")
		}
	})
}
