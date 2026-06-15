package docker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// startUnixSocketServer binds an httptest.Server to a unix socket
// inside t.TempDir() and returns the socket path plus a recorder of
// every request method+path the server saw. Cleanup unbinds the
// socket on test exit.
func startUnixSocketServer(t *testing.T, handler http.Handler) (string, *requestLog) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	log := &requestLog{}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.record(r.Method, r.URL.Path)
		handler.ServeHTTP(w, r)
	}))
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return socket, log
}

type requestLog struct {
	mu      sync.Mutex
	entries []string
}

func (r *requestLog) record(method, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, method+" "+path)
}

func (r *requestLog) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.entries...)
}

// fixtureHandler routes the two endpoints we use.
func fixtureHandler(containers, images string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(containers))
		case strings.HasSuffix(r.URL.Path, "/images/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(images))
		default:
			http.NotFound(w, r)
		}
	}
}

func TestDocker_Inspect_HappyPath(t *testing.T) {
	containers := `[{
		"Id":"abcdef0123456789",
		"Names":["/web-app"],
		"Image":"nginx:1.27",
		"ImageID":"sha256:1111",
		"Created":1700000000,
		"State":"running",
		"Status":"Up 2 minutes",
		"Labels":{"env":"prod"}
	}]`
	images := `[{
		"Id":"sha256:1111",
		"RepoTags":["nginx:1.27","nginx:latest"],
		"Created":1699000000,
		"Size":142857
	}]`
	socket, log := startUnixSocketServer(t, fixtureHandler(containers, images))

	d := Docker{Socket: socket}
	if ok, reason := d.Available(); !ok {
		t.Fatalf("Available = false (%s)", reason)
	}
	got, err := d.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2", len(got))
	}
	if got[0].ProbeID != "inventory.docker.container" || got[0].Subject != "web-app" {
		t.Errorf("container finding wrong: %+v", got[0])
	}
	if got[0].Attributes["image"] != "nginx:1.27" {
		t.Errorf("container image attr = %v", got[0].Attributes["image"])
	}
	if got[0].Attributes["state"] != "running" {
		t.Errorf("container state attr = %v", got[0].Attributes["state"])
	}
	if got[1].ProbeID != "inventory.docker.image" || got[1].Subject != "nginx:1.27" {
		t.Errorf("image finding wrong: %+v", got[1])
	}
	if got[1].Attributes["digest"] != "sha256:1111" {
		t.Errorf("image digest attr = %v", got[1].Attributes["digest"])
	}

	// Read-only contract: every call seen by the server is a GET, no
	// mutating paths anywhere.
	for _, e := range log.snapshot() {
		if !strings.HasPrefix(e, "GET ") {
			t.Errorf("non-GET request: %s", e)
		}
		for _, banned := range []string{"/exec", "/wait", "/start", "/stop", "/kill", "/pause", "/unpause"} {
			if strings.Contains(e, banned) {
				t.Errorf("forbidden path: %s", e)
			}
		}
	}
}

func TestDocker_Inspect_UnnamedContainerUsesShortID(t *testing.T) {
	containers := `[{
		"Id":"abcdef0123456789aaaaaaaaaaaa",
		"Names":[],
		"Image":"alpine",
		"ImageID":"sha256:abc",
		"Created":1700000000,
		"State":"exited",
		"Status":"Exited"
	}]`
	socket, _ := startUnixSocketServer(t, fixtureHandler(containers, "[]"))

	d := Docker{Socket: socket}
	got, err := d.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got[0].Subject != "abcdef012345" {
		t.Errorf("Subject = %q, want short id", got[0].Subject)
	}
}

func TestDocker_Inspect_500ReturnsAPIError(t *testing.T) {
	socket, _ := startUnixSocketServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))

	d := Docker{Socket: socket}
	_, err := d.Inspect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAPIError(err) {
		t.Errorf("err = %v; want apiError", err)
	}
	if StatusCode(err) != http.StatusInternalServerError {
		t.Errorf("status = %d", StatusCode(err))
	}
}

func TestDocker_Inspect_404ReturnsAPIError(t *testing.T) {
	socket, _ := startUnixSocketServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	d := Docker{Socket: socket}
	_, err := d.Inspect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if StatusCode(err) != http.StatusNotFound {
		t.Errorf("status = %d", StatusCode(err))
	}
}

func TestDocker_Inspect_BadJSONReturnsDecodeError(t *testing.T) {
	socket, _ := startUnixSocketServer(t, fixtureHandler("not json", "[]"))

	d := Docker{Socket: socket}
	_, err := d.Inspect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if IsAPIError(err) {
		t.Errorf("decode error should not be tagged as apiError")
	}
}

func TestDocker_Available_NoSocket(t *testing.T) {
	d := Docker{Socket: ""}
	ok, reason := d.Available()
	if ok {
		t.Fatal("Available true for empty socket")
	}
	if reason == "" {
		t.Error("Available reason empty")
	}
}

func TestDocker_Available_MissingFile(t *testing.T) {
	d := Docker{Socket: "/nonexistent/docker.sock"}
	ok, reason := d.Available()
	if ok {
		t.Fatal("Available true for missing socket")
	}
	if !strings.Contains(reason, "not present") {
		t.Errorf("reason = %q, want 'not present'", reason)
	}
}

func TestStatusCode_NonAPIErrorReturnsZero(t *testing.T) {
	if got := StatusCode(errors.New("plain")); got != 0 {
		t.Errorf("StatusCode of plain error = %d, want 0", got)
	}
}
