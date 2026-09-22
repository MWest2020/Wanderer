package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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

// NewBatchID mints a random identifier for one batch of findings. The
// caller mints it once per batch and reuses the same value for every
// delivery attempt of that batch (live retries, and — via the outbox
// — a delivery after a restart), so the core can recognise a replay.
//
// A random ID rather than a hash of the body: two batches with
// identical findings (an inventory tick with no observed change, say)
// are still two distinct deliveries and must both be stored. A
// content hash would conflate them; a per-batch random value does
// not.
func NewBatchID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("agent: batch id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Send POSTs findings to the configured core, signed with the
// shared secret. The core's response status determines success.
func (r *Remote) Send(ctx context.Context, scanID, batchID string, findings []models.Finding) error {
	body, err := MarshalBatch(findings)
	if err != nil {
		return err
	}
	return r.SendBytes(ctx, scanID, batchID, body)
}

// SendBytes POSTs an already-marshalled batch body. Used by the
// outbox drain to retry a spooled batch without re-marshalling
// (preserving the exact bytes signed by the original attempt is not
// necessary because HMAC is timestamp-based and re-signs every send).
// batchID travels as a header, not part of body, so the core can
// deduplicate a replay of this exact batch.
func (r *Remote) SendBytes(ctx context.Context, scanID, batchID string, body []byte) error {
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
	req.Header.Set(HeaderBatchID, batchID)

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
