package wand

import (
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// ruleByID finds one rule from DefaultRules by its ID so tests can
// exercise rules individually without coupling to slice indexes.
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

func TestCertIssuerEEA(t *testing.T) {
	r := ruleByID(t, "wand.juridisch.cert_issuer_eea")
	cases := []struct {
		name       string
		findings   []models.Finding
		wantScore  models.Score
		wantHasEvi bool
	}{
		{
			name: "EEA issuer",
			findings: []models.Finding{
				f("f1", "tls.issuer", map[string]any{"_subject": "example.nl", "issuer_country": []string{"NL"}}),
			},
			wantScore:  models.ScoreSoeverein,
			wantHasEvi: true,
		},
		{
			name: "US issuer",
			findings: []models.Finding{
				f("f1", "tls.issuer", map[string]any{"_subject": "example.nl", "issuer_country": []string{"US"}}),
			},
			wantScore:  models.ScoreAfhankelijk,
			wantHasEvi: true,
		},
		{
			name:      "no tls.issuer finding",
			findings:  nil,
			wantScore: models.ScoreOnbekend,
		},
		{
			name: "missing issuer_country attribute",
			findings: []models.Finding{
				f("f1", "tls.issuer", map[string]any{"_subject": "example.nl"}),
			},
			wantScore: models.ScoreOnbekend,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := r.Match(tc.findings)
			if got.Score != tc.wantScore {
				t.Errorf("score = %s, want %s", got.Score, tc.wantScore)
			}
			hasEvi := len(got.Evidence) > 0
			if hasEvi != tc.wantHasEvi {
				t.Errorf("evidence presence = %v, want %v", hasEvi, tc.wantHasEvi)
			}
		})
	}
}

func TestApexIPInEEA(t *testing.T) {
	r := ruleByID(t, "wand.juridisch.apex_ip_eea")
	got := r.Match([]models.Finding{
		f("d1", "dns.a", map[string]any{"_subject": "example.nl", "address": "1.2.3.4"}),
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "NL", "organisation": "TransIP"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EEA apex: score = %s, want soeverein", got.Score)
	}

	got = r.Match([]models.Finding{
		f("d1", "dns.a", map[string]any{"_subject": "example.com", "address": "5.6.7.8"}),
		f("a1", "ip.asn", map[string]any{"_subject": "example.com", "country": "US", "organisation": "Amazon"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US apex: score = %s, want afhankelijk", got.Score)
	}

	got = r.Match([]models.Finding{
		f("d1", "dns.a", map[string]any{"_subject": "example.nl", "address": "1.2.3.4"}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no ip.asn: score = %s, want onbekend", got.Score)
	}
}

func TestMXVendorJurisdiction(t *testing.T) {
	r := ruleByID(t, "wand.juridisch.mx_vendor_jurisdiction")
	got := r.Match([]models.Finding{
		f("m1", "dns.mx", map[string]any{"_subject": "example.nl", "host": "mail.example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "mail.example.nl", "country": "NL", "organisation": "TransIP"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EEA mx: score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) < 2 {
		t.Errorf("expected evidence to cite both mx and ip.asn findings; got %v", got.Evidence)
	}

	got = r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}

func TestCertValidity(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.cert_validity")
	got := r.Match([]models.Finding{
		f("v1", "tls.validity", map[string]any{"_subject": "example.nl", "days_left": 90, "not_after": time.Now().Add(90 * 24 * time.Hour)}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("valid cert: score = %s, want soeverein", got.Score)
	}

	got = r.Match([]models.Finding{
		f("v1", "tls.validity", map[string]any{"_subject": "example.nl", "expired": true, "days_left": -1}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("expired cert: score = %s, want afhankelijk", got.Score)
	}

	got = r.Match([]models.Finding{
		f("v1", "tls.validity", map[string]any{"_subject": "example.nl", "expiring_soon": true, "days_left": 10}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("expiring cert: score = %s, want voldoende", got.Score)
	}
}

func TestDNSRedundancy(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.dns_redundancy")
	got := r.Match([]models.Finding{
		f("n1", "dns.ns", map[string]any{"_subject": "example.nl", "host": "ns1.example.nl"}),
		f("n2", "dns.ns", map[string]any{"_subject": "example.nl", "host": "ns2.example.nl"}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("two ns: score = %s, want voldoende", got.Score)
	}

	got = r.Match([]models.Finding{
		f("n1", "dns.ns", map[string]any{"_subject": "example.nl", "host": "ns1.example.nl"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("one ns: score = %s, want afhankelijk", got.Score)
	}

	got = r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no ns: score = %s, want onbekend", got.Score)
	}
}

func TestCAARestricts(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.caa_restricts_issuance")
	got := r.Match([]models.Finding{
		f("c1", "dns.caa", map[string]any{"_subject": "example.nl", "tag": "issue", "value": "letsencrypt.org", "flag": 0}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("restrictive caa: score = %s, want voldoende", got.Score)
	}

	got = r.Match([]models.Finding{
		f("c1", "dns.caa", map[string]any{"_subject": "example.nl", "no_answer": true, "reason": "no CAA records"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("no caa: score = %s, want afhankelijk", got.Score)
	}
}

func TestThirdPartiesEEA(t *testing.T) {
	r := ruleByID(t, "wand.technologie.third_parties_eea")
	got := r.Match([]models.Finding{
		f("t1", "http.third_party", map[string]any{"_subject": "cdn.example.nl", "source_domain": "example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "cdn.example.nl", "country": "NL", "organisation": "Leaseweb"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EEA third party: score = %s, want soeverein", got.Score)
	}

	got = r.Match([]models.Finding{
		f("t1", "http.third_party", map[string]any{"_subject": "www.googleapis.com", "source_domain": "example.nl"}),
		f("a1", "ip.asn", map[string]any{"_subject": "www.googleapis.com", "country": "US", "organisation": "Google LLC"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US third party: score = %s, want afhankelijk", got.Score)
	}

	got = r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no third parties: score = %s, want onbekend", got.Score)
	}
}

func TestNoUSHyperscaler(t *testing.T) {
	r := ruleByID(t, "wand.technologie.no_us_hyperscaler")
	got := r.Match([]models.Finding{
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "NL", "organisation": "Leaseweb Netherlands"}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("no hyperscaler: score = %s, want voldoende", got.Score)
	}

	got = r.Match([]models.Finding{
		f("a1", "ip.asn", map[string]any{"_subject": "example.nl", "country": "US", "organisation": "Amazon.com, Inc."}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("Amazon: score = %s, want afhankelijk", got.Score)
	}
}

func TestMXPresent(t *testing.T) {
	r := ruleByID(t, "wand.data_ai.mx_present")
	got := r.Match([]models.Finding{
		f("m1", "dns.mx", map[string]any{"_subject": "example.nl", "host": "mail.example.nl"}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("with mx: score = %s, want voldoende", got.Score)
	}

	got = r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no mx: score = %s, want onbekend", got.Score)
	}
}

// Regression for the smoke-test bug: a non-resolvable domain emits
// dns.mx Findings carrying lookup-error metadata. Before the
// IsEvidenceLike filter these counted as evidence and scored voldoende.
func TestMXPresent_LookupErrorIsNotEvidence(t *testing.T) {
	r := ruleByID(t, "wand.data_ai.mx_present")
	got := r.Match([]models.Finding{
		f("m1", "dns.mx", map[string]any{"_subject": "wanderer-test-host.invalid", "error": "no such host", "kind": "nxdomain"}),
		f("m2", "dns.mx", map[string]any{"_subject": "wanderer-test-host.invalid", "no_answer": true, "reason": "domain returns NXDOMAIN"}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("nxdomain mx: score = %s, want onbekend", got.Score)
	}
	if len(got.Evidence) != 0 {
		t.Errorf("nxdomain mx: evidence = %v, want empty", got.Evidence)
	}
}

func TestOIDCFederation_AlwaysNoEvidence(t *testing.T) {
	r := ruleByID(t, "wand.data_ai.oidc_federation")
	got := r.Match([]models.Finding{
		f("x", "dns.mx", map[string]any{"_subject": "example.nl", "host": "m.example.nl"}),
	})
	if len(got.Evidence) != 0 {
		t.Errorf("oidc rule should return no evidence until egress probe ships; got %v", got.Evidence)
	}
}

func TestDefaultRules_HasTenRules(t *testing.T) {
	n := len(DefaultRules())
	if n < 10 {
		t.Errorf("DefaultRules count = %d, want >= 10", n)
	}
}

func TestEveryRuleHasRationale(t *testing.T) {
	for _, r := range DefaultRules() {
		if r.Rationale == "" {
			t.Errorf("rule %s: Rationale is empty (every wand rule must explain why it matters)", r.ID)
		}
		if r.Rationale == r.Description {
			t.Errorf("rule %s: Rationale equals Description (must add why-this-matters context, not duplicate the summary)", r.ID)
		}
	}
}

// TestFrameworkRename_RegressionGuard pins ADR-0011: every rule
// registered under the wand pack carries a `wand.` prefix on its
// CriteriumID. A future edit that accidentally re-introduces a
// `dictu.` prefix (a copy-paste from an old branch, an incomplete
// rename) breaks this test, so the rebrand we did for legal /
// reputational reasons cannot quietly regress.
func TestFrameworkRename_RegressionGuard(t *testing.T) {
	rules := DefaultRules()
	if len(rules) == 0 {
		t.Fatal("DefaultRules returned zero rules")
	}
	for _, r := range rules {
		if !strings.HasPrefix(r.ID, "wand.") {
			t.Errorf("rule %q: ID must start with `wand.` (ADR-0011)", r.ID)
		}
		if strings.HasPrefix(r.ID, "dictu.") {
			t.Errorf("rule %q: ID still starts with the deprecated `dictu.` prefix; rerun the rename per ADR-0011", r.ID)
		}
	}
}
