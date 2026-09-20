package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
)

func newAgentTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(memory)&"+t.Name())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestEnrolAgent_HappyPath(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()

	plainToken, tok, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if plainToken == "" {
		t.Fatal("empty plain token")
	}
	if tok.TokenHash == plainToken {
		t.Error("token hash equals the plain token — the plain value must never be stored verbatim")
	}

	secret, ag, err := st.EnrolAgent(ctx, plainToken, "webapp-01")
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	if secret == "" {
		t.Fatal("empty secret")
	}
	if ag.SecretHash == secret {
		t.Error("secret hash equals the plain secret — the plain value must never be stored verbatim")
	}
	if ag.Hostname != "webapp-01" {
		t.Errorf("hostname = %q, want webapp-01", ag.Hostname)
	}
	if ag.EnrolledAt.IsZero() {
		t.Error("EnrolledAt not set")
	}
	if ag.Revoked() {
		t.Error("freshly enrolled agent must not be revoked")
	}
}

// TestEnrolAgent_FailureModesAreIndistinguishable pins the spec
// requirement: an unknown token, an expired token, and an
// already-used token all fail with the exact same error, so a
// caller — and therefore an attacker probing enrolment — cannot
// learn which of the three happened.
func TestEnrolAgent_FailureModesAreIndistinguishable(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()

	usedToken, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.EnrolAgent(ctx, usedToken, "webapp-used"); err != nil {
		t.Fatal(err)
	}

	expiredToken, expTok, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`UPDATE enrolment_tokens SET expires_at = ? WHERE token_hash = ?`,
		time.Now().UTC().Add(-time.Minute), expTok.TokenHash); err != nil {
		t.Fatalf("force-expire: %v", err)
	}

	cases := []struct {
		name  string
		token string
	}{
		{"unknown token", "never-issued-token"},
		{"expired token", expiredToken},
		{"already-used token", usedToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			secret, ag, err := st.EnrolAgent(ctx, tc.token, "webapp-x")
			if !errors.Is(err, store.ErrEnrolmentFailed) {
				t.Errorf("got err %v, want ErrEnrolmentFailed", err)
			}
			if secret != "" || ag != nil {
				t.Error("a refused exchange must not return a secret or an agent")
			}
		})
	}

	// None of the refused exchanges enrolled webapp-x.
	if _, err := st.GetAgentByHostname(ctx, "webapp-x"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("webapp-x should not exist, got %v", err)
	}
}

func TestEnrolAgent_TokenUsedTwice_SecondEnrolmentDoesNotHappen(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()

	plainToken, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, _, err := st.EnrolAgent(ctx, plainToken, "webapp-01"); err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	if _, _, err := st.EnrolAgent(ctx, plainToken, "webapp-02"); !errors.Is(err, store.ErrEnrolmentFailed) {
		t.Errorf("second exchange: got %v, want ErrEnrolmentFailed", err)
	}
	if _, err := st.GetAgentByHostname(ctx, "webapp-02"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("webapp-02 should not have been enrolled, got %v", err)
	}
}

func TestRevokeAgent_RefusesThatAgentLeavesOtherAlone(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()

	tokenA, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.EnrolAgent(ctx, tokenA, "webapp-a"); err != nil {
		t.Fatal(err)
	}
	tokenB, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.EnrolAgent(ctx, tokenB, "webapp-b"); err != nil {
		t.Fatal(err)
	}

	if err := st.RevokeAgent(ctx, "webapp-a"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	gotA, err := st.GetAgentByHostname(ctx, "webapp-a")
	if err != nil {
		t.Fatal(err)
	}
	if !gotA.Revoked() {
		t.Error("webapp-a should be revoked")
	}

	gotB, err := st.GetAgentByHostname(ctx, "webapp-b")
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Revoked() {
		t.Error("webapp-b must be unaffected by revoking webapp-a")
	}
}

func TestRevokeAgent_UnknownHostnameReturnsErrNotFound(t *testing.T) {
	st := newAgentTestStore(t)
	err := st.RevokeAgent(context.Background(), "nope")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestRevokeAgent_IdempotentKeepsOriginalTimestamp(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()

	token, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.EnrolAgent(ctx, token, "webapp-a"); err != nil {
		t.Fatal(err)
	}
	if err := st.RevokeAgent(ctx, "webapp-a"); err != nil {
		t.Fatal(err)
	}
	first, err := st.GetAgentByHostname(ctx, "webapp-a")
	if err != nil {
		t.Fatal(err)
	}

	if err := st.RevokeAgent(ctx, "webapp-a"); err != nil {
		t.Fatalf("second revoke: %v", err)
	}
	second, err := st.GetAgentByHostname(ctx, "webapp-a")
	if err != nil {
		t.Fatal(err)
	}
	if first.RevokedAt == nil || second.RevokedAt == nil || !first.RevokedAt.Equal(*second.RevokedAt) {
		t.Errorf("revoked_at changed on idempotent revoke: %v vs %v", first.RevokedAt, second.RevokedAt)
	}
}

func TestListAgents_OrderedByHostname(t *testing.T) {
	st := newAgentTestStore(t)
	ctx := context.Background()
	for _, host := range []string{"zeta", "alpha", "mu"} {
		token, _, err := st.CreateEnrolmentToken(ctx, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := st.EnrolAgent(ctx, token, host); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.ListAgents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 agents, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Hostname > got[i].Hostname {
			t.Errorf("not sorted: %s before %s", got[i-1].Hostname, got[i].Hostname)
		}
	}
}
