package whois

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

const sampleRDAP = `{
  "objectClassName": "domain",
  "ldhName": "example.nl",
  "entities": [
    {
      "roles": ["registrant"],
      "vcardArray": ["vcard", [
        ["version", {}, "text", "4.0"],
        ["adr", {"cc": "NL"}, "text", ["", "", "Strawinskylaan 1", "Amsterdam", "", "1077XX", "Netherlands"]]
      ]]
    },
    {
      "roles": ["registrar"],
      "vcardArray": ["vcard", [
        ["fn", {}, "text", "TransIP B.V."]
      ]]
    }
  ]
}`

func TestRun_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write([]byte(sampleRDAP))
	}))
	defer srv.Close()

	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	probes := map[string]bool{}
	for _, f := range got {
		probes[f.ProbeID] = true
		if f.ProbeID == "whois.registrant" {
			if f.Attributes["country"] != "NL" {
				t.Errorf("country = %v", f.Attributes["country"])
			}
		}
		if f.ProbeID == "whois.registrar" {
			if f.Attributes["name"] != "TransIP B.V." {
				t.Errorf("name = %v", f.Attributes["name"])
			}
		}
	}
	if !probes["whois.registrant"] {
		t.Errorf("missing whois.registrant")
	}
	if !probes["whois.registrar"] {
		t.Errorf("missing whois.registrar")
	}
}

func TestRun_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(got) != 1 || got[0].ProbeID != "whois.unavailable" {
		t.Errorf("expected unavailable; got %+v", got)
	}
}

func TestRun_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, _ := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if len(got) != 1 || got[0].ProbeID != "whois.unavailable" {
		t.Errorf("expected unavailable on bad JSON; got %+v", got)
	}
}

func TestRun_EmptyDomain(t *testing.T) {
	p := &Probe{}
	_, err := p.Run(context.Background(), models.Target{}, probe.Config{})
	if err == nil {
		t.Errorf("expected error on empty domain")
	}
}

func TestRun_NoEntities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ldhName":"example.nl"}`))
	}))
	defer srv.Close()
	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, _ := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if len(got) != 1 || got[0].ProbeID != "whois.unavailable" {
		t.Errorf("no entities should produce unavailable; got %+v", got)
	}
}
