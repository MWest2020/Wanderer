package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubAssessProbe struct{}

func (stubAssessProbe) ID() string { return "stub" }
func (stubAssessProbe) Run(_ context.Context, t models.Target, _ probe.Config) ([]models.Finding, error) {
	return []models.Finding{{
		ProbeID:    "stub.hello",
		Subject:    t.Domain,
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{"ok": true},
	}}, nil
}

func newTestScheduler(t *testing.T) (*Scheduler, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubAssessProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(st, sc, sc.Logger), st
}

// 1.4: schedule levert scan én assessment.
func TestMakeJob_AssessesSuccessfulScan(t *testing.T) {
	sched, st := newTestScheduler(t)
	cfg := &Config{Schedules: []Schedule{{
		Name:   "daily",
		Target: Target{Domain: "example.nl"},
		Cron:   "0 6 * * *",
	}}}
	if err := sched.Reload(cfg); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := sched.RunOnce(context.Background(), "daily"); err != nil {
		t.Fatalf("run once: %v", err)
	}

	scans, err := st.ListScans(context.Background(), store.Selectors{})
	if err != nil {
		t.Fatalf("list scans: %v", err)
	}
	if len(scans) != 1 {
		t.Fatalf("want 1 scan, got %d", len(scans))
	}

	assessments, err := st.ListAssessmentsForScan(context.Background(), scans[0].ID)
	if err != nil {
		t.Fatalf("list assessments: %v", err)
	}
	if len(assessments) != 2 {
		t.Fatalf("want 2 assessments (wand+eucsf), got %d", len(assessments))
	}
	gotFrameworks := map[string]bool{}
	for _, a := range assessments {
		gotFrameworks[a.Framework] = true
	}
	if !gotFrameworks["wand"] || !gotFrameworks["eucsf"] {
		t.Fatalf("want both wand and eucsf assessments, got %v", gotFrameworks)
	}
}

// 1.4: assess: false levert alleen een scan.
func TestMakeJob_AssessFalse_OnlyScan(t *testing.T) {
	sched, st := newTestScheduler(t)
	no := false
	cfg := &Config{Schedules: []Schedule{{
		Name:   "no-assess",
		Target: Target{Domain: "example.nl"},
		Cron:   "0 6 * * *",
		Assess: &no,
	}}}
	if err := sched.Reload(cfg); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := sched.RunOnce(context.Background(), "no-assess"); err != nil {
		t.Fatalf("run once: %v", err)
	}

	scans, err := st.ListScans(context.Background(), store.Selectors{})
	if err != nil {
		t.Fatalf("list scans: %v", err)
	}
	if len(scans) != 1 {
		t.Fatalf("want 1 scan, got %d", len(scans))
	}

	assessments, err := st.ListAssessmentsForScan(context.Background(), scans[0].ID)
	if err != nil {
		t.Fatalf("list assessments: %v", err)
	}
	if len(assessments) != 0 {
		t.Fatalf("want 0 assessments, got %d", len(assessments))
	}
}

type failingPersister struct{ err error }

func (f failingPersister) CreateAssessment(_ context.Context, _ *models.Assessment) error {
	return f.err
}

func (f failingPersister) FindingsForAssessment(_ context.Context, scan *models.Scan) ([]models.Finding, error) {
	return scan.Findings, nil
}

// 1.4 / 1.2: een falende beoordeling laat de scan staan.
func TestAssessScan_FailureLeavesScanInPlace(t *testing.T) {
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubAssessProbe{}}, probe.Config{})
	sc.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	org, err := st.GetOrganisationBySlug(context.Background(), models.DefaultOrganisationSlug)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	scan, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl", OrganisationID: org.ID})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	wantErr := errors.New("boom")
	if err := assessScan(context.Background(), failingPersister{err: wantErr}, scan, "example.nl"); err == nil {
		t.Fatal("want error from assessScan, got nil")
	}

	got, err := st.GetScan(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("scan should still exist after a failed assessment: %v", err)
	}
	if got.ID != scan.ID {
		t.Fatalf("scan id = %q, want %q", got.ID, scan.ID)
	}

	assessments, err := st.ListAssessmentsForScan(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("list assessments: %v", err)
	}
	if len(assessments) != 0 {
		t.Fatalf("want 0 assessments persisted after a failure, got %d", len(assessments))
	}
}
