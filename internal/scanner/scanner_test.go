package scanner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubProbe struct {
	id       string
	findings []models.Finding
	err      error
	panic    bool
	sleep    time.Duration
}

func (s *stubProbe) ID() string { return s.id }
func (s *stubProbe) Run(ctx context.Context, _ models.Target, _ probe.Config) ([]models.Finding, error) {
	if s.panic {
		panic("stub panic")
	}
	if s.sleep > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.sleep):
		}
	}
	return s.findings, s.err
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func mkFinding(probeID string) models.Finding {
	return models.Finding{
		ProbeID:    probeID,
		Subject:    "example.nl",
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{},
	}
}

func TestScanAllProbesSucceed(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}},
		&stubProbe{id: "b", findings: []models.Finding{mkFinding("b.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusComplete {
		t.Errorf("status = %q, want complete", res.Status)
	}
	if len(res.Findings) != 2 {
		t.Errorf("findings = %d, want 2", len(res.Findings))
	}
}

func TestScanOneProbeFailsScanContinues(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}},
		&stubProbe{id: "b", err: errors.New("boom")},
		&stubProbe{id: "c", findings: []models.Finding{mkFinding("c.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusPartial {
		t.Errorf("status = %q, want partial", res.Status)
	}
	var sawA, sawC, sawBErr bool
	for _, f := range res.Findings {
		switch f.ProbeID {
		case "a.ok":
			sawA = true
		case "c.ok":
			sawC = true
		case "b.error":
			sawBErr = true
		}
	}
	if !sawA || !sawC {
		t.Errorf("other probes did not run: a=%v c=%v", sawA, sawC)
	}
	if !sawBErr {
		t.Error("failing probe did not produce a .error finding")
	}
}

func TestScanPanicContained(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", panic: true},
		&stubProbe{id: "b", findings: []models.Finding{mkFinding("b.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusPartial {
		t.Errorf("status = %q, want partial", res.Status)
	}
	var sawPanic, sawB bool
	for _, f := range res.Findings {
		if f.ProbeID == "a.panic" {
			sawPanic = true
		}
		if f.ProbeID == "b.ok" {
			sawB = true
		}
	}
	if !sawPanic {
		t.Error("panic did not produce a .panic finding")
	}
	if !sawB {
		t.Error("probe after panic did not run")
	}
}

func TestScanTimeoutIsNotFatal(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "slow", sleep: 100 * time.Millisecond},
		&stubProbe{id: "fast", findings: []models.Finding{mkFinding("fast.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: 10 * time.Millisecond})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	// Timeout alone is not a hard failure — scan should be complete.
	if res.Status != models.ScanStatusComplete {
		t.Errorf("status = %q, want complete (timeout is not a hard fail)", res.Status)
	}
	var sawTimeout bool
	for _, f := range res.Findings {
		if f.ProbeID == "slow.timeout" {
			sawTimeout = true
		}
	}
	if !sawTimeout {
		t.Error("timeout did not produce a .timeout finding")
	}
}

func TestScanInvalidDomain(t *testing.T) {
	s := newStore(t)
	sc := scanner.New(s, []probe.Probe{&stubProbe{id: "a"}}, probe.Config{})
	_, err := sc.Scan(context.Background(), models.Target{Domain: ""})
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}
