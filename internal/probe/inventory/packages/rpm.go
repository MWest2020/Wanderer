package packages

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Rpm is the rpm-based packages inspector.
type Rpm struct {
	QueryFunc func(ctx context.Context) (string, error)
}

func (Rpm) ID() string { return "packages.rpm" }

func (r Rpm) Available() (bool, string) {
	if r.QueryFunc != nil {
		return true, ""
	}
	if _, err := exec.LookPath("rpm"); err != nil {
		return false, "rpm not found in PATH"
	}
	return true, ""
}

func (r Rpm) Inspect(ctx context.Context) ([]models.Finding, error) {
	q := r.QueryFunc
	if q == nil {
		q = realRpmQuery
	}
	raw, err := q(ctx)
	if err != nil {
		return nil, fmt.Errorf("rpm: %w", err)
	}
	return parseRpm(raw), nil
}

func realRpmQuery(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx,
		"rpm", "-qa", "--qf=%{NAME} %{VERSION}-%{RELEASE} %{ARCH}\n",
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseRpm parses lines of the form `name version-release arch`.
func parseRpm(raw string) []models.Finding {
	var out []models.Finding
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		name, version, arch := parts[0], parts[1], parts[2]
		sev := models.SeverityInfo
		if isEOL(name, version) {
			sev = models.SeverityObservation
		}
		out = append(out, models.Finding{
			ProbeID:       "inventory.packages.rpm",
			DimensionHint: models.DimensionOperationeel,
			Subject:       name,
			Severity:      sev,
			Attributes: map[string]any{
				"version": version,
				"arch":    arch,
			},
		})
	}
	return out
}
