// Package systemd inspects systemd-managed services on the host.
// Implementation shells out to `systemctl list-units --type=service
// --all --output=json`; D-Bus integration was rejected for the MVP
// to avoid a heavy dependency.
package systemd

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Systemd is the inspector. QueryFunc is overridable for tests.
type Systemd struct {
	QueryFunc func(ctx context.Context) (string, error)
}

func (Systemd) ID() string { return "systemd" }

func (s Systemd) Available() (bool, string) {
	if s.QueryFunc != nil {
		return true, ""
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false, "systemctl not found in PATH"
	}
	return true, ""
}

func (s Systemd) Inspect(ctx context.Context) ([]models.Finding, error) {
	q := s.QueryFunc
	if q == nil {
		q = realSystemctl
	}
	raw, err := q(ctx)
	if err != nil {
		return nil, fmt.Errorf("systemd: %w", err)
	}
	return Parse(raw)
}

func realSystemctl(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(
		ctx,
		"systemctl", "list-units", "--type=service", "--all", "--output=json", "--no-pager",
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// unitJSON is the row shape `systemctl list-units --output=json` emits.
type unitJSON struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

// Parse converts the raw JSON to Findings. Exported so tests can
// drive it directly with a fixture.
func Parse(raw string) ([]models.Finding, error) {
	var rows []unitJSON
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("systemd parse: %w", err)
	}
	out := make([]models.Finding, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.Finding{
			ProbeID:       "inventory.systemd.service",
			DimensionHint: models.DimensionOperationeel,
			Subject:       r.Unit,
			Severity:      models.SeverityInfo,
			Attributes: map[string]any{
				"load_state":   r.Load,
				"active_state": r.Active,
				"sub_state":    r.Sub,
				"description":  r.Description,
			},
		})
	}
	return out, nil
}
