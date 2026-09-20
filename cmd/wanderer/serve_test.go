package main

import (
	"bytes"
	"context"
	"log/slog"
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
