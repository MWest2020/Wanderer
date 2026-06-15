package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func secHdrFinding(id string, missing []string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "http.security_headers", Subject: "example.nl",
		Severity:   models.SeverityObservation,
		Attributes: map[string]any{"missing": missing, "present": map[string]string{}},
	}
}

func respFinding(id, server, poweredBy string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "http.response", Subject: "example.nl",
		Attributes: map[string]any{"server": server, "powered_by": poweredBy},
	}
}

func TestHTTPExposure_MissingHSTSAfhankelijk(t *testing.T) {
	got := httpExposure().Match([]models.Finding{
		secHdrFinding("s1", []string{"Strict-Transport-Security", "Content-Security-Policy"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "HSTS") || len(got.Evidence) == 0 {
		t.Errorf("verdict/evidence: %q %v", got.Verdict, got.Evidence)
	}
}

func TestHTTPExposure_HSTSPresentOthersMissingVoldoende(t *testing.T) {
	got := httpExposure().Match([]models.Finding{
		secHdrFinding("s1", []string{"Content-Security-Policy"}),
	})
	if got.Score != models.ScoreVoldoende {
		t.Fatalf("score = %s, want voldoende", got.Score)
	}
}

func TestHTTPExposure_AllPresentSoeverein(t *testing.T) {
	got := httpExposure().Match([]models.Finding{secHdrFinding("s1", nil)})
	if got.Score != models.ScoreSoeverein {
		t.Fatalf("score = %s, want soeverein", got.Score)
	}
}

func TestHTTPExposure_BannerDisclosureNoted(t *testing.T) {
	got := httpExposure().Match([]models.Finding{
		secHdrFinding("s1", nil),
		respFinding("r1", "Apache/2.4.41 (Ubuntu)", "PHP/8.1"),
	})
	if !strings.Contains(got.Verdict, "discloses stack") || !strings.Contains(got.Verdict, "Apache/2.4.41") {
		t.Errorf("verdict should note banner: %q", got.Verdict)
	}
}

func TestHTTPExposure_NoFindingOnbekend(t *testing.T) {
	if got := httpExposure().Match([]models.Finding{respFinding("r1", "nginx", "")}); got.Score != models.ScoreOnbekend {
		t.Fatalf("score = %s, want onbekend", got.Score)
	}
}

// store round-trip: "missing" arrives as []any of strings.
func TestHTTPExposure_StoreRoundTripMissingSlice(t *testing.T) {
	f := models.Finding{
		ID: "s1", ProbeID: "http.security_headers", Severity: models.SeverityObservation,
		Attributes: map[string]any{"missing": []any{"Strict-Transport-Security"}},
	}
	if got := httpExposure().Match([]models.Finding{f}); got.Score != models.ScoreAfhankelijk {
		t.Fatalf("score = %s, want afhankelijk (([]any) missing handled)", got.Score)
	}
}
