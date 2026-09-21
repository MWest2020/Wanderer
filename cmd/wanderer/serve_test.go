package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
)

func TestWarnIfNoAgentsEnrolled_EmptyStoreLogsInactive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if err := warnIfNoAgentsEnrolled(context.Background(), st, logger); err != nil {
		t.Fatalf("warnIfNoAgentsEnrolled: %v", err)
	}
	if !strings.Contains(buf.String(), "agent.ingest.inactive") {
		t.Errorf("expected an inactive-ingest log line, got %q", buf.String())
	}
}

func TestWarnIfNoAgentsEnrolled_AllRevokedLogsInactive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	ctx := context.Background()
	tok := mustNewToken(t, st)
	if _, _, err := st.EnrolAgent(ctx, tok, "webapp-a"); err != nil {
		t.Fatalf("enrol: %v", err)
	}
	if err := st.RevokeAgent(ctx, "webapp-a"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if err := warnIfNoAgentsEnrolled(ctx, st, logger); err != nil {
		t.Fatalf("warnIfNoAgentsEnrolled: %v", err)
	}
	if !strings.Contains(buf.String(), "agent.ingest.inactive") {
		t.Errorf("expected an inactive-ingest log line when every agent is revoked, got %q", buf.String())
	}
}

func TestWarnIfNoAgentsEnrolled_ActiveAgentStaysQuiet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	ctx := context.Background()
	tok := mustNewToken(t, st)
	if _, _, err := st.EnrolAgent(ctx, tok, "webapp-a"); err != nil {
		t.Fatalf("enrol: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if err := warnIfNoAgentsEnrolled(ctx, st, logger); err != nil {
		t.Fatalf("warnIfNoAgentsEnrolled: %v", err)
	}
	if strings.Contains(buf.String(), "agent.ingest.inactive") {
		t.Errorf("did not expect an inactive-ingest log line with an active agent, got %q", buf.String())
	}
}

func TestMountDemo_EmptyTargetMountsNothingAndLogsInactive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	root := http.NewServeMux()
	if err := mountDemo(root, st, logger, ""); err != nil {
		t.Fatalf("mountDemo: %v", err)
	}
	if !strings.Contains(buf.String(), "demo.disabled") {
		t.Errorf("expected a demo.disabled log line, got %q", buf.String())
	}

	srv := httptest.NewServer(root)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when demo.target is empty", resp.StatusCode)
	}
}

func TestMountDemo_TargetMountsRouteAndLogsEnabled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	root := http.NewServeMux()
	if err := mountDemo(root, st, logger, "westerweel.work"); err != nil {
		t.Fatalf("mountDemo: %v", err)
	}
	if !strings.Contains(buf.String(), "demo.enabled") {
		t.Errorf("expected a demo.enabled log line, got %q", buf.String())
	}

	srv := httptest.NewServer(root)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 when demo.target is set", resp.StatusCode)
	}
}
