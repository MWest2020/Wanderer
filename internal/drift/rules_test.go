package drift

import (
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func makeScan(id string, findings []models.Finding) *models.Scan {
	return &models.Scan{ID: id, Findings: findings}
}

func TestDiff_Baseline(t *testing.T) {
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "ZeroSSL"}},
	})
	got := Diff(nil, curr)
	if len(got) != 1 {
		t.Fatalf("want 1 baseline finding, got %d", len(got))
	}
	if got[0].ProbeID != "drift.baseline_established" {
		t.Errorf("want baseline ProbeID, got %s", got[0].ProbeID)
	}
}

func TestDiff_NoChanges(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "ZeroSSL"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "ZeroSSL"}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 {
		t.Fatalf("want 1 no_changes, got %d", len(got))
	}
	if got[0].ProbeID != "drift.no_changes" {
		t.Errorf("want no_changes ProbeID, got %s", got[0].ProbeID)
	}
}

func TestTLSIssuerChanged(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "ZeroSSL"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "Let's Encrypt"}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].ProbeID != "drift.tls.issuer_changed" {
		t.Errorf("want issuer_changed, got %s", got[0].ProbeID)
	}
	if got[0].Severity != models.SeverityFinding {
		t.Errorf("severity = %s, want finding", got[0].Severity)
	}
	if got[0].Attributes["prev_issuer_cn"] != "ZeroSSL" {
		t.Errorf("prev_issuer_cn = %v", got[0].Attributes["prev_issuer_cn"])
	}
}

func TestTLSDaysLeftDropped(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "tls.validity", Subject: "example.nl", Attributes: map[string]any{"days_left": 60}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "tls.validity", Subject: "example.nl", Attributes: map[string]any{"days_left": 20}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].ProbeID != "drift.tls.days_left_dropped" {
		t.Errorf("ProbeID = %s", got[0].ProbeID)
	}

	// Threshold not crossed: no finding.
	prev = makeScan("s_a", []models.Finding{
		{ProbeID: "tls.validity", Subject: "example.nl", Attributes: map[string]any{"days_left": 60}},
	})
	curr = makeScan("s_b", []models.Finding{
		{ProbeID: "tls.validity", Subject: "example.nl", Attributes: map[string]any{"days_left": 50}},
	})
	got = Diff(prev, curr)
	if len(got) != 1 || got[0].ProbeID != "drift.no_changes" {
		t.Errorf("expected no_changes when threshold not crossed; got %v", got)
	}
}

func TestDNSMXSetChanged(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "dns.mx", Subject: "example.nl", Attributes: map[string]any{"host": "mail1"}},
		{ProbeID: "dns.mx", Subject: "example.nl", Attributes: map[string]any{"host": "mail2"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "dns.mx", Subject: "example.nl", Attributes: map[string]any{"host": "mail1"}},
		{ProbeID: "dns.mx", Subject: "example.nl", Attributes: map[string]any{"host": "mail3"}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 || got[0].ProbeID != "drift.dns.mx_set_changed" {
		t.Fatalf("want mx_set_changed, got %v", got)
	}
	if added := got[0].Attributes["added"].([]string); len(added) != 1 || added[0] != "mail3" {
		t.Errorf("added = %v, want [mail3]", added)
	}
	if removed := got[0].Attributes["removed"].([]string); len(removed) != 1 || removed[0] != "mail2" {
		t.Errorf("removed = %v, want [mail2]", removed)
	}
	if got[0].DimensionHint != models.DimensionDataAI {
		t.Errorf("dim = %s, want data_ai", got[0].DimensionHint)
	}
	if got[0].Attributes["prev_scan_id"] != "s_a" {
		t.Errorf("prev_scan_id missing")
	}
}

func TestIPCountryChanged(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"country": "NL"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"country": "US"}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 || got[0].ProbeID != "drift.ip.country_changed" {
		t.Fatalf("want ip.country_changed, got %v", got)
	}
}

// lennoxAddresses is the real prod shape: six addresses for
// lennox.ns.cloudflare.com across four countries.
func lennoxAddresses() []models.Finding {
	rows := []struct{ address, country string }{
		{"108.162.195.214", "CA"},
		{"162.159.44.214", "US"},
		{"172.64.35.214", "CA"},
		{"2606:4700:58::a29f:2cd6", "CA"},
		{"2803:f800:50::6ca2:c3d6", "CR"},
		{"2a06:98c1:50::ac40:23d6", "GB"},
	}
	out := make([]models.Finding, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.Finding{
			ProbeID: "ip.asn",
			Subject: "lennox.ns.cloudflare.com",
			Attributes: map[string]any{
				"address": r.address,
				"country": r.country,
			},
		})
	}
	return out
}

func TestIPCountryChanged_MultiAddressHostUnchanged(t *testing.T) {
	prev := makeScan("s_a", lennoxAddresses())
	curr := makeScan("s_b", lennoxAddresses())
	got := Diff(prev, curr)
	for _, f := range got {
		if f.ProbeID == "drift.ip.country_changed" {
			t.Errorf("unexpected country_changed for unchanged multi-address host: %v", f)
		}
	}
	if len(got) != 1 || got[0].ProbeID != "drift.no_changes" {
		t.Fatalf("want exactly one drift.no_changes, got %v", got)
	}
}

func TestIPCountryChanged_SameAddressDifferentCountry(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"address": "192.0.2.10", "country": "NL"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"address": "192.0.2.10", "country": "US"}},
	})
	got := Diff(prev, curr)
	if len(got) != 1 || got[0].ProbeID != "drift.ip.country_changed" {
		t.Fatalf("want ip.country_changed, got %v", got)
	}
	if got[0].Attributes["address"] != "192.0.2.10" {
		t.Errorf("address = %v, want 192.0.2.10", got[0].Attributes["address"])
	}
	if got[0].Attributes["prev_country"] != "NL" {
		t.Errorf("prev_country = %v, want NL", got[0].Attributes["prev_country"])
	}
	if got[0].Attributes["curr_country"] != "US" {
		t.Errorf("curr_country = %v, want US", got[0].Attributes["curr_country"])
	}
}

func TestIPCountryChanged_NewAddressNoCountryChange(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"address": "192.0.2.10", "country": "NL"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"address": "192.0.2.10", "country": "NL"}},
		{ProbeID: "ip.asn", Subject: "example.nl", Attributes: map[string]any{"address": "198.51.100.20", "country": "US"}},
	})
	got := Diff(prev, curr)
	for _, f := range got {
		if f.ProbeID == "drift.ip.country_changed" {
			t.Errorf("unexpected country_changed for added address: %v", f)
		}
	}
}

func TestHTTPThirdPartyChanged(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "http.third_party", Subject: "cdn.example.nl"},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "http.third_party", Subject: "fonts.googleapis.com"},
	})
	got := Diff(prev, curr)
	probes := map[string]bool{}
	for _, f := range got {
		probes[f.ProbeID] = true
	}
	if !probes["drift.http.third_party_added"] {
		t.Errorf("missing added drift")
	}
	if !probes["drift.http.third_party_removed"] {
		t.Errorf("missing removed drift")
	}
}

func TestDriftCarriesSourceModus(t *testing.T) {
	prev := makeScan("s_a", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "A"}},
	})
	curr := makeScan("s_b", []models.Finding{
		{ProbeID: "tls.issuer", Subject: "example.nl", Attributes: map[string]any{"issuer_cn": "B"}},
	})
	got := Diff(prev, curr)
	for _, f := range got {
		if f.Attributes["source_modus"] != SourceModusDrift {
			t.Errorf("missing source_modus on %s", f.ProbeID)
		}
	}
}
