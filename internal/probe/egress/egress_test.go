package egress

import (
	"context"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe/egress/scanners"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubScanner struct {
	id    string
	avail bool
	cands []scanners.Candidate
}

func (s stubScanner) ID() string { return s.id }
func (s stubScanner) Available() (bool, string) {
	if s.avail {
		return true, ""
	}
	return false, "stub"
}

func (s stubScanner) Scan(_ context.Context) ([]scanners.Candidate, error) {
	return s.cands, nil
}

type stubResolver struct{}

func (stubResolver) Resolve(host string) (uint, string, string, bool) {
	if strings.Contains(host, "amazonaws.com") {
		return 16509, "Amazon.com, Inc.", "IE", true
	}
	return 0, "", "", false
}

func TestInspect_RedactsAndClassifies(t *testing.T) {
	p := Probe{
		Scanners: []scanners.Scanner{
			stubScanner{id: "configfiles", avail: true, cands: []scanners.Candidate{
				{Source: "/etc/app.env", Key: "DATABASE_URL", Value: "postgres://app:hunter2@db.example:5432/app"},
				{Source: "/etc/app.env", Key: "S3_ENDPOINT", Value: "https://s3.eu-west-1.amazonaws.com"},
				{Source: "/etc/app.env", Key: "DEBUG", Value: "true"},
			}},
		},
		Resolver: stubResolver{},
	}
	got := p.Inspect(context.Background())
	if len(got) < 2 {
		t.Fatalf("want at least 2 findings, got %d", len(got))
	}
	for _, f := range got {
		if f.SourceModus != models.SourceModusEgress {
			t.Errorf("source_modus = %s, want egress", f.SourceModus)
		}
		if v, ok := f.Attributes["value"].(string); ok && strings.Contains(v, "hunter2") {
			t.Errorf("password leaked: %v", f.Attributes)
		}
		if string(f.Evidence) != "" && strings.Contains(string(f.Evidence), "hunter2") {
			t.Errorf("password leaked into Evidence: %s", string(f.Evidence))
		}
	}
}

func TestInspect_HostResolutionAnnotation(t *testing.T) {
	p := Probe{
		Scanners: []scanners.Scanner{
			stubScanner{id: "configfiles", avail: true, cands: []scanners.Candidate{
				{Source: "/etc/app.env", Key: "S3_ENDPOINT", Value: "https://s3.eu-west-1.amazonaws.com"},
			}},
		},
		Resolver: stubResolver{},
	}
	got := p.Inspect(context.Background())
	var found bool
	for _, f := range got {
		if f.ProbeID == "egress.object_storage" {
			found = true
			if f.Attributes["country"] != "IE" {
				t.Errorf("country annotation missing: %v", f.Attributes)
			}
		}
	}
	if !found {
		t.Fatalf("egress.object_storage finding missing")
	}
}

func TestInspect_HostResolutionUnavailableEmittedOnce(t *testing.T) {
	p := Probe{
		Scanners: []scanners.Scanner{
			stubScanner{id: "configfiles", avail: true, cands: []scanners.Candidate{
				{Source: "/etc/app.env", Key: "S3_ENDPOINT", Value: "https://s3.eu-west-1.amazonaws.com"},
				{Source: "/etc/app.env", Key: "OIDC_ISSUER", Value: "https://login.example.nl/realms/x"},
			}},
		},
		Resolver: nil, // unavailable
	}
	got := p.Inspect(context.Background())
	count := 0
	for _, f := range got {
		if f.ProbeID == "egress.host_resolution.unavailable" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want exactly 1 host_resolution.unavailable, got %d", count)
	}
}

func TestInspect_EmitsClassifierRule(t *testing.T) {
	p := Probe{Scanners: []scanners.Scanner{stubScanner{id: "configfiles", avail: true, cands: []scanners.Candidate{
		{Source: "/etc/app.env", Key: "S3_ENDPOINT", Value: "https://s3.eu-west-1.amazonaws.com"},
	}}}}
	got := p.Inspect(context.Background())
	for _, f := range got {
		if f.ProbeID == "egress.object_storage" {
			if f.Attributes["classifier_rule"] != "aws_s3_region_host" {
				t.Errorf("classifier_rule missing: %v", f.Attributes)
			}
			if f.Attributes["region"] != "eu-west-1" {
				t.Errorf("region missing: %v", f.Attributes)
			}
		}
	}
}

func TestInspect_UnconfiguredScanner(t *testing.T) {
	p := Probe{
		Scanners: []scanners.Scanner{stubScanner{id: "configfiles", avail: false}},
	}
	got := p.Inspect(context.Background())
	if len(got) != 1 || got[0].ProbeID != "egress.configfiles.unconfigured" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestInspect_DropsUnknownsWithoutHost(t *testing.T) {
	p := Probe{Scanners: []scanners.Scanner{stubScanner{id: "configfiles", avail: true, cands: []scanners.Candidate{
		{Source: "/etc/app.env", Key: "DEBUG", Value: "true"},
	}}}}
	got := p.Inspect(context.Background())
	for _, f := range got {
		if f.ProbeID == "egress.unknown" {
			t.Errorf("plain DEBUG=true should not produce a finding: %+v", f)
		}
	}
}
