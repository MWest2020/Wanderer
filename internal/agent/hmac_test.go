package agent

import (
	"errors"
	"testing"
	"time"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	secret := []byte("shared-secret-for-tests")
	body := []byte(`{"findings":[{"probe_id":"inventory.systemd.service"}]}`)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	ts, sig := Sign(secret, body, now)
	if err := Verify(secret, body, ts, sig, now); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	body := []byte(`{}`)
	now := time.Now().UTC()
	ts, sig := Sign([]byte("agent-secret"), body, now)
	err := Verify([]byte("server-thinks-this-is-the-secret"), body, ts, sig, now)
	if !errors.Is(err, ErrBadSignature) {
		t.Errorf("want ErrBadSignature, got %v", err)
	}
}

func TestVerify_TamperedBody(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"a":1}`)
	now := time.Now().UTC()
	ts, sig := Sign(secret, body, now)
	if err := Verify(secret, []byte(`{"a":2}`), ts, sig, now); !errors.Is(err, ErrBadSignature) {
		t.Errorf("want ErrBadSignature, got %v", err)
	}
}

func TestVerify_Replay(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{}`)
	signedAt := time.Now().UTC().Add(-10 * time.Minute)
	ts, sig := Sign(secret, body, signedAt)
	err := Verify(secret, body, ts, sig, time.Now().UTC())
	if !errors.Is(err, ErrSkew) {
		t.Errorf("want ErrSkew, got %v", err)
	}
}

func TestVerify_BadTimestamp(t *testing.T) {
	err := Verify([]byte("s"), []byte("{}"), "not-a-time", "AAAA", time.Now())
	if !errors.Is(err, ErrSkew) {
		t.Errorf("want ErrSkew, got %v", err)
	}
}

func TestVerify_UnknownHost(t *testing.T) {
	err := Verify(nil, []byte("{}"), time.Now().UTC().Format(time.RFC3339), "AAAA", time.Now())
	if !errors.Is(err, ErrUnknownHost) {
		t.Errorf("want ErrUnknownHost, got %v", err)
	}
}
