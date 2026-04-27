package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HeaderHostname is set by the agent to identify itself. The core
// uses it to look up the shared secret.
const HeaderHostname = "X-Wanderer-Agent"

// HeaderTimestamp is the RFC 3339 UTC time the agent computed the
// signature. Used for replay protection.
const HeaderTimestamp = "X-Wanderer-Timestamp"

// HeaderSignature is the base64-encoded HMAC-SHA256 over the bytes
// `<timestamp> + "\n" + <body>`.
const HeaderSignature = "X-Wanderer-Signature"

// MaxClockSkew bounds how far apart agent and core clocks may drift
// before a signed request is rejected.
const MaxClockSkew = 5 * time.Minute

// Sign returns the (timestamp, signature) header pair the agent
// SHOULD attach to a request. The body is the raw JSON about to be
// posted; the timestamp is now (UTC, second precision).
func Sign(secret, body []byte, now time.Time) (timestamp, signature string) {
	timestamp = now.UTC().Format(time.RFC3339)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	return timestamp, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Verify checks a signed request. ErrBadSignature, ErrSkew, or
// ErrUnknownHost are returned for the three failure shapes; callers
// translate to 401 / 401 / 401 respectively (we return 401 for all
// auth failures, so callers must not leak which one happened).
func Verify(secret, body []byte, timestamp, signature string, now time.Time) error {
	if secret == nil {
		return ErrUnknownHost
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ErrSkew
	}
	delta := now.Sub(t)
	if delta < 0 {
		delta = -delta
	}
	if delta > MaxClockSkew {
		return fmt.Errorf("%w: timestamp %v is outside ±%v window", ErrSkew, t, MaxClockSkew)
	}
	got, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return ErrBadSignature
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	want := mac.Sum(nil)
	if !hmac.Equal(got, want) {
		return ErrBadSignature
	}
	return nil
}

// Sentinel errors the verifier returns. All represent authentication
// failure; callers should respond with HTTP 401 without leaking which
// branch fired (information-leak safety).
var (
	ErrBadSignature = errors.New("agent: bad signature")
	ErrSkew         = errors.New("agent: timestamp skew")
	ErrUnknownHost  = errors.New("agent: unknown agent hostname")
)
