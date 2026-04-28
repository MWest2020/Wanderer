package eucsf

import (
	"testing"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

func ruleByID(t *testing.T, id string) assessor.Rule {
	t.Helper()
	for _, r := range DefaultRules() {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("rule %s not registered", id)
	return assessor.Rule{}
}

func f(id, probeID string, attrs map[string]any) models.Finding {
	return models.Finding{
		ID:         id,
		ProbeID:    probeID,
		Subject:    attrs["_subject"].(string),
		Severity:   models.SeverityFinding,
		Attributes: dropKey(attrs, "_subject"),
	}
}

func dropKey(m map[string]any, key string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k == key {
			continue
		}
		out[k] = v
	}
	return out
}

func TestCertIssuerEU(t *testing.T) {
	r := ruleByID(t, "eucsf.sov2.cert_issuer_eu")
	got := r.Match([]models.Finding{
		f("f1", "tls.issuer", map[string]any{"_subject": "example.nl", "issuer_country": []string{"NL"}}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU issuer: score = %s, want soeverein (SEAL 4)", got.Score)
	}
	got = r.Match([]models.Finding{
		f("f1", "tls.issuer", map[string]any{"_subject": "example.com", "issuer_country": []string{"US"}}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US issuer: score = %s, want afhankelijk (SEAL 1)", got.Score)
	}
	got = r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend (SEAL 0)", got.Score)
	}
}

func TestApexJurisdiction(t *testing.T) {
	r := ruleByID(t, "eucsf.sov2.apex_jurisdiction")
	got := r.Match([]models.Finding{
		f("d1", "dns.a", map[string]any{"_subject": "example.nl", "address": "1.2.3.4"}),
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "NL", "organisation": "TransIP"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU apex: score = %s, want soeverein", got.Score)
	}
	got = r.Match([]models.Finding{
		f("d1", "dns.a", map[string]any{"_subject": "example.com", "address": "5.6.7.8"}),
		f("a1", "ip.asn", map[string]any{"_subject": "example.com", "country": "US", "organisation": "Amazon"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US apex: score = %s, want afhankelijk", got.Score)
	}
}

func TestMXJurisdiction(t *testing.T) {
	r := ruleByID(t, "eucsf.sov3.mx_jurisdiction")
	got := r.Match([]models.Finding{
		f("m1", "dns.mx", map[string]any{"_subject": "example.nl", "host": "mail.example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "mail.example.nl", "country": "DE", "organisation": "Hetzner"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU mx: score = %s, want soeverein", got.Score)
	}
}

func TestOperationalEU(t *testing.T) {
	r := ruleByID(t, "eucsf.sov4.operational_eu")
	got := r.Match([]models.Finding{
		f("t1", "http.third_party", map[string]any{"_subject": "cdn.example.nl", "source_domain": "example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "cdn.example.nl", "country": "NL", "organisation": "Leaseweb"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU third party: score = %s, want soeverein", got.Score)
	}
	got = r.Match([]models.Finding{
		f("t1", "http.third_party", map[string]any{"_subject": "fonts.googleapis.com", "source_domain": "example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "fonts.googleapis.com", "country": "US", "organisation": "Google LLC"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US third party: score = %s, want afhankelijk", got.Score)
	}
}

func TestNoUSHyperscaler(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.no_us_hyperscaler")
	got := r.Match([]models.Finding{
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "NL", "organisation": "Leaseweb Netherlands"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("no hyperscaler: score = %s, want soeverein", got.Score)
	}
	got = r.Match([]models.Finding{
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "US", "organisation": "Amazon.com, Inc."}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("hyperscaler: score = %s, want afhankelijk", got.Score)
	}
}

func TestDefaultRules_FiveRules(t *testing.T) {
	if got := len(DefaultRules()); got != 5 {
		t.Errorf("DefaultRules count = %d, want 5", got)
	}
}
