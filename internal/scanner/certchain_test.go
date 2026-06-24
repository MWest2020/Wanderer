package scanner

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestCertAuthority(t *testing.T) {
	cases := []struct {
		name      string
		issuerOrg string
		issuerCN  string
		want      string
	}{
		{"lets encrypt by org", "Let's Encrypt", "R3", "Let's Encrypt"},
		{"lets encrypt by cn only", "", "R10", "R10"}, // CN "R10" not in table → raw CN
		{"isrg cn", "", "ISRG Root X1", "Let's Encrypt"},
		{"digicert", "DigiCert Inc", "DigiCert Global G2", "DigiCert"},
		{"google", "Google Trust Services LLC", "WE1", "Google Trust Services"},
		{"unknown falls back to org", "Acme Internal CA", "acme-ca", "Acme Internal CA"},
		{"unknown cn fallback", "", "acme-ca", "acme-ca"},
		{"empty stays empty", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := certAuthority(c.issuerOrg, c.issuerCN); got != c.want {
				t.Fatalf("certAuthority(%q, %q) = %q, want %q", c.issuerOrg, c.issuerCN, got, c.want)
			}
		})
	}
}

func issuerFinding(domain, org, cn, country string) models.Finding {
	return models.Finding{
		ProbeID: "tls.issuer",
		Subject: domain,
		Attributes: map[string]any{
			"issuer_o":       []string{org},
			"issuer_cn":      cn,
			"issuer_country": []string{country},
		},
	}
}

func chainFinding(domain string, details ...map[string]any) models.Finding {
	return models.Finding{
		ProbeID: "tls.chain",
		Subject: domain,
		Attributes: map[string]any{
			"length":               len(details) + 1,
			"intermediate_details": details,
		},
	}
}

func TestSynthesiseCertChain(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("names CA and country with chain", func(t *testing.T) {
		findings := []models.Finding{
			issuerFinding("example.com", "Let's Encrypt", "R3", "US"),
			chainFinding("example.com",
				map[string]any{"cn": "ISRG Root X1", "organisation": "Internet Security Research Group", "country": "US"}),
		}
		f, ok := synthesiseCertChain(target, findings)
		if !ok {
			t.Fatal("expected a cert-chain finding")
		}
		if f.ProbeID != "tls.chain_geography" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		if f.Attributes["ca"] != "Let's Encrypt" || f.Attributes["country"] != "US" {
			t.Fatalf("unexpected attributes: %+v", f.Attributes)
		}
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "issued by Let's Encrypt (US)") || !strings.Contains(got, "chain ← Let's Encrypt (US)") {
			t.Fatalf("summary = %q, want CA + chain", got)
		}
	})

	t.Run("no issuer country names CA with undetermined", func(t *testing.T) {
		findings := []models.Finding{
			issuerFinding("example.com", "DigiCert Inc", "DigiCert Global G2", ""),
		}
		f, _ := synthesiseCertChain(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "DigiCert") || !strings.Contains(got, "jurisdiction undetermined") {
			t.Fatalf("summary = %q, want CA with undetermined jurisdiction", got)
		}
	})

	t.Run("unrecognised issuer falls back to raw org", func(t *testing.T) {
		findings := []models.Finding{
			issuerFinding("example.com", "Acme Internal CA", "acme-ca", "NL"),
		}
		f, _ := synthesiseCertChain(target, findings)
		if f.Attributes["ca"] != "Acme Internal CA" {
			t.Fatalf("ca = %v, want raw org", f.Attributes["ca"])
		}
	})

	t.Run("no tls.issuer yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "http.response", Subject: "example.com", Attributes: map[string]any{"status": 200}},
		}
		if _, ok := synthesiseCertChain(target, findings); ok {
			t.Fatal("expected no finding when the TLS probe produced no issuer")
		}
	})
}
