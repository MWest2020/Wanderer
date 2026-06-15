package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func transitHop(id string, hop int, country, org string) models.Finding {
	return models.Finding{
		ID:       id,
		ProbeID:  "transit.hop",
		Subject:  "example.nl",
		Severity: models.SeverityFinding,
		Attributes: map[string]any{
			"hop": hop, "ip": "203.0.113." + id, "country": country, "organisation": org,
		},
	}
}

func TestTransitEUPath_EEADestinationSoeverein(t *testing.T) {
	r := transitEUPath()
	got := r.Match([]models.Finding{
		transitHop("1", 1, "NL", "Ziggo"),
		transitHop("9", 9, "NL", "CYSO"), // destination, EEA
	})
	if got.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Error("soeverein verdict must cite evidence")
	}
	if !strings.Contains(got.Verdict, "CYSO") {
		t.Errorf("verdict should name the destination host: %q", got.Verdict)
	}
}

func TestTransitEUPath_NonEEADestinationAfhankelijk(t *testing.T) {
	r := transitEUPath()
	got := r.Match([]models.Finding{
		transitHop("1", 1, "NL", "Ziggo"),
		transitHop("8", 8, "US", "Amazon"), // destination, non-EEA
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "Amazon") || !strings.Contains(got.Verdict, "non-EEA") {
		t.Errorf("verdict should name the non-EEA destination: %q", got.Verdict)
	}
}

func TestTransitEUPath_EEADestNamesNonEEATransit(t *testing.T) {
	r := transitEUPath()
	got := r.Match([]models.Finding{
		transitHop("5", 5, "US", "NTT America"), // non-EEA transit
		transitHop("9", 9, "NL", "CYSO"),        // EEA destination
	})
	if got.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein (transit does not downgrade)", got.Score)
	}
	if !strings.Contains(got.Verdict, "transits non-EEA") || !strings.Contains(got.Verdict, "NTT") {
		t.Errorf("verdict should mention the non-EEA transit hop: %q", got.Verdict)
	}
}

func TestTransitEUPath_NoGeoOnbekend(t *testing.T) {
	r := transitEUPath()
	// Hops without a country (no GeoIP) → onbekend.
	got := r.Match([]models.Finding{
		{ID: "x", ProbeID: "transit.hop", Severity: models.SeverityFinding, Attributes: map[string]any{"hop": 1, "ip": "10.0.0.1"}},
	})
	if got.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend", got.Score)
	}
}
