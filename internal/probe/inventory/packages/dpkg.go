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
	// Tab-separated. Maintainer and Status both contain spaces;
	// a space-delimited format conflates them.
	out, err := exec.CommandContext(ctx,
		"dpkg-query", "-W", "-f=${Package}\t${Version}\t${Architecture}\t${Maintainer}\t${Status}\n",
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseDpkg parses dpkg-query output. Each line:
//
//	name<TAB>version<TAB>arch<TAB>maintainer<TAB>install ok installed
//
// The legacy space-separated three-field format is still
// accepted so pre-2026-05-11 fixtures parse.
func parseDpkg(raw string) []models.Finding {
	var out []models.Finding
	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		var name, version, arch, maintainer, status string
		if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 5)
			if len(parts) < 3 {
				continue
			}
			name, version, arch = parts[0], parts[1], parts[2]
			if len(parts) >= 4 {
				maintainer = strings.TrimSpace(parts[3])
			}
			if len(parts) == 5 {
				status = strings.TrimSpace(parts[4])
			}
		} else {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) < 3 {
				continue
			}
			name, version, arch = parts[0], parts[1], parts[2]
			if len(parts) == 4 {
				status = parts[3]
			}
		}
		// Skip half-installed packages — they are noise.
		if status != "" && !strings.Contains(status, "installed") {
			continue
		}
		sev := models.SeverityInfo
		if isEOL(name, version) {
			sev = models.SeverityObservation
		}
		attrs := map[string]any{
			"version": version,
			"arch":    arch,
			"status":  status,
		}
		if maintainer != "" {
			attrs["maintainer"] = maintainer
		}
		out = append(out, models.Finding{
			ProbeID:       "inventory.packages.dpkg",
			DimensionHint: models.DimensionOperationeel,
			Subject:       name,
			Severity:      sev,
			Attributes:    attrs,
		})
	}
	return out
}
