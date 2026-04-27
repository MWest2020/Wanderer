// Package packages parses dpkg and rpm output to inventory installed
// packages. Both inspectors share a tiny EOL-version lookup table so
// reviewers can see at a glance what counts as a concern.
package packages

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Dpkg is the dpkg-based packages inspector.
type Dpkg struct {
	// QueryFunc returns the raw `dpkg-query -W ...` output. Defaults to
	// invoking the real binary; tests override this.
	QueryFunc func(ctx context.Context) (string, error)
}

// ID implements inventory.Inspector.
func (Dpkg) ID() string { return "packages.dpkg" }

// Available reports whether dpkg-query is callable on this host.
func (d Dpkg) Available() (bool, string) {
	if d.QueryFunc != nil {
		return true, ""
	}
	if _, err := exec.LookPath("dpkg-query"); err != nil {
		return false, "dpkg-query not found in PATH"
	}
	return true, ""
}

// Inspect runs dpkg-query and returns one Finding per installed
// package.
func (d Dpkg) Inspect(ctx context.Context) ([]models.Finding, error) {
	q := d.QueryFunc
	if q == nil {
		q = realDpkgQuery
	}
	raw, err := q(ctx)
	if err != nil {
		return nil, fmt.Errorf("dpkg: %w", err)
	}
	return parseDpkg(raw), nil
}

func realDpkgQuery(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx,
		"dpkg-query", "-W", "-f=${Package} ${Version} ${Architecture} ${Status}\n",
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseDpkg parses the output of dpkg-query. Each line:
//
//	<name> <version> <arch> install ok installed
//
// The "Status" field is variable; we keep it whole as a single string
// in the finding so reviewers can see the raw upstream value.
func parseDpkg(raw string) []models.Finding {
	var out []models.Finding
	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 3 {
			continue
		}
		name, version, arch := parts[0], parts[1], parts[2]
		status := ""
		if len(parts) == 4 {
			status = parts[3]
		}
		// Skip half-installed packages — they are noise.
		if status != "" && !strings.Contains(status, "installed") {
			continue
		}
		sev := models.SeverityInfo
		if isEOL(name, version) {
			sev = models.SeverityObservation
		}
		out = append(out, models.Finding{
			ProbeID:       "inventory.packages.dpkg",
			DimensionHint: models.DimensionOperationeel,
			Subject:       name,
			Severity:      sev,
			Attributes: map[string]any{
				"version": version,
				"arch":    arch,
				"status":  status,
			},
		})
	}
	return out
}
