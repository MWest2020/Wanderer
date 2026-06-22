package scanner

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestThirdPartyVendor(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		asnOrg string
		want   string
	}{
		{"google fonts", "fonts.googleapis.com", "GOOGLE", "Google Fonts"},
		{"google fonts static", "fonts.gstatic.com.", "GOOGLE", "Google Fonts"},
		{"jsdelivr", "cdn.jsdelivr.net", "FASTLY", "jsDelivr"},
		{"unknown falls back to asn org", "assets.acme-cdn.example", "ACME CDN BV", "ACME CDN BV"},
		{"unknown no org", "assets.acme-cdn.example", "", ""},
		{"suffix must be a label boundary", "notjsdelivr.net", "EVILCORP", "EVILCORP"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := thirdPartyVendor(c.host, c.asnOrg); got != c.want {
				t.Fatalf("thirdPartyVendor(%q, %q) = %q, want %q", c.host, c.asnOrg, got, c.want)
			}
		})
	}
}

func tpFinding(domain, host string, kinds ...string) models.Finding {
	return models.Finding{
		ProbeID:    "http.third_party",
		Subject:    host,
		Attributes: map[string]any{"source_domain": domain, "kinds": kinds},
	}
}

func TestSynthesiseOriginMap(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("groups hosts by vendor and unions kinds", func(t *testing.T) {
		findings := []models.Finding{
			tpFinding("example.com", "fonts.googleapis.com", "link"),
			tpFinding("example.com", "fonts.gstatic.com", "link"),
			asnFinding("fonts.googleapis.com", "US", "GOOGLE", 15169),
			asnFinding("fonts.gstatic.com", "US", "GOOGLE", 15169),
		}
		f, ok := synthesiseOriginMap(target, findings)
		if !ok {
			t.Fatal("expected an origin-map finding")
		}
		if f.ProbeID != "http.origin_map" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		vendors := f.Attributes["vendors"].([]map[string]any)
		if len(vendors) != 1 {
			t.Fatalf("expected the two Google hosts to collapse into one vendor, got %+v", vendors)
		}
		if vendors[0]["vendor"] != "Google Fonts" || vendors[0]["country"] != "US" {
			t.Fatalf("unexpected vendor entry: %+v", vendors[0])
		}
		hosts := vendors[0]["hosts"].([]string)
		if len(hosts) != 2 {
			t.Fatalf("expected both raw hosts retained as evidence, got %+v", hosts)
		}
	})

	t.Run("multiple vendors are joined", func(t *testing.T) {
		findings := []models.Finding{
			tpFinding("example.com", "fonts.googleapis.com", "link"),
			tpFinding("example.com", "cdn.jsdelivr.net", "script"),
			asnFinding("fonts.googleapis.com", "US", "GOOGLE", 15169),
			asnFinding("cdn.jsdelivr.net", "US", "FASTLY", 54113),
		}
		f, _ := synthesiseOriginMap(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "Google Fonts") || !strings.Contains(got, "jsDelivr") {
			t.Fatalf("summary = %q, want both vendors named", got)
		}
	})

	t.Run("no geoip degrades to vendor only", func(t *testing.T) {
		findings := []models.Finding{
			tpFinding("example.com", "fonts.googleapis.com", "link"),
			// no ip.asn — the suffix table still names the vendor.
		}
		f, _ := synthesiseOriginMap(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "Google Fonts") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want vendor with undetermined country", got)
		}
	})

	t.Run("unrecognised host falls back to asn org", func(t *testing.T) {
		findings := []models.Finding{
			tpFinding("example.com", "assets.acme-cdn.example", "img"),
			asnFinding("assets.acme-cdn.example", "NL", "ACME CDN BV", 64500),
		}
		f, _ := synthesiseOriginMap(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "ACME CDN BV") {
			t.Fatalf("summary = %q, want raw ASN org as vendor", got)
		}
	})

	t.Run("page fetched with no third parties states so", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "http.response", Subject: "example.com", Attributes: map[string]any{"status": 200}},
		}
		f, ok := synthesiseOriginMap(target, findings)
		if !ok {
			t.Fatal("expected a finding when the page was fetched but loaded no third parties")
		}
		if f.Attributes["no_third_parties"] != true {
			t.Fatalf("expected no_third_parties=true, got %+v", f.Attributes)
		}
	})

	t.Run("http probe did not run yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "tls.issuer", Subject: "example.com"},
		}
		if _, ok := synthesiseOriginMap(target, findings); ok {
			t.Fatal("expected no finding when the HTTP probe did not run")
		}
	})
}
