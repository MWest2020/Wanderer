package eucsf

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestContainerSupplyChain_Soeverein(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.container_supply_chain")
	got := r.Match([]models.Finding{
		{
			ID:       "i1",
			ProbeID:  "inventory.docker.image",
			Subject:  "harbor.example.de/team/app:v3",
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"repo_tags": []string{"harbor.example.de/team/app:v3"},
			},
		},
		{
			ID:       "c1",
			ProbeID:  "inventory.docker.container",
			Subject:  "app",
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"image": "harbor.example.de/team/app:v3",
			},
		},
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU registry: score = %s, want soeverein", got.Score)
	}
	if !strings.Contains(got.Verdict, "1 images + 1 containers") {
		t.Errorf("verdict = %q must include both inspected counts", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "[SEAL 4]") {
		t.Errorf("verdict = %q must carry SEAL tag", got.Verdict)
	}
}

func TestContainerSupplyChain_BothHits(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.container_supply_chain")
	got := r.Match([]models.Finding{
		{
			ID:       "i1",
			ProbeID:  "inventory.docker.image",
			Subject:  "gcr.io/foo/bar:v1",
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"repo_tags": []string{"gcr.io/foo/bar:v1"},
			},
		},
		{
			ID:       "c1",
			ProbeID:  "inventory.docker.container",
			Subject:  "app",
			Severity: models.SeverityInfo,
			Attributes: map[string]any{
				"image": "gcr.io/foo/bar:v1",
			},
		},
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("gcr hits: score = %s, want afhankelijk", got.Score)
	}
}

func TestContainerSupplyChain_NoFindings(t *testing.T) {
	r := ruleByID(t, "eucsf.sov6.container_supply_chain")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}
