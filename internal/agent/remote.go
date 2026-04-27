package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Remote ships Findings to a Wanderer core over HMAC-signed HTTPS.
// It owns no scheduling — call Send for each batch.
type Remote struct {
	BaseURL  string
	Secret   []byte
	Hostname string
	HTTP     *http.Client

	// Now lets tests freeze the clock used for the timestamp header.
	// Defaults to time.Now.
	Now func() time.Time
}

// Send POSTs findings to the configured core, signed with the
// shared secret. The core's response status determines success.
func (r *Remote) Send(ctx context.Context, scanID string, findings []models.Finding) error {
	if r.HTTP == nil {
		r.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	body, err := json.Marshal(map[string]any{"findings": findings})
	if err != nil {
		return fmt.Errorf("agent: marshal findings: %w", err)
	}
	url := strings.TrimRight(r.BaseURL, "/") + "/scans/" + scanID + "/findings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("agent: build request: %w", err)
	}
	ts, sig := Sign(r.Secret, body, now())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderHostname, r.Hostname)
	req.Header.Set(HeaderTimestamp, ts)
	req.Header.Set(HeaderSignature, sig)

	resp, err := r.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("agent: post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("agent: core returned %d", resp.StatusCode)
}
