package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpprobe "github.com/MWest2020/wanderer/internal/probe/http"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

const homepage = `<!doctype html><html><head>
<title>T</title>
<link rel="stylesheet" href="/static/app.css">
<link rel="stylesheet" href="https://cdn.example.com/lib.css">
<script src="https://analytics.example.com/a.js"></script>
<script src="/local.js"></script>
</head><body>
<img src="https://images.example.net/logo.png">
</body></html>`

func TestHTTPSFetchAndExtract(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			http.NotFound(w, r)
		default:
			w.Header().Set("Server", "test/1.0")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
			_, _ = w.Write([]byte(homepage))
		}
	}))
	defer ts.Close()

	// Strip scheme, map test server cert via its client.
	host := strings.TrimPrefix(ts.URL, "https://")
	p := &httpprobe.Probe{Client: ts.Client()}
	// The probe builds URL "https://<domain>/". We pass the
	// host:port pair as the domain so the URL ends up at the test
	// server.
	findings, err := p.Run(context.Background(), models.Target{Domain: host}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	kinds := map[string]int{}
	hosts := map[string]struct{}{}
	for _, f := range findings {
		kinds[f.ProbeID]++
		if f.ProbeID == "http.third_party" {
			hosts[f.Subject] = struct{}{}
		}
	}
	if kinds["http.response"] == 0 {
		t.Error("no http.response finding")
	}
	if kinds["http.security_headers"] == 0 {
		t.Error("no http.security_headers finding")
	}
	if _, ok := hosts["cdn.example.com"]; !ok {
		t.Errorf("cdn.example.com not seen in third parties: %v", hosts)
	}
	if _, ok := hosts["analytics.example.com"]; !ok {
		t.Errorf("analytics.example.com not seen in third parties: %v", hosts)
	}
	if _, ok := hosts["images.example.net"]; !ok {
		t.Errorf("images.example.net not seen in third parties: %v", hosts)
	}
}

func TestRobotsBlocked(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
		default:
			t.Errorf("unexpected fetch of %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	host := strings.TrimPrefix(ts.URL, "https://")
	p := &httpprobe.Probe{Client: ts.Client()}
	findings, err := p.Run(context.Background(), models.Target{Domain: host}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(findings) != 1 || findings[0].ProbeID != "http.robots_blocked" {
		t.Errorf("expected single robots_blocked finding, got %+v", findings)
	}
}
