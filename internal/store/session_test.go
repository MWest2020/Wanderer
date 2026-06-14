package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
)

func newSessionStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestSessionRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newSessionStore(t)

	sess := &store.Session{
		ID:           "cookie-1",
		Subject:      "alice",
		Email:        "alice@example.nl",
		AccessToken:  "at",
		RefreshToken: "rt",
		TokenExpiry:  time.Now().UTC().Add(time.Hour),
		ExpiresAt:    time.Now().UTC().Add(12 * time.Hour),
	}
	if err := st.CreateSession(ctx, sess); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := st.GetSession(ctx, "cookie-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Subject != "alice" || got.RefreshToken != "rt" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.LastValidatedAt.IsZero() {
		t.Error("LastValidatedAt should be stamped on create")
	}

	// RefreshSession rotates the token set and bumps LastValidatedAt.
	got.AccessToken = "at2"
	got.RefreshToken = "rt2"
	prevValidated := got.LastValidatedAt
	time.Sleep(2 * time.Millisecond)
	if err := st.RefreshSession(ctx, got); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	after, _ := st.GetSession(ctx, "cookie-1")
	if after.AccessToken != "at2" || after.RefreshToken != "rt2" {
		t.Fatalf("refresh did not persist tokens: %+v", after)
	}
	if !after.LastValidatedAt.After(prevValidated) {
		t.Error("RefreshSession should advance LastValidatedAt")
	}

	if err := st.DeleteSession(ctx, "cookie-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := st.GetSession(ctx, "cookie-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("deleted session should be ErrNotFound, got %v", err)
	}
}

func TestSessionExpiryTreatedAsAbsent(t *testing.T) {
	ctx := context.Background()
	st := newSessionStore(t)

	sess := &store.Session{
		ID:        "stale",
		Subject:   "bob",
		ExpiresAt: time.Now().UTC().Add(-time.Minute), // already past
	}
	if err := st.CreateSession(ctx, sess); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.GetSession(ctx, "stale"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired session must read as ErrNotFound, got %v", err)
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	ctx := context.Background()
	st := newSessionStore(t)

	_ = st.CreateSession(ctx, &store.Session{ID: "live", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	_ = st.CreateSession(ctx, &store.Session{ID: "dead", ExpiresAt: time.Now().UTC().Add(-time.Hour)})

	n, err := st.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 purged session, got %d", n)
	}
	if _, err := st.GetSession(ctx, "live"); err != nil {
		t.Fatalf("live session should survive purge: %v", err)
	}
}
