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
	// Tab-separated so the VENDOR field (which can contain
	// spaces and commas, e.g. "Red Hat, Inc.") parses cleanly.
	out, err := exec.CommandContext(
		ctx,
		"rpm", "-qa", "--qf=%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\t%{VENDOR}\n",
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseRpm parses tab-separated `name<TAB>version-release<TAB>arch<TAB>vendor`
// lines. Older callers (pre-2026-05-11) used space-separated
// three-field output without a vendor; that shape is still
// accepted so a stored fixture or a test parses cleanly.
func parseRpm(raw string) []models.Finding {
	var out []models.Finding
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		var name, version, arch, vendor string
		if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) < 3 {
				continue
			}
			name, version, arch = parts[0], parts[1], parts[2]
			if len(parts) == 4 {
				vendor = strings.TrimSpace(parts[3])
			}
		} else {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			name, version, arch = parts[0], parts[1], parts[2]
		}
		sev := models.SeverityInfo
		if isEOL(name, version) {
			sev = models.SeverityObservation
		}
		attrs := map[string]any{
			"version": version,
			"arch":    arch,
		}
		// RPM's Vendor field is "(none)" when no vendor is set
		// (locally-built RPMs, srcpms). Skip the placeholder so
		// the assessor doesn't classify on noise.
		if vendor != "" && vendor != "(none)" {
			attrs["vendor"] = vendor
		}
		out = append(out, models.Finding{
			ProbeID:       "inventory.packages.rpm",
			DimensionHint: models.DimensionOperationeel,
			Subject:       name,
			Severity:      sev,
			Attributes:    attrs,
		})
	}
	return out
}
