package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func nextcloudFinding(id, probeID, subject string, attrs map[string]any) models.Finding {
	if attrs == nil {
		attrs = map[string]any{}
	}
	return models.Finding{
		ID:         id,
		ProbeID:    probeID,
		Subject:    subject,
		Severity:   models.SeverityInfo,
		Attributes: attrs,
	}
}

func TestObjectstoreEU_Soeverein(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.objectstore_eu")
	got := r.Match([]models.Finding{
		nextcloudFinding("o1", "inventory.nextcloud.objectstore", "data", map[string]any{
			"country": "NL",
			"endpoint_host": "s3.transip.eu",
		}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU objectstore: score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Errorf("soeverein verdict must cite negative evidence")
	}
	if !strings.Contains(got.Verdict, "inspected 1") {
		t.Errorf("verdict = %q must include inspected count", got.Verdict)
	}
}

func TestObjectstoreEU_Afhankelijk(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.objectstore_eu")
	got := r.Match([]models.Finding{
		nextcloudFinding("o1", "inventory.nextcloud.objectstore", "data", map[string]any{
			"country": "US",
			"endpoint_host": "s3.amazonaws.com",
		}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US objectstore: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "data") {
		t.Errorf("verdict = %q must name the offending bucket", got.Verdict)
	}
	if len(got.Evidence) != 1 || got.Evidence[0] != "o1" {
		t.Errorf("evidence = %v, want [o1]", got.Evidence)
	}
}

func TestObjectstoreEU_NoFindingsIsOnbekend(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.objectstore_eu")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}

func TestObjectstoreEU_MissingCountryTreatedAsNonEEA(t *testing.T) {
	// The geoip resolver did not populate `country` — treat
	// missing as non-EEA so an undocumented backend does not
	// silently score soeverein.
	r := ruleByID(t, "wand.nextcloud.objectstore_eu")
	got := r.Match([]models.Finding{
		nextcloudFinding("o1", "inventory.nextcloud.objectstore", "data", map[string]any{
			"endpoint_host": "minio.internal",
		}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("missing country: score = %s, want afhankelijk (better safe than soeverein)", got.Score)
	}
}

func TestOIDCProviderEU_Afhankelijk(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.oidc_provider_eu")
	got := r.Match([]models.Finding{
		nextcloudFinding("p1", "inventory.nextcloud.oidc_provider", "okta-prod", map[string]any{
			"country":     "US",
			"issuer_host": "okta.example.com",
		}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US IdP: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "okta-prod") {
		t.Errorf("verdict = %q must name the offending IdP", got.Verdict)
	}
}

func TestOIDCProviderEU_Soeverein(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.oidc_provider_eu")
	got := r.Match([]models.Finding{
		nextcloudFinding("p1", "inventory.nextcloud.oidc_provider", "keycloak", map[string]any{
			"country":     "NL",
			"issuer_host": "login.example.nl",
		}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU IdP: score = %s, want soeverein", got.Score)
	}
}

func TestOIDCProviderEU_UnavailableMentionsAlternative(t *testing.T) {
	r := ruleByID(t, "wand.nextcloud.oidc_provider_eu")
	got := r.Match([]models.Finding{
		{
			ID:       "u1",
			ProbeID:  "inventory.nextcloud.oidc.unavailable",
			Subject:  "user_oidc",
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"unavailable":     true,
				"alternative_app": "social_login",
				"reason":          "occ command missing",
			},
		},
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("unavailable: score = %s, want onbekend", got.Score)
	}
	if !strings.Contains(got.Verdict, "social_login") {
		t.Errorf("verdict = %q must name the alternative app", got.Verdict)
	}
}

func TestDefaultRules_RegistersNextcloudRules(t *testing.T) {
	want := map[string]bool{
		"wand.nextcloud.objectstore_eu":   false,
		"wand.nextcloud.oidc_provider_eu": false,
	}
	for _, r := range DefaultRules() {
		if _, ok := want[r.ID]; ok {
			want[r.ID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("nextcloud rule %q not registered in DefaultRules", id)
		}
	}
}
