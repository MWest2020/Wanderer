package eucsf

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/assessor"
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
	if got.Reason != "" {
		t.Errorf("no findings: reason = %q, want empty (this is 'not configured', not 'unreadable')", got.Reason)
	}
}

func TestNextcloudSupplyChain_UnreadableSystemConfig(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.nextcloud_supply_chain")
	got := r.Match([]models.Finding{
		ncFinding("u1", "inventory.nextcloud.system_config.unreadable", "config:list system", map[string]any{
			"unavailable": true,
			"reason":      "occ config:list system --output=json unreadable: invalid character 'D' looking for beginning of value (58 bytes)",
		}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("unreadable output: score = %s, want onbekend", got.Score)
	}
	if got.Reason != assessor.ReasonScannerUnreadableOutput {
		t.Errorf("unreadable output: reason = %q, want %q", got.Reason, assessor.ReasonScannerUnreadableOutput)
	}
	if strings.Contains(got.Verdict, "no relevant configuration") {
		t.Errorf("verdict = %q must not read as 'nothing configured'", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "could not read") {
		t.Errorf("verdict = %q must say the scanner could not read the output", got.Verdict)
	}
}
