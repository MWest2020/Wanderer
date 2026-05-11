package fixtures

import (
	"context"
	"fmt"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// BuildAgentHost layers on top of the baseline: the conduction
// organisation gains an `alma` host target with a synthetic
// agent scan that mirrors the shape `wanderer agent --once`
// produces on a real Fedora-42 host, plus one US-telemetry hit
// (`datadog-agent`) so the host rule deep-dive shows
// `afhankelijk`.
//
// Numbers (32 packages + 14 systemd units) are intentionally
// small — the fixture's job is to exercise the rule path, not to
// stress-test the store.
func BuildAgentHost(ctx context.Context, st *store.Store) error {
	if err := BuildBaseline(ctx, st); err != nil {
		return fmt.Errorf("agent-host: base scenario: %w", err)
	}

	cond, err := st.GetOrganisationBySlug(ctx, "conduction")
	if err != nil {
		return fmt.Errorf("agent-host: lookup conduction: %w", err)
	}

	host, err := upsertTarget(ctx, st, "alma", models.TargetKindHost, cond.ID)
	if err != nil {
		return err
	}

	findings := append(agentHostFindings(), nextcloudFindings()...)
	if _, err := addCompletedScan(ctx, st, host, baseTime.Add(1*60*60_000_000_000), findings); err != nil {
		return fmt.Errorf("agent-host: scan: %w", err)
	}
	return nil
}

// nextcloudFindings returns a curated Nextcloud signal layered on
// top of the agent-host inventory. One US-hosted S3 objectstore +
// one US-hosted OIDC IdP so the three new nextcloud rules score
// afhankelijk for Playwright.
func nextcloudFindings() []models.Finding {
	return []models.Finding{
		{
			ProbeID:       "inventory.nextcloud.version",
			Subject:       "28.0.5",
			Severity:      models.SeverityInfo,
			SourceModus:   models.SourceModusInventory,
			DimensionHint: models.DimensionTechnologie,
			Attributes: map[string]any{
				"version":       "28.0.5.1",
				"versionstring": "28.0.5",
				"major":         28,
				"supported":     true,
				"edition":       "",
				"productname":   "Nextcloud",
			},
		},
		{
			ProbeID:       "inventory.nextcloud.trusted_domain",
			Subject:       "cloud.alma.local",
			Severity:      models.SeverityInfo,
			SourceModus:   models.SourceModusInventory,
			DimensionHint: models.DimensionTechnologie,
			Attributes:    map[string]any{},
		},
		{
			ProbeID:       "inventory.nextcloud.objectstore",
			Subject:       "nextcloud-data",
			Severity:      models.SeverityInfo,
			SourceModus:   models.SourceModusInventory,
			DimensionHint: models.DimensionTechnologie,
			Attributes: map[string]any{
				"class":            "OC\\Files\\ObjectStore\\S3",
				"bucket":           "nextcloud-data",
				"region":           "us-east-1",
				"endpoint":         "",
				"endpoint_host":    "s3.amazonaws.com",
				"asn":              uint(16509),
				"asn_organisation": "Amazon.com, Inc.",
				"country":          "US",
			},
		},
		{
			ProbeID:       "inventory.nextcloud.oidc_provider",
			Subject:       "okta-prod",
			Severity:      models.SeverityInfo,
			SourceModus:   models.SourceModusInventory,
			DimensionHint: models.DimensionTechnologie,
			Attributes: map[string]any{
				"client_id":          "nextcloud",
				"discovery_endpoint": "https://okta.example.com/oauth2/default/.well-known/openid-configuration",
				"issuer_url":         "https://okta.example.com/oauth2/default",
				"issuer_host":        "okta.example.com",
				"asn":                uint(16509),
				"asn_organisation":   "Okta, Inc.",
				"country":            "US",
			},
		},
	}
}

// agentHostFindings produces the curated package + systemd
// surface. Most names are open-source distro packages that stay
// off the US-telemetry list (collectd, openssh, postgresql,
// etc.); `datadog-agent` is the single hit.
func agentHostFindings() []models.Finding {
	packages := []string{
		"openssh-server", "openssh-clients", "systemd", "systemd-resolved",
		"postgresql-server", "postgresql-client", "nginx", "nginx-mod-http-perl",
		"collectd", "collectd-disk", "fail2ban", "fail2ban-systemd",
		"rsyslog", "rsyslog-relp", "chrony", "chrony-data",
		"firewalld", "firewalld-filesystem", "dnsmasq", "dnsmasq-utils",
		"podman", "podman-compose", "buildah", "skopeo",
		"kernel", "kernel-modules", "kernel-core", "kernel-headers",
		"glibc", "glibc-common", "openssl-libs",
		// One US-telemetry agent — the rule's afhankelijk path.
		"datadog-agent",
	}
	services := []string{
		"sshd.service", "systemd-resolved.service", "postgresql.service",
		"nginx.service", "collectd.service", "fail2ban.service",
		"rsyslog.service", "chronyd.service", "firewalld.service",
		"dnsmasq.service", "podman.socket", "auditd.service",
		// One US-telemetry service — same vendor as the package.
		"datadog-agent.service",
		// And one open-source agent that stays off the list.
		"nginx-prometheus-exporter.service",
	}

	var out []models.Finding
	for _, p := range packages {
		out = append(out, mkInventoryFinding("inventory.packages.rpm", p, map[string]any{
			"version": "1.0.0-1.fc42",
			"arch":    "x86_64",
		}))
	}
	for _, s := range services {
		out = append(out, mkInventoryFinding("inventory.systemd.service", s, map[string]any{
			"active": "active",
		}))
	}
	return out
}
