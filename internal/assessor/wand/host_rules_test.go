package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func hostFinding(id, probeID, subject string, attrs map[string]any) models.Finding {
	if attrs == nil {
		attrs = map[string]any{}
	}
	return models.Finding{
		ID:         id,
		ProbeID:    probeID,
		Subject:    subject,
		Severity:   models.SeverityFinding,
		Attributes: attrs,
	}
}

func TestHostNoUSTelemetryPackages_Soeverein(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_packages")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "nginx", nil),
		hostFinding("p2", "inventory.packages.dpkg", "openssh-server", nil),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("clean host: score = %s, want soeverein", got.Score)
	}
	// Soeverein verdict MUST cite at least one inspected finding ID
	// as negative evidence — the assessor engine forces verdicts
	// without Evidence back to onbekend, so a "clean host" path
	// that returned zero IDs would silently degrade.
	if len(got.Evidence) == 0 {
		t.Errorf("clean host: evidence is empty; soeverein without evidence becomes onbekend after engine normalisation")
	}
	if !strings.Contains(got.Verdict, "inspected 2 packages") {
		t.Errorf("verdict = %q, want operator-readable count of inspected findings", got.Verdict)
	}
}

func TestHostNoUSTelemetryPackages_Afhankelijk(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_packages")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "nginx", nil),
		hostFinding("p2", "inventory.packages.rpm", "datadog-agent", nil),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("datadog hit: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "datadog-agent") {
		t.Errorf("verdict = %q, must name the matched package", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "Datadog") {
		t.Errorf("verdict = %q, must name the vendor of record", got.Verdict)
	}
	if len(got.Evidence) != 1 || got.Evidence[0] != "p2" {
		t.Errorf("evidence = %v, want [p2]", got.Evidence)
	}
}

func TestHostNoUSTelemetryPackages_DeterministicOrder(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_packages")
	findings := []models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "splunkforwarder", nil),
		hostFinding("p2", "inventory.packages.rpm", "datadog-agent", nil),
		hostFinding("p3", "inventory.packages.rpm", "newrelic-infra", nil),
	}
	got := r.Match(findings)
	if got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk", got.Score)
	}
	idx := func(s string) int { return strings.Index(got.Verdict, s) }
	if !(idx("datadog-agent") < idx("newrelic-infra") && idx("newrelic-infra") < idx("splunkforwarder")) {
		t.Errorf("verdict = %q must list hits alphabetically by subject", got.Verdict)
	}
}

func TestHostNoUSTelemetryPackages_NoFindingsIsOnbekend(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_packages")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("nil findings: score = %s, want onbekend", got.Score)
	}
}

func TestHostNoUSTelemetryPackages_OnlyMetaIsOnbekend(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_packages")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm.unavailable", "rpm", map[string]any{"unavailable": true}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("only meta findings: score = %s, want onbekend", got.Score)
	}
}

func TestHostNoUSTelemetryServices_Soeverein(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_services")
	got := r.Match([]models.Finding{
		hostFinding("s1", "inventory.systemd.service", "nginx.service", nil),
		hostFinding("s2", "inventory.systemd.service", "ssh.service", nil),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("clean host: score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Errorf("clean host: evidence is empty; engine forces evidence-less verdicts to onbekend")
	}
	if !strings.Contains(got.Verdict, "inspected 2 systemd units") {
		t.Errorf("verdict = %q, want operator-readable count of inspected findings", got.Verdict)
	}
}

func TestHostNoUSTelemetryServices_Afhankelijk(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_services")
	got := r.Match([]models.Finding{
		hostFinding("s1", "inventory.systemd.service", "nginx.service", nil),
		hostFinding("s2", "inventory.systemd.service", "datadog-agent.service", nil),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("datadog hit: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "datadog-agent.service") {
		t.Errorf("verdict = %q must name the matched unit", got.Verdict)
	}
}

func TestHostNoUSTelemetryServices_PackageFindingsIgnored(t *testing.T) {
	r := ruleByID(t, "wand.host.no_us_telemetry_services")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "datadog-agent", nil),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("only package findings: score = %s, want onbekend (services rule ignores packages)", got.Score)
	}
}
