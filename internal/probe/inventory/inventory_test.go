package inventory

import (
	"context"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe/inventory/docker"
	"github.com/MWest2020/wanderer/internal/probe/inventory/packages"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestInspect_DockerUnavailable(t *testing.T) {
	got := Inspect(context.Background(), []Inspector{docker.Docker{}})
	if len(got) != 1 {
		t.Fatalf("want 1 unavailable finding, got %d", len(got))
	}
	if got[0].ProbeID != "inventory.docker.unavailable" {
		t.Errorf("probe = %s", got[0].ProbeID)
	}
	if got[0].SourceModus != models.SourceModusInventory {
		t.Errorf("source modus = %s", got[0].SourceModus)
	}
}

func TestInspect_TagsFindingsWithModus(t *testing.T) {
	stub := stubInspector{
		findings: []models.Finding{
			{ProbeID: "inventory.packages.dpkg", Subject: "bash", Severity: models.SeverityInfo, Attributes: map[string]any{}},
		},
	}
	got := Inspect(context.Background(), []Inspector{stub})
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].SourceModus != models.SourceModusInventory {
		t.Errorf("modus = %s", got[0].SourceModus)
	}
}

func TestInspect_OneUnavailableDoesNotBlockOthers(t *testing.T) {
	dpkg := packages.Dpkg{
		QueryFunc: func(_ context.Context) (string, error) {
			return "bash 5.2 amd64 install ok installed\n", nil
		},
	}
	got := Inspect(context.Background(), []Inspector{
		docker.Docker{}, // unavailable
		dpkg,            // works
	})
	if len(got) < 2 {
		t.Fatalf("want >=2, got %d", len(got))
	}
}

type stubInspector struct {
	findings []models.Finding
}

func (stubInspector) ID() string                                     { return "stub" }
func (stubInspector) Available() (bool, string)                      { return true, "" }
func (s stubInspector) Inspect(_ context.Context) ([]models.Finding, error) {
	return s.findings, nil
}
