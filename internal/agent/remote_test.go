package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestRemote_SendSignsCorrectly(t *testing.T) {
	secret := []byte("agent-secret")
	var got struct {
		hostname  string
		timestamp string
		signature string
		body      []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.hostname = r.Header.Get(HeaderHostname)
		got.timestamp = r.Header.Get(HeaderTimestamp)
		got.signature = r.Header.Get(HeaderSignature)
		got.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	r := &Remote{
		BaseURL:  srv.URL,
		Secret:   secret,
		Hostname: "webapp-01",
	}
	findings := []models.Finding{
		{ProbeID: "inventory.systemd.service", Subject: "sshd.service", Severity: models.SeverityInfo, Attributes: map[string]any{}},
	}
	if err := r.Send(context.Background(), "s_abc", findings); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got.hostname != "webapp-01" {
		t.Errorf("hostname header = %s", got.hostname)
	}
	if err := Verify(secret, got.body, got.timestamp, got.signature, time.Now().UTC()); err != nil {
		t.Errorf("server-side verify failed: %v", err)
	}
}

func TestRemote_NonSuccessReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	r := &Remote{BaseURL: srv.URL, Secret: []byte("s"), Hostname: "h"}
	err := r.Send(context.Background(), "s", []models.Finding{{ProbeID: "x", Subject: "y", Severity: models.SeverityInfo, Attributes: map[string]any{}}})
	if err == nil {
		t.Errorf("expected error on non-2xx")
	}
}
