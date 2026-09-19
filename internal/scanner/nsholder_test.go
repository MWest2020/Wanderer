package scanner

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestRegistrableDomain(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"ns1.provider-a.nl", "provider-a.nl"},
		{"ns2.provider-a.nl.", "provider-a.nl"},
		{"ns1.provider-b.eu", "provider-b.eu"},
		{"provider-a.nl", "provider-a.nl"},
		{"nl", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := registrableDomain(c.host); got != c.want {
			t.Errorf("registrableDomain(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

func TestUniqueRegistrableDomains(t *testing.T) {
	findings := []models.Finding{
		nsFinding("target.nl", "ns1.provider-a.nl"),
		nsFinding("target.nl", "ns2.provider-a.nl"),
		nsFinding("target.nl", "ns1.provider-b.eu"),
		{ProbeID: "dns.mx", Subject: "target.nl", Attributes: map[string]any{"host": "mail.provider-c.nl"}},
	}
	got := uniqueRegistrableDomains(findings)
	want := []string{"provider-a.nl", "provider-b.eu"}
	if len(got) != len(want) {
		t.Fatalf("domains = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("domains[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
