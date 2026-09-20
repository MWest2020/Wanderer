package api_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
)

func enrolTestServer(t *testing.T) (string, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&"+t.Name())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	srv := httptest.NewServer(api.Router(st, sc, sc.Logger))
	t.Cleanup(srv.Close)
	return srv.URL, st
}

func TestEnrolAgent_HappyPath(t *testing.T) {
	srv, st := enrolTestServer(t)
	plainToken, _, err := st.CreateEnrolmentToken(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	resp, err := http.Post(srv+"/agents/enrol", "application/json",
		strings.NewReader(`{"token":"`+plainToken+`","hostname":"webapp-01"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		Hostname string `json:"hostname"`
		Secret   string `json:"secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Hostname != "webapp-01" {
		t.Errorf("hostname = %q, want webapp-01", body.Hostname)
	}
	if body.Secret == "" {
		t.Error("empty secret in response")
	}

	got, err := st.GetAgentByHostname(context.Background(), "webapp-01")
	if err != nil {
		t.Fatalf("GetAgentByHostname: %v", err)
	}
	if got.SecretHash == body.Secret {
		t.Error("stored secret_hash equals the plain secret returned to the caller")
	}
}

// TestEnrolAgent_RefusalsLookIdentical exercises the HTTP surface of
// the "no difference in the error" requirement: an unknown token and
// an already-used token both come back as the same 401 shape.
func TestEnrolAgent_RefusalsLookIdentical(t *testing.T) {
	srv, st := enrolTestServer(t)
	usedToken, _, err := st.CreateEnrolmentToken(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	first, err := http.Post(srv+"/agents/enrol", "application/json",
		strings.NewReader(`{"token":"`+usedToken+`","hostname":"webapp-01"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first exchange status = %d, want 201", first.StatusCode)
	}

	cases := map[string]string{
		"unknown token":      "never-issued",
		"already-used token": usedToken,
	}
	var bodies []string
	for name, tok := range cases {
		resp, err := http.Post(srv+"/agents/enrol", "application/json",
			strings.NewReader(`{"token":"`+tok+`","hostname":"webapp-02"}`))
		if err != nil {
			t.Fatalf("%s: post: %v", name, err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, resp.StatusCode)
		}
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		resp.Body.Close()
		b, _ := json.Marshal(raw)
		bodies = append(bodies, string(b))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("refusal bodies differ: %q vs %q", bodies[0], bodies[1])
	}
}

func TestEnrolAgent_MissingFields(t *testing.T) {
	srv, _ := enrolTestServer(t)
	resp, err := http.Post(srv+"/agents/enrol", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
