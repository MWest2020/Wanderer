package scanner

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestCDNFront(t *testing.T) {
	cases := []struct {
		name        string
		asnOrg      string
		server      string
		wantEdge    string
		wantSignals []string
	}{
		{"cloudflare both signals", "CLOUDFLARENET", "cloudflare", "Cloudflare", []string{"asn", "server"}},
		{"cloudflare org only", "CLOUDFLARENET", "", "Cloudflare", []string{"asn"}},
		{"cloudfront server only", "AMAZON-02", "CloudFront", "Amazon CloudFront", []string{"server"}},
		{"vercel header only", "", "Vercel", "Vercel", []string{"server"}},
		{"fastly", "FASTLY", "", "Fastly", []string{"asn"}},
		{"no match", "ACME HOSTING BV", "nginx", "", nil},
		{"empty inputs no match", "", "", "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			edge, signals := cdnFront(c.asnOrg, c.server)
			if edge != c.wantEdge {
				t.Fatalf("cdnFront(%q, %q) edge = %q, want %q", c.asnOrg, c.server, edge, c.wantEdge)
			}
			if !reflect.DeepEqual(signals, c.wantSignals) {
				t.Fatalf("cdnFront(%q, %q) signals = %v, want %v", c.asnOrg, c.server, signals, c.wantSignals)
			}
		})
	}
}

func responseFinding(domain, server string) models.Finding {
	return models.Finding{
		ProbeID:    "http.response",
		Subject:    domain,
		Attributes: map[string]any{"server": server, "status": 200},
	}
}

func TestSynthesiseCDNFront(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("fronted apex names the edge and country", func(t *testing.T) {
		findings := []models.Finding{
			asnFinding("example.com", "US", "CLOUDFLARENET", 13335),
			responseFinding("example.com", "cloudflare"),
		}
		f, ok := synthesiseCDNFront(target, findings)
		if !ok {
			t.Fatal("expected a cdn-front finding")
		}
		if f.ProbeID != "http.cdn_front" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		if f.Attributes["fronted"] != true || f.Attributes["edge"] != "Cloudflare" || f.Attributes["country"] != "US" {
			t.Fatalf("unexpected attributes: %+v", f.Attributes)
		}
		want := "example.com's apex is fronted by Cloudflare (US)"
		if got := f.Attributes["summary"].(string); got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
		signals := f.Attributes["signals"].([]string)
		if len(signals) != 2 {
			t.Fatalf("expected both signals recorded, got %v", signals)
		}
	})

	t.Run("server-header-only detection without geoip", func(t *testing.T) {
		findings := []models.Finding{
			responseFinding("example.com", "Vercel"),
			// no ip.asn — header alone names the edge, country omitted.
		}
		f, _ := synthesiseCDNFront(target, findings)
		if f.Attributes["edge"] != "Vercel" {
			t.Fatalf("expected Vercel from header, got %+v", f.Attributes)
		}
		if _, ok := f.Attributes["country"]; ok {
			t.Fatalf("expected no country without geoip, got %+v", f.Attributes["country"])
		}
		if !strings.Contains(f.Attributes["summary"].(string), "country undetermined") {
			t.Fatalf("summary should note undetermined country, got %q", f.Attributes["summary"])
		}
	})

	t.Run("anycast edge without country", func(t *testing.T) {
		findings := []models.Finding{
			asnFinding("example.com", "", "FASTLY", 54113),
			responseFinding("example.com", "Fastly"),
		}
		f, _ := synthesiseCDNFront(target, findings)
		if f.Attributes["edge"] != "Fastly" {
			t.Fatalf("expected Fastly, got %+v", f.Attributes)
		}
		if !strings.Contains(f.Attributes["summary"].(string), "anycast") {
			t.Fatalf("summary should note anycast, got %q", f.Attributes["summary"])
		}
	})

	t.Run("directly-served apex reads as no front", func(t *testing.T) {
		findings := []models.Finding{
			asnFinding("example.com", "NL", "TransIP B.V.", 20857),
			responseFinding("example.com", "nginx"),
		}
		f, ok := synthesiseCDNFront(target, findings)
		if !ok {
			t.Fatal("expected a finding for a directly-served apex")
		}
		if f.Attributes["fronted"] != false {
			t.Fatalf("expected fronted=false, got %+v", f.Attributes)
		}
		if !strings.Contains(f.Attributes["summary"].(string), "served directly") {
			t.Fatalf("summary should say served directly, got %q", f.Attributes["summary"])
		}
	})

	t.Run("no apex evidence yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "tls.issuer", Subject: "example.com", Attributes: map[string]any{"issuer_o": []string{"Let's Encrypt"}}},
		}
		if _, ok := synthesiseCDNFront(target, findings); ok {
			t.Fatal("expected no finding when the IP and HTTP probes did not run")
		}
	})
}
