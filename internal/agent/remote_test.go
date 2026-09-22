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
	var gotBatchID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.hostname = r.Header.Get(HeaderHostname)
		got.timestamp = r.Header.Get(HeaderTimestamp)
		got.signature = r.Header.Get(HeaderSignature)
		gotBatchID = r.Header.Get(HeaderBatchID)
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
	if err := r.Send(context.Background(), "s_abc", "batch-1", findings); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got.hostname != "webapp-01" {
		t.Errorf("hostname header = %s", got.hostname)
	}
	if gotBatchID != "batch-1" {
		t.Errorf("batch id header = %s, want batch-1", gotBatchID)
	}
	if err := Verify(secret, got.body, got.timestamp, got.signature, time.Now().UTC()); err != nil {
		t.Errorf("server-side verify failed: %v", err)
	}
}

func TestNewBatchID_Unique(t *testing.T) {
	a, err := NewBatchID()
	if err != nil {
		t.Fatalf("NewBatchID: %v", err)
	}
	b, err := NewBatchID()
	if err != nil {
		t.Fatalf("NewBatchID: %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("NewBatchID returned an empty id")
	}
	if a == b {
		t.Errorf("two calls to NewBatchID returned the same id: %s", a)
	}
}

func TestRemote_NonSuccessReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	r := &Remote{BaseURL: srv.URL, Secret: []byte("s"), Hostname: "h"}
	err := r.Send(context.Background(), "s", "batch-1", []models.Finding{{ProbeID: "x", Subject: "y", Severity: models.SeverityInfo, Attributes: map[string]any{}}})
	if err == nil {
		t.Errorf("expected error on non-2xx")
	}
}
