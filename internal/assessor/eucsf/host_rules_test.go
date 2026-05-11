package eucsf

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

func TestHostNoUSTelemetry_Soeverein(t *testing.T) {
	r := ruleByID(t, "eucsf.sov5.host_no_us_telemetry")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "nginx", nil),
		hostFinding("s1", "inventory.systemd.service", "nginx.service", nil),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("clean host: score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Errorf("clean host: evidence is empty; engine forces evidence-less verdicts to onbekend")
	}
	if !strings.Contains(got.Verdict, "inspected 1 packages + 1 systemd units") {
		t.Errorf("verdict = %q, want operator-readable count of inspected findings", got.Verdict)
	}
}

func TestHostNoUSTelemetry_PackageHit(t *testing.T) {
	r := ruleByID(t, "eucsf.sov5.host_no_us_telemetry")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "datadog-agent", nil),
		hostFinding("s1", "inventory.systemd.service", "nginx.service", nil),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("package hit: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "datadog-agent") {
		t.Errorf("verdict = %q must reference matched package", got.Verdict)
	}
	if len(got.Evidence) != 1 || got.Evidence[0] != "p1" {
		t.Errorf("evidence = %v, want [p1]", got.Evidence)
	}
}

func TestHostNoUSTelemetry_ServiceHit(t *testing.T) {
	r := ruleByID(t, "eucsf.sov5.host_no_us_telemetry")
	got := r.Match([]models.Finding{
		hostFinding("p1", "inventory.packages.rpm", "nginx", nil),
		hostFinding("s1", "inventory.systemd.service", "splunkforwarder.service", nil),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("service hit: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "splunkforwarder.service") {
		t.Errorf("verdict = %q must reference matched unit", got.Verdict)
	}
}

func TestHostNoUSTelemetry_BothShapesMissing(t *testing.T) {
	r := ruleByID(t, "eucsf.sov5.host_no_us_telemetry")
	got := r.Match([]models.Finding{
		hostFinding("a1", "ip.asn", "example.nl", map[string]any{"country": "NL"}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("only perimeter findings: score = %s, want onbekend (rule needs host shapes)", got.Score)
	}
}

func TestHostNoUSTelemetry_DeterministicVerdict(t *testing.T) {
	r := ruleByID(t, "eucsf.sov5.host_no_us_telemetry")
	got := r.Match([]models.Finding{
		hostFinding("s1", "inventory.systemd.service", "datadog-agent.service", nil),
		hostFinding("p1", "inventory.packages.rpm", "splunkforwarder", nil),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "package splunkforwarder") {
		t.Errorf("verdict = %q must list package shape", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "service datadog-agent.service") {
		t.Errorf("verdict = %q must list service shape", got.Verdict)
	}
	if pkgIdx, svcIdx := strings.Index(got.Verdict, "package "), strings.Index(got.Verdict, "service "); !(pkgIdx >= 0 && svcIdx >= 0 && pkgIdx < svcIdx) {
		t.Errorf("verdict = %q must list package shape before service shape (deterministic order)", got.Verdict)
	}
}
