package api_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/MWest2020/wanderer/internal/agent"
	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
)

// TestOutageAndReplay pins the "Outage and replay" scenario in
// specs/scanner/spec.md: three batches spooled while the core was
// down are drained once the core returns, and the drain runs a
// second time over the same logical batches (a restarted agent that
// re-spools before delivery is confirmed, or a second drain tick that
// still finds them). Each batch's findings must appear exactly once.
func TestOutageAndReplay(t *testing.T) {
	st, err := store.Open(context.Background(), "file::memory:?cache=shared&"+t.Name())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	secret := []byte("agent-secret-fixture")
	secrets := api.NewStaticAgentSecrets(map[string][]byte{"webapp-01": secret})
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	srv := httptest.NewServer(api.RouterWithSecrets(st, sc, sc.Logger, secrets))
	t.Cleanup(srv.Close)

	scanID := seedScan(t, st)

	type batch struct {
		id   string
		body []byte
	}
	batches := make([]batch, 3)
	for i := range batches {
		batches[i] = batch{
			id: fmt.Sprintf("batch-%d", i),
			body: []byte(fmt.Sprintf(
				`{"findings":[{"probe_id":"inventory.systemd.service","subject":"svc-%d","severity":"info","attributes":{}}]}`, i)),
		}
	}

	r := &agent.Remote{BaseURL: srv.URL, Secret: secret, Hostname: "webapp-01"}
	send := func(scanID, batchID string, body []byte) error {
		return r.SendBytes(context.Background(), scanID, batchID, body)
	}

	spoolAll := func(dir string) *agent.Outbox {
		ob := &agent.Outbox{Dir: dir, MaxBytes: 1 << 20}
		for _, b := range batches {
			if err := ob.Spool(scanID, b.id, b.body); err != nil {
				t.Fatalf("spool %s: %v", b.id, err)
			}
		}
		return ob
	}

	// First drain: the outage is over, all three batches deliver.
	first := spoolAll(t.TempDir())
	if err := first.Drain(send); err != nil {
		t.Fatalf("first drain: %v", err)
	}

	// Second drain over the same three (scanID, batchID, body)
	// triples — a restarted agent that re-spools before it learned
	// the first drain succeeded, or a second tick that still has them
	// queued. Every one of these is a replay of a batch the core
	// already has.
	second := spoolAll(t.TempDir())
	if err := second.Drain(send); err != nil {
		t.Fatalf("second drain: %v", err)
	}

	scan, err := st.GetScan(context.Background(), scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if len(scan.Findings) != len(batches) {
		t.Fatalf("findings after two drains = %d, want %d (one per batch, no duplicates)", len(scan.Findings), len(batches))
	}
}
