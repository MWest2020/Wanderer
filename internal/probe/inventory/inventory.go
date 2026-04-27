// Package inventory hosts the agent-side inspectors that report what
// is installed and running on the host the agent runs on. Each
// inspector lives in its own sub-package so adding (or skipping) one
// is a localised change. The Inspect helper runs every enabled
// inspector and returns the merged Finding list, tagging each
// Finding with SourceModus = "inventory".
package inventory

import (
	"context"
	"os"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Inspector is one host-side inspector (systemd, docker, ...).
type Inspector interface {
	// ID is a stable short identifier used in ProbeID prefixes
	// (e.g. "systemd" → "inventory.systemd.service").
	ID() string
	// Available reports whether this inspector can run on the
	// current host. An unavailable inspector emits a single
	// inventory.<id>.unavailable Finding instead of skipping silently.
	Available() (bool, string)
	// Inspect runs the actual data collection. It is called only
	// when Available() returned true.
	Inspect(ctx context.Context) ([]models.Finding, error)
}

// Inspect runs each Inspector and merges the results. Findings are
// tagged with SourceModus = inventory; unavailable inspectors are
// surfaced as info-severity findings rather than dropped.
func Inspect(ctx context.Context, inspectors []Inspector) []models.Finding {
	host := hostname()
	var out []models.Finding
	for _, ins := range inspectors {
		if ok, reason := ins.Available(); !ok {
			out = append(out, models.Finding{
				ProbeID:     "inventory." + ins.ID() + ".unavailable",
				SourceModus: models.SourceModusInventory,
				Subject:     host,
				Severity:    models.SeverityInfo,
				Attributes:  map[string]any{"reason": reason},
			})
			continue
		}
		findings, err := ins.Inspect(ctx)
		if err != nil {
			out = append(out, models.Finding{
				ProbeID:     "inventory." + ins.ID() + ".error",
				SourceModus: models.SourceModusInventory,
				Subject:     host,
				Severity:    models.SeverityInfo,
				Attributes:  map[string]any{"error": err.Error()},
			})
			continue
		}
		for i := range findings {
			findings[i].SourceModus = models.SourceModusInventory
			if findings[i].Subject == "" {
				findings[i].Subject = host
			}
		}
		out = append(out, findings...)
	}
	return out
}

func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
