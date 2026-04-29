package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// DefaultOutboxMaxBytes caps the on-disk footprint of the agent's
// outbox spool. 100 MiB is large enough to absorb a multi-day outage
// for a typical inventory cadence and small enough not to surprise
// an operator who ssh's into the host.
const DefaultOutboxMaxBytes int64 = 100 << 20

// Outbox is a directory of failed-batch JSON files the agent re-tries
// on every tick. Files use a lexicographically-sortable filename so a
// directory listing is also a chronological order.
//
// On-disk format: one file per failed batch, each file is a JSON
// document of shape {"scan_id": "...", "body": "<base64 of POST body>"}.
// The body is base64-encoded so the JSON wrapper round-trips without
// double-escaping the original payload.
type Outbox struct {
	Dir      string
	MaxBytes int64

	// Now is a test seam for the timestamp prefix; defaults to
	// time.Now.UTC().
	Now func() time.Time
}

// SpooledBatch is the on-disk shape.
type SpooledBatch struct {
	ScanID string `json:"scan_id"`
	// Body is the raw POST body the agent would have sent. Stored as
	// json.RawMessage so it round-trips bit-for-bit; the core's HMAC
	// re-signing on retry uses the current timestamp so an attacker
	// cannot replay an old batch.
	Body json.RawMessage `json:"body"`
}

// EnsureDir creates the outbox directory if it does not exist and
// confirms it is writable. Call once at startup so a misconfigured
// path produces a hard error rather than a silent first-tick drop.
func (o *Outbox) EnsureDir() error {
	if o.Dir == "" {
		return errors.New("outbox: Dir is required")
	}
	if err := os.MkdirAll(o.Dir, 0o700); err != nil {
		return fmt.Errorf("outbox: mkdir %s: %w", o.Dir, err)
	}
	probe := filepath.Join(o.Dir, ".writable_probe")
	f, err := os.OpenFile(probe, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("outbox: %s not writable: %w", o.Dir, err)
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return nil
}

// Spool persists one batch. The filename is
// <RFC3339-basic UTC>_<random6>.json so lexicographic sort matches
// chronological order. After write, Spool calls Prune so a hard
// outage cannot grow the directory past MaxBytes.
func (o *Outbox) Spool(scanID string, body []byte) error {
	if scanID == "" {
		return errors.New("outbox: scanID required")
	}
	now := time.Now().UTC
	if o.Now != nil {
		now = func() time.Time { return o.Now().UTC() }
	}
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("outbox: rand: %w", err)
	}
	filename := fmt.Sprintf("%s_%s.json",
		now().Format("20060102T150405Z"),
		hex.EncodeToString(suffix),
	)
	path := filepath.Join(o.Dir, filename)

	envelope, err := json.Marshal(SpooledBatch{
		ScanID: scanID,
		Body:   json.RawMessage(body),
	})
	if err != nil {
		return fmt.Errorf("outbox: marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, envelope, 0o600); err != nil {
		return fmt.Errorf("outbox: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("outbox: rename: %w", err)
	}
	return o.Prune()
}

// Drain reads spooled batches in lexicographic (age) order and calls
// send for each. A successful send removes the file; a failure stops
// the drain so the next tick retries the same batch first. A
// malformed file is logged once and skipped.
func (o *Outbox) Drain(send func(scanID string, body []byte) error) error {
	files, err := o.listBatchFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		path := filepath.Join(o.Dir, f.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("outbox: read %s: %w", path, err)
		}
		var b SpooledBatch
		if err := json.Unmarshal(raw, &b); err != nil {
			// Move corrupt file aside so it does not block forever.
			bad := path + ".corrupt"
			_ = os.Rename(path, bad)
			continue
		}
		if err := send(b.ScanID, b.Body); err != nil {
			return fmt.Errorf("outbox: send %s: %w", f.Name(), err)
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("outbox: remove %s: %w", path, err)
		}
	}
	return nil
}

// Prune removes the oldest files until the directory total falls at
// or below MaxBytes. A zero MaxBytes uses DefaultOutboxMaxBytes.
func (o *Outbox) Prune() error {
	limit := o.MaxBytes
	if limit <= 0 {
		limit = DefaultOutboxMaxBytes
	}
	files, err := o.listBatchFiles()
	if err != nil {
		return err
	}
	var total int64
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	if total <= limit {
		return nil
	}
	// Files are sorted oldest-first; remove from the front until under
	// the cap.
	for _, f := range files {
		if total <= limit {
			break
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(o.Dir, f.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("outbox: prune %s: %w", path, err)
		}
		total -= info.Size()
	}
	return nil
}

func (o *Outbox) listBatchFiles() ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(o.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("outbox: list %s: %w", o.Dir, err)
	}
	var batches []fs.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		batches = append(batches, e)
	}
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].Name() < batches[j].Name()
	})
	return batches, nil
}
