// Package nextcloud inspects a local Nextcloud installation by
// shelling out to its `occ` admin CLI. The MVP ships the parser and
// the Available() check; on hosts without `occ` (or without a
// configured path) the inspector reports unavailable.
package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Nextcloud is the inspector.
type Nextcloud struct {
	OccPath string
	RunAs   string
	// QueryFunc returns the raw `occ app:list --output=json` output.
	// When nil the inspector shells out at runtime.
	QueryFunc func(ctx context.Context) (string, error)
}

func (Nextcloud) ID() string { return "nextcloud" }

func (n Nextcloud) Available() (bool, string) {
	if n.QueryFunc != nil {
		return true, ""
	}
	if n.OccPath == "" {
		return false, "occ path not configured"
	}
	if _, err := exec.LookPath(n.OccPath); err != nil {
		return false, "occ not callable: " + err.Error()
	}
	return true, ""
}

func (n Nextcloud) Inspect(ctx context.Context) ([]models.Finding, error) {
	q := n.QueryFunc
	if q == nil {
		q = func(ctx context.Context) (string, error) {
			args := []string{"app:list", "--output=json"}
			cmd := n.OccPath
			if n.RunAs != "" {
				args = append([]string{"-u", n.RunAs, n.OccPath}, args...)
				cmd = "sudo"
			}
			out, err := exec.CommandContext(ctx, cmd, args...).Output()
			if err != nil {
				return "", err
			}
			return string(out), nil
		}
	}
	raw, err := q(ctx)
	if err != nil {
		return nil, fmt.Errorf("nextcloud: %w", err)
	}
	return Parse(raw)
}

// occListJSON is the {enabled: {}, disabled: {}} shape `occ app:list`
// emits as of Nextcloud 28.
type occListJSON struct {
	Enabled  map[string]string `json:"enabled"`
	Disabled map[string]string `json:"disabled"`
}

// Parse converts `occ app:list --output=json` output to Findings.
func Parse(raw string) ([]models.Finding, error) {
	var doc occListJSON
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("nextcloud parse: %w", err)
	}
	var out []models.Finding
	for app, version := range doc.Enabled {
		out = append(out, finding(app, version, true))
	}
	for app, version := range doc.Disabled {
		out = append(out, finding(app, version, false))
	}
	return out, nil
}

func finding(app, version string, enabled bool) models.Finding {
	return models.Finding{
		ProbeID:       "inventory.nextcloud.app",
		DimensionHint: models.DimensionTechnologie,
		Subject:       app,
		Severity:      models.SeverityInfo,
		Attributes: map[string]any{
			"version": version,
			"enabled": enabled,
		},
	}
}
