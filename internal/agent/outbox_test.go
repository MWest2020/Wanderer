package agent

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newTestOutbox(t *testing.T) *Outbox {
	t.Helper()
	return &Outbox{Dir: t.TempDir(), MaxBytes: 1 << 20}
}

func TestOutbox_EnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "outbox")
	o := &Outbox{Dir: dir}
	if err := o.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
}

func TestOutbox_SpoolDrainRoundTrip(t *testing.T) {
	o := newTestOutbox(t)
	body := []byte(`{"findings":[{"id":"f_1","probe_id":"dns.mx"}]}`)
	if err := o.Spool("t_abc", body); err != nil {
		t.Fatalf("spool: %v", err)
	}

	var captured []struct {
		scanID string
		body   []byte
	}
	send := func(scanID string, b []byte) error {
		captured = append(captured, struct {
			scanID string
			body   []byte
		}{scanID, append([]byte(nil), b...)})
		return nil
	}
	if err := o.Drain(send); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(captured) != 1 {
		t.Fatalf("captured = %d, want 1", len(captured))
	}
	if captured[0].scanID != "t_abc" {
		t.Errorf("scanID = %s", captured[0].scanID)
	}
	if string(captured[0].body) != string(body) {
		t.Errorf("body lost in round-trip")
	}
	// The file is gone after a successful drain.
	entries, _ := os.ReadDir(o.Dir)
	if len(entries) != 0 {
		t.Errorf("dir not empty after drain: %v", entries)
	}
}

func TestOutbox_DrainStopsOnFailure(t *testing.T) {
	o := newTestOutbox(t)
	if err := o.Spool("t_one", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := o.Spool("t_two", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	var calls int32
	stub := func(scanID string, body []byte) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("network down")
	}
	if err := o.Drain(stub); err == nil {
		t.Fatal("drain should propagate send error")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (drain stops on first failure)", calls)
	}
	// Both files remain.
	entries, _ := os.ReadDir(o.Dir)
	if len(entries) != 2 {
		t.Errorf("expected 2 files after failed drain, got %d", len(entries))
	}
}

func TestOutbox_PrunesOnSpool(t *testing.T) {
	dir := t.TempDir()
	// Tiny cap so two ~150-byte writes already trigger a prune.
	o := &Outbox{Dir: dir, MaxBytes: 200}

	// Force monotonically-increasing timestamps so older files sort
	// first.
	t0 := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	o.Now = func() time.Time { t0 = t0.Add(time.Second); return t0 }

	// Body must be valid JSON because the outbox wraps it as
	// json.RawMessage; pad it out with a string that brings each file
	// over ~120 bytes after the envelope.
	body := []byte(`{"pad":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`)
	for i := 0; i < 5; i++ {
		if err := o.Spool("t_x", body); err != nil {
			t.Fatalf("spool %d: %v", i, err)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) >= 5 {
		t.Errorf("prune did not run: %d files", len(entries))
	}
	// Total size at or below MaxBytes.
	var total int64
	for _, e := range entries {
		info, _ := e.Info()
		total += info.Size()
	}
	if total > o.MaxBytes {
		t.Errorf("total size %d exceeds cap %d", total, o.MaxBytes)
	}
}

func TestOutbox_DrainSkipsCorrupt(t *testing.T) {
	o := newTestOutbox(t)
	bad := filepath.Join(o.Dir, "20260427T120000Z_corrupt.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := o.Spool("t_good", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := o.Drain(func(scanID string, body []byte) error {
		called = true
		if scanID != "t_good" {
			t.Errorf("drain called with corrupt file: %s", scanID)
		}
		return nil
	}); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if !called {
		t.Error("drain skipped the good file too")
	}
	if _, err := os.Stat(bad + ".corrupt"); err != nil {
		t.Errorf("corrupt file not renamed: %v", err)
	}
}
