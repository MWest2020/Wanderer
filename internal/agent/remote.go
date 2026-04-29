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

// MarshalBatch is the canonical JSON encoding of a batch the core
// expects on `POST /scans/{id}/findings`. Exported so the outbox can
// store the same bytes the live POST would have sent.
func MarshalBatch(findings []models.Finding) ([]byte, error) {
	body, err := json.Marshal(map[string]any{"findings": findings})
	if err != nil {
		return nil, fmt.Errorf("agent: marshal findings: %w", err)
	}
	return body, nil
}

// Send POSTs findings to the configured core, signed with the
// shared secret. The core's response status determines success.
func (r *Remote) Send(ctx context.Context, scanID string, findings []models.Finding) error {
	body, err := MarshalBatch(findings)
	if err != nil {
		return err
	}
	return r.SendBytes(ctx, scanID, body)
}

// SendBytes POSTs an already-marshalled batch body. Used by the
// outbox drain to retry a spooled batch without re-marshalling
// (preserving the exact bytes signed by the original attempt is not
// necessary because HMAC is timestamp-based and re-signs every send).
func (r *Remote) SendBytes(ctx context.Context, scanID string, body []byte) error {
	if r.HTTP == nil {
		r.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
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
