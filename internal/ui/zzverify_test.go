package ui_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/fixtures"
	"github.com/MWest2020/wanderer/internal/ui"
)

func TestZZVerifyVlootEnRegelsFixture(t *testing.T) {
	st := newTestStore(t)
	if err := fixtures.BuildBaseline(context.Background(), st); err != nil {
		t.Fatalf("build baseline: %v", err)
	}
	h, err := ui.Handler(st, ui.Options{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	get := func(path string) string {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Fatalf("get %s: status %d body:\n%s", path, resp.StatusCode, string(b))
		}
		return string(b)
	}

	fleet := get("/ui/orgs/voorbeeld/fleet?sort=score")
	t.Logf("=== fleet sort=score ===\n%s", fleet)
	fleetLastScan := get("/ui/orgs/voorbeeld/fleet?sort=last_scan")
	t.Logf("=== fleet sort=last_scan ===\n%s", fleetLastScan)

	rule := get("/ui/reporting/wand/wand.operationeel.domain_expiry")
	t.Logf("=== rule domain_expiry ===\n%s", rule)
	for _, want := range []string{"90", "30", "acme.example.com", "score-afhankelijk", "answer-remediation", "Verleng de domeinregistratie van acme.example.com"} {
		if !strings.Contains(rule, want) {
			t.Errorf("rule page missing %q", want)
		}
	}
}
