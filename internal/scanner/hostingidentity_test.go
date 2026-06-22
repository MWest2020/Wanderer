package scanner

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestHostingOperator(t *testing.T) {
	cases := []struct {
		name   string
		asnOrg string
		want   string
	}{
		{"hetzner friendly", "HETZNER-AS", "Hetzner"},
		{"amazon friendly", "AMAZON-02", "AWS"},
		{"cloudflare friendly", "CLOUDFLARENET", "Cloudflare"},
		{"microsoft friendly", "MICROSOFT-CORP-MSN-AS-BLOCK", "Microsoft Azure"},
		{"case-insensitive", "ovh sas", "OVH"},
		{"unknown falls back to raw org", "ACME HOSTING BV", "ACME HOSTING BV"},
		{"empty stays empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hostingOperator(c.asnOrg); got != c.want {
				t.Fatalf("hostingOperator(%q) = %q, want %q", c.asnOrg, got, c.want)
			}
		})
	}
}

func aFinding(domain, addr string) models.Finding {
	return models.Finding{
		ProbeID:    "dns.a",
		Subject:    domain,
		Attributes: map[string]any{"address": addr},
	}
}

func asnAddrFinding(host, addr, country, org string, asn uint) models.Finding {
	return models.Finding{
		ProbeID: "ip.asn",
		Subject: host,
		Attributes: map[string]any{
			"address":      addr,
			"country":      country,
			"organisation": org,
			"asn":          asn,
		},
	}
}

func TestSynthesiseHostingIdentity(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("names operator and country", func(t *testing.T) {
		findings := []models.Finding{
			aFinding("example.com", "1.2.3.4"),
			asnAddrFinding("example.com", "1.2.3.4", "DE", "HETZNER-AS", 24940),
		}
		f, ok := synthesiseHostingIdentity(target, findings)
		if !ok {
			t.Fatal("expected a hosting-identity finding")
		}
		if f.ProbeID != "ip.hosting" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		want := "example.com is hosted at Hetzner (DE)"
		if got := f.Attributes["summary"].(string); got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
		routes := f.Attributes["routes"].([]map[string]any)
		if len(routes) != 1 || routes[0]["operator"] != "Hetzner" || routes[0]["organisation"] != "HETZNER-AS" {
			t.Fatalf("unexpected routes: %+v", routes)
		}
	})

	t.Run("unrecognised org falls back to raw name", func(t *testing.T) {
		findings := []models.Finding{
			aFinding("example.com", "1.2.3.4"),
			asnAddrFinding("example.com", "1.2.3.4", "NL", "ACME HOSTING BV", 64500),
		}
		f, _ := synthesiseHostingIdentity(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "ACME HOSTING BV") || !strings.Contains(got, "(NL)") {
			t.Fatalf("summary = %q, want raw org with country", got)
		}
	})

	t.Run("no geoip reports operator undetermined", func(t *testing.T) {
		findings := []models.Finding{
			aFinding("example.com", "1.2.3.4"),
			// no ip.asn — no GeoIP database.
		}
		f, ok := synthesiseHostingIdentity(target, findings)
		if !ok {
			t.Fatal("expected a finding even without GeoIP")
		}
		if f.Attributes["operator_undetermined"] != true {
			t.Fatalf("expected operator_undetermined=true, got %+v", f.Attributes)
		}
	})

	t.Run("anycast operator without country", func(t *testing.T) {
		findings := []models.Finding{
			aFinding("example.com", "1.2.3.4"),
			asnAddrFinding("example.com", "1.2.3.4", "", "CLOUDFLARENET", 13335),
		}
		f, _ := synthesiseHostingIdentity(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "Cloudflare") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want operator with undetermined country", got)
		}
	})

	t.Run("no resolvable apex states so", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "dns.a", Subject: "example.com", Attributes: map[string]any{"no_answer": true, "reason": "no A/AAAA answers"}},
		}
		f, ok := synthesiseHostingIdentity(target, findings)
		if !ok {
			t.Fatal("expected a finding even with no resolvable apex")
		}
		if f.Attributes["no_apex_host"] != true {
			t.Fatalf("expected no_apex_host=true, got %+v", f.Attributes)
		}
	})

	t.Run("no dns.a finding yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "tls.issuer", Subject: "example.com"},
		}
		if _, ok := synthesiseHostingIdentity(target, findings); ok {
			t.Fatal("expected no finding when the dns probe did not run")
		}
	})
}
