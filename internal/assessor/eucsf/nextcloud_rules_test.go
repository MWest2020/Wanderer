package eucsf

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func ncFinding(id, probeID, subject string, attrs map[string]any) models.Finding {
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

func TestNextcloudSupplyChain_Soeverein(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.nextcloud_supply_chain")
	got := r.Match([]models.Finding{
		ncFinding("o1", "inventory.nextcloud.objectstore", "data", map[string]any{"country": "NL"}),
		ncFinding("p1", "inventory.nextcloud.oidc_provider", "keycloak", map[string]any{"country": "DE"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("all-EU: score = %s, want soeverein", got.Score)
	}
	if !strings.Contains(got.Verdict, "1 objectstore + 1 OIDC") {
		t.Errorf("verdict = %q must include the inspected mix", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "[SEAL 4]") {
		t.Errorf("verdict = %q must carry SEAL tag", got.Verdict)
	}
}

func TestNextcloudSupplyChain_ObjectstoreHit(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.nextcloud_supply_chain")
	got := r.Match([]models.Finding{
		ncFinding("o1", "inventory.nextcloud.objectstore", "data", map[string]any{"country": "US"}),
		ncFinding("p1", "inventory.nextcloud.oidc_provider", "keycloak", map[string]any{"country": "NL"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("US objectstore: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "objectstore data") {
		t.Errorf("verdict = %q must name the offending backend", got.Verdict)
	}
}

func TestNextcloudSupplyChain_NoFindingsIsOnbekend(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.nextcloud_supply_chain")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}
