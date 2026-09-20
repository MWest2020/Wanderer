package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MWest2020/wanderer/internal/store"
)

func TestAgentEnrolCLI_TokenCreateIntrekkenListRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")

	// Seed a token and an agent directly through the store — the CLI
	// commands under test are list and intrekken, not the HTTP enrol
	// endpoint (covered in internal/api).
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	token, _, err := st.CreateEnrolmentToken(context.Background(), 3600000000000)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, _, err := st.EnrolAgent(context.Background(), token, "webapp-a"); err != nil {
		t.Fatalf("enrol webapp-a: %v", err)
	}
	if _, _, err := st.EnrolAgent(context.Background(), mustNewToken(t, st), "webapp-b"); err != nil {
		t.Fatalf("enrol webapp-b: %v", err)
	}
	st.Close()

	if rc := runAgentIntrekken([]string{"--db", dbPath, "webapp-a"}); rc != 0 {
		t.Fatalf("runAgentIntrekken exit = %d, want 0", rc)
	}

	st2, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("re-open store: %v", err)
	}
	defer st2.Close()

	gotA, err := st2.GetAgentByHostname(context.Background(), "webapp-a")
	if err != nil {
		t.Fatalf("GetAgentByHostname(webapp-a): %v", err)
	}
	if !gotA.Revoked() {
		t.Error("webapp-a should be revoked after runAgentIntrekken")
	}
	gotB, err := st2.GetAgentByHostname(context.Background(), "webapp-b")
	if err != nil {
		t.Fatalf("GetAgentByHostname(webapp-b): %v", err)
	}
	if gotB.Revoked() {
		t.Error("webapp-b must be unaffected by revoking webapp-a")
	}

	if rc := runAgentList([]string{"--db", dbPath}); rc != 0 {
		t.Fatalf("runAgentList exit = %d, want 0", rc)
	}
}

func TestRunAgentIntrekken_UnknownHostnameFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	st.Close()

	if rc := runAgentIntrekken([]string{"--db", dbPath, "nope"}); rc == 0 {
		t.Fatal("expected non-zero exit for unknown hostname")
	}
}

func TestRunAgentTokenNieuw_PrintsTokenOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wanderer.db")
	if rc := runAgentTokenNieuw([]string{"--db", dbPath, "--ttl", "1h"}); rc != 0 {
		t.Fatalf("runAgentTokenNieuw exit = %d, want 0", rc)
	}

	st, err := store.Open(context.Background(), "file:"+dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	var count int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM enrolment_tokens`).Scan(&count); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if count != 1 {
		t.Errorf("enrolment_tokens rows = %d, want 1", count)
	}
}

func TestRunAgentToken_UnknownVerbFails(t *testing.T) {
	if rc := runAgentToken([]string{"bestaat-niet"}); rc != 2 {
		t.Errorf("runAgentToken unknown verb exit = %d, want 2", rc)
	}
}

// mustNewToken creates a fresh enrolment token directly through the
// store and returns its plain value, failing the test on error.
func mustNewToken(t *testing.T, st *store.Store) string {
	t.Helper()
	token, _, err := st.CreateEnrolmentToken(context.Background(), 3600000000000)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}
