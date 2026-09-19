package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	httpprobe "github.com/MWest2020/wanderer/internal/probe/http"
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

func findingByProbeID(findings []models.Finding, id string) *models.Finding {
	for i := range findings {
		if findings[i].ProbeID == id {
			return &findings[i]
		}
	}
	return nil
}

func TestSecurityTxtValid(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			http.NotFound(w, r)
		case "/.well-known/security.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Contact: mailto:security@voorbeeld.nl\nExpires: 2027-01-01T00:00:00.000Z\n"))
		default:
			_, _ = w.Write([]byte("<!doctype html><html><body>hi</body></html>"))
		}
	}))
	defer ts.Close()
	host := strings.TrimPrefix(ts.URL, "https://")
	p := &httpprobe.Probe{Client: ts.Client()}
	findings, err := p.Run(context.Background(), models.Target{Domain: host}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.securitytxt")
	if f == nil {
		t.Fatalf("no http.securitytxt finding, got %+v", findings)
	}
	if f.Attributes["present"] != true {
		t.Errorf("present = %v, want true", f.Attributes["present"])
	}
	if f.Attributes["parseable"] != true {
		t.Errorf("parseable = %v, want true", f.Attributes["parseable"])
	}
	contact, _ := f.Attributes["contact"].([]string)
	if len(contact) != 1 || contact[0] != "mailto:security@voorbeeld.nl" {
		t.Errorf("contact = %v, want [mailto:security@voorbeeld.nl]", f.Attributes["contact"])
	}
	if f.Attributes["expires"] != "2027-01-01T00:00:00.000Z" {
		t.Errorf("expires = %v", f.Attributes["expires"])
	}
}

func TestSecurityTxtNotFound(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			http.NotFound(w, r)
		case "/.well-known/security.txt":
			http.NotFound(w, r)
		default:
			_, _ = w.Write([]byte("<!doctype html><html><body>hi</body></html>"))
		}
	}))
	defer ts.Close()
	host := strings.TrimPrefix(ts.URL, "https://")
	p := &httpprobe.Probe{Client: ts.Client()}
	findings, err := p.Run(context.Background(), models.Target{Domain: host}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.securitytxt")
	if f == nil {
		t.Fatalf("no http.securitytxt finding, got %+v", findings)
	}
	if f.Attributes["present"] != false {
		t.Errorf("present = %v, want false (404 is a valid observation)", f.Attributes["present"])
	}
	if f.Attributes["parseable"] != false {
		t.Errorf("parseable = %v, want false", f.Attributes["parseable"])
	}
	if f.Attributes["status"] != 404 {
		t.Errorf("status = %v, want 404", f.Attributes["status"])
	}
	if findingByProbeID(findings, "http.securitytxt.unavailable") != nil {
		t.Errorf("404 must not emit http.securitytxt.unavailable")
	}
}

func TestSecurityTxtHTMLAtPath(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			http.NotFound(w, r)
		case "/.well-known/security.txt":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><html><body>Not Found</body></html>"))
		default:
			_, _ = w.Write([]byte("<!doctype html><html><body>hi</body></html>"))
		}
	}))
	defer ts.Close()
	host := strings.TrimPrefix(ts.URL, "https://")
	p := &httpprobe.Probe{Client: ts.Client()}
	findings, err := p.Run(context.Background(), models.Target{Domain: host}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.securitytxt")
	if f == nil {
		t.Fatalf("no http.securitytxt finding, got %+v", findings)
	}
	if f.Attributes["present"] != true {
		t.Errorf("present = %v, want true (200 is present even if unparseable)", f.Attributes["present"])
	}
	if f.Attributes["parseable"] != false {
		t.Errorf("parseable = %v, want false", f.Attributes["parseable"])
	}
}

// errTransport always fails, standing in for a transport-level failure
// (DNS, TLS, connection refused) distinct from a valid 4xx/5xx
// response.
type errTransport struct{}

func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")}
}

func TestSecurityTxtTransportFailure(t *testing.T) {
	p := &httpprobe.Probe{Client: &http.Client{Transport: errTransport{}}}
	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.securitytxt.unavailable")
	if f == nil {
		t.Fatalf("no http.securitytxt.unavailable finding, got %+v", findings)
	}
	if findingByProbeID(findings, "http.securitytxt") != nil {
		t.Errorf("transport failure must not emit http.securitytxt")
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
