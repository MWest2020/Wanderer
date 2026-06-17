package scanner

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestMailOperator(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		asnOrg string
		want   string
	}{
		{"google exact", "aspmx.l.google.com", "GOOGLE", "Google Workspace"},
		{"google trailing dot", "aspmx.l.google.com.", "GOOGLE", "Google Workspace"},
		{"microsoft subdomain", "example-com.mail.protection.outlook.com", "MICROSOFT-CORP", "Microsoft 365"},
		{"proton", "mail.protonmail.ch", "Proton AG", "Proton"},
		{"unknown falls back to asn org", "mx.acme-mail.example", "ACME HOSTING BV", "ACME HOSTING BV"},
		{"unknown no org", "mx.acme-mail.example", "", ""},
		{"suffix must be a label boundary", "notgoogle.com", "EVILCORP", "EVILCORP"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mailOperator(c.host, c.asnOrg); got != c.want {
				t.Fatalf("mailOperator(%q, %q) = %q, want %q", c.host, c.asnOrg, got, c.want)
			}
		})
	}
}

func mxFinding(domain, host string, pref int) models.Finding {
	return models.Finding{
		ProbeID:    "dns.mx",
		Subject:    domain,
		Attributes: map[string]any{"host": host, "preference": pref},
	}
}

func asnFinding(host, country, org string, asn uint) models.Finding {
	return models.Finding{
		ProbeID: "ip.asn",
		Subject: host,
		Attributes: map[string]any{
			"country":      country,
			"organisation": org,
			"asn":          asn,
		},
	}
}

func TestSynthesiseMailRouting(t *testing.T) {
	target := models.Target{Domain: "example.com"}

	t.Run("names operator and country", func(t *testing.T) {
		findings := []models.Finding{
			mxFinding("example.com", "aspmx.l.google.com.", 1),
			asnFinding("aspmx.l.google.com", "US", "GOOGLE", 15169),
		}
		f, ok := synthesiseMailRouting(target, findings)
		if !ok {
			t.Fatal("expected a mail-routing finding")
		}
		if f.ProbeID != "dns.mx_routing" || f.Severity != models.SeverityObservation {
			t.Fatalf("unexpected finding shape: %+v", f)
		}
		want := "inbound mail for example.com lands at Google Workspace (US)"
		if got := f.Attributes["summary"].(string); got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
		routes := f.Attributes["routes"].([]map[string]any)
		if len(routes) != 1 || routes[0]["operator"] != "Google Workspace" || routes[0]["country"] != "US" {
			t.Fatalf("unexpected routes: %+v", routes)
		}
	})

	t.Run("no geoip degrades to operator only", func(t *testing.T) {
		findings := []models.Finding{
			mxFinding("example.com", "aspmx.l.google.com.", 1),
			// no ip.asn — the suffix table still names the operator.
		}
		f, ok := synthesiseMailRouting(target, findings)
		if !ok {
			t.Fatal("expected a mail-routing finding")
		}
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "Google Workspace") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want operator with undetermined country", got)
		}
	})

	t.Run("anycast operator without country", func(t *testing.T) {
		findings := []models.Finding{
			mxFinding("example.com", "mx.acme-mail.example.", 10),
			asnFinding("mx.acme-mail.example", "", "ACME HOSTING BV", 64500),
		}
		f, _ := synthesiseMailRouting(target, findings)
		got := f.Attributes["summary"].(string)
		if !strings.Contains(got, "ACME HOSTING BV") || !strings.Contains(got, "country undetermined") {
			t.Fatalf("summary = %q, want org name with undetermined country", got)
		}
	})

	t.Run("multiple operators are joined", func(t *testing.T) {
		findings := []models.Finding{
			mxFinding("example.com", "example-com.mail.protection.outlook.com.", 0),
			asnFinding("example-com.mail.protection.outlook.com", "IE", "MICROSOFT-CORP", 8075),
			mxFinding("example.com", "mail.protonmail.ch.", 20),
			asnFinding("mail.protonmail.ch", "CH", "Proton AG", 209103),
		}
		f, _ := synthesiseMailRouting(target, findings)
		got := f.Attributes["summary"].(string)
		// Preference order: Microsoft (0) before Proton (20).
		want := "inbound mail for example.com lands at Microsoft 365 (IE) and Proton (CH)"
		if got != want {
			t.Fatalf("summary = %q, want %q", got, want)
		}
	})

	t.Run("no MX records states no inbound mail", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "dns.mx", Subject: "example.com", Attributes: map[string]any{"no_answer": true, "reason": "no MX records"}},
		}
		f, ok := synthesiseMailRouting(target, findings)
		if !ok {
			t.Fatal("expected a finding even with no MX")
		}
		if f.Attributes["no_inbound_mail"] != true {
			t.Fatalf("expected no_inbound_mail=true, got %+v", f.Attributes)
		}
	})

	t.Run("no dns.mx finding yields nothing", func(t *testing.T) {
		findings := []models.Finding{
			{ProbeID: "tls.issuer", Subject: "example.com"},
		}
		if _, ok := synthesiseMailRouting(target, findings); ok {
			t.Fatal("expected no finding when the dns probe did not run")
		}
	})
}
