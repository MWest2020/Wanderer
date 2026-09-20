package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsureSecret returns the HMAC signing key a remote-mode agent uses
// to talk to core, reading it from secretPath when that file already
// exists. When it does not, EnsureSecret exchanges token for a
// freshly issued secret via `POST {coreURL}/agents/enrol`, derives
// the signing key from it, and writes that key to secretPath (mode
// 0600) so the exchange never repeats on later starts — an existing
// secretPath always wins over token, enrolled or not.
//
// An empty token with no existing secretPath is an error: there is
// nothing to enrol with.
func EnsureSecret(ctx context.Context, httpClient *http.Client, coreURL, hostname, token, secretPath string) ([]byte, error) {
	if key, err := os.ReadFile(secretPath); err == nil {
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("agent: read secret file: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("agent: no secret file at %s and no enrolment token configured", secretPath)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	secret, err := exchangeToken(ctx, httpClient, coreURL, hostname, token)
	if err != nil {
		return nil, err
	}
	key := signingKey(secret)
	if dir := filepath.Dir(secretPath); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("agent: create secret dir: %w", err)
		}
	}
	if err := os.WriteFile(secretPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("agent: write secret file: %w", err)
	}
	return key, nil
}

// signingKey derives the HMAC key from a freshly issued plain secret.
// Core never stores the plain secret — only hex(sha256(secret)) — so
// that digest is the actual shared key both sides sign and verify
// with.
func signingKey(plainSecret string) []byte {
	sum := sha256.Sum256([]byte(plainSecret))
	return []byte(hex.EncodeToString(sum[:]))
}

// exchangeToken posts token+hostname to the core's enrolment endpoint
// and returns the plain secret it issues. The token and secret are
// never logged: a caller that fails must report the error only.
func exchangeToken(ctx context.Context, httpClient *http.Client, coreURL, hostname, token string) (string, error) {
	body, err := json.Marshal(map[string]string{"token": token, "hostname": hostname})
	if err != nil {
		return "", fmt.Errorf("agent: marshal enrol request: %w", err)
	}
	url := strings.TrimRight(coreURL, "/") + "/agents/enrol"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("agent: build enrol request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("agent: enrol: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("agent: enrol: core returned %d", resp.StatusCode)
	}
	var out struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("agent: decode enrol response: %w", err)
	}
	if out.Secret == "" {
		return "", errors.New("agent: enrol response missing secret")
	}
	return out.Secret, nil
}
