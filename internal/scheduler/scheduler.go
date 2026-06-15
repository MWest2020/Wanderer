package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/MWest2020/wanderer/internal/drift"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// SetDefaultOrganisation records the serve-config fallback slug so
// schedules without their own `organisation:` field have somewhere
// to go. Called by the serve startup once the precedence resolver
// has chosen the right value.
func (s *Scheduler) SetDefaultOrganisation(slug string) {
	s.defaultOrgSlug = slug
}

// Scheduler owns the in-process cron and the fan-out into the
// scanner. Lifecycle: New → Reload → Start → (SIGHUP → Reload) → Stop.
type Scheduler struct {
	defaultOrgSlug string
	store          *store.Store
	scanner        *scanner.Scanner
	logger         *slog.Logger

	mu     sync.Mutex
	cron   *cron.Cron
	loaded []Schedule
}

// New constructs a Scheduler bound to a store and scanner.
func New(st *store.Store, sc *scanner.Scanner, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{store: st, scanner: sc, logger: logger}
}

// Reload swaps in a new schedule set. Existing in-flight jobs run to
// completion; future ticks use the new set. A nil or empty cfg
// removes all schedules.
func (s *Scheduler) Reload(cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg == nil {
		cfg = &Config{}
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if s.cron != nil {
		stopCtx := s.cron.Stop()
		<-stopCtx.Done()
	}
	c := cron.New(cron.WithLogger(cron.PrintfLogger(slogPrintf{s.logger})))
	for _, sched := range cfg.Schedules {
		sched := sched // capture
		if _, err := c.AddFunc(sched.Cron, s.makeJob(sched)); err != nil {
			return fmt.Errorf("scheduler: register %q: %w", sched.Name, err)
		}
	}
	s.cron = c
	s.loaded = append(s.loaded[:0], cfg.Schedules...)
	return nil
}

// Start begins firing jobs. Must be called after Reload.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil {
		return
	}
	s.cron.Start()
	s.logger.Info("scheduler.start", "schedules", len(s.loaded))
}

// Stop blocks until all in-flight jobs finish or the context is done.
func (s *Scheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	c := s.cron
	s.cron = nil
	s.mu.Unlock()
	if c == nil {
		return
	}
	stopCtx := c.Stop()
	select {
	case <-stopCtx.Done():
	case <-ctx.Done():
		s.logger.Warn("scheduler.stop_timeout")
	}
}

// Schedules exposes a copy of the currently loaded schedule set.
func (s *Scheduler) Schedules() []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Schedule, len(s.loaded))
	copy(out, s.loaded)
	return out
}

// RunOnce executes the schedule named sched name immediately. Useful
// for tests and `wanderer serve --run-once <name>` — not wired into
// the CLI yet.
func (s *Scheduler) RunOnce(_ context.Context, name string) error {
	for _, sched := range s.Schedules() {
		if sched.Name == name {
			s.makeJob(sched)()
			return nil
		}
	}
	return errors.New("scheduler: no such schedule")
}

func (s *Scheduler) makeJob(sched Schedule) func() {
	return func() {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("scheduler.panic", "name", sched.Name, "rec", fmt.Sprint(rec))
				s.persistPanicFinding(sched, rec)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.scheduleTimeout(sched))
		defer cancel()
		// Resolve organisation: per-schedule slug overrides the
		// serve-config fallback, which in turn falls back to the
		// seeded `default` org. Lookup happens per-tick so a fresh
		// `wanderer org rename` takes effect without restarting.
		orgSlug := sched.Organisation
		if orgSlug == "" {
			orgSlug = s.defaultOrgSlug
		}
		if orgSlug == "" {
			orgSlug = models.DefaultOrganisationSlug
		}
		org, err := s.store.GetOrganisationBySlug(ctx, orgSlug)
		if err != nil {
			s.logger.Error("scheduler.org_lookup", "name", sched.Name, "slug", orgSlug, "err", err)
			return
		}
		target := models.Target{Domain: sched.Target.Domain, Related: sched.Target.Related, OrganisationID: org.ID}
		s.logger.Info("scheduler.tick", "name", sched.Name, "domain", target.Domain, "organisation", org.Slug)
		scan, err := s.scanner.Scan(ctx, target)
		if err != nil {
			s.logger.Error("scheduler.scan_error", "name", sched.Name, "err", err)
			return
		}
		findings, err := drift.Compute(ctx, s.store, scan)
		if err != nil {
			s.logger.Error("scheduler.drift_error", "name", sched.Name, "err", err)
			return
		}
		if len(findings) == 0 {
			return
		}
		if err := s.store.AppendFindings(ctx, scan.ID, findings); err != nil {
			s.logger.Error("scheduler.persist_drift_error", "name", sched.Name, "err", err)
		}
	}
}

func (s *Scheduler) scheduleTimeout(sched Schedule) time.Duration {
	if sched.Timeout > 0 {
		return sched.Timeout
	}
	return s.scanner.GlobalBudget + 10*time.Second
}

func (s *Scheduler) persistPanicFinding(sched Schedule, rec any) {
	// We do not have a scan ID at this point — drop a synthetic
	// observation onto the most recent scan if we can find one.
	scans, err := s.store.ListScans(context.Background(), store.Selectors{})
	if err != nil || len(scans) == 0 {
		return
	}
	last := scans[len(scans)-1].ID
	_ = s.store.AppendFindings(context.Background(), last, []models.Finding{{
		ProbeID:  "scheduler.panic",
		Subject:  sched.Target.Domain,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"schedule": sched.Name,
			"detail":   fmt.Sprint(rec),
		},
	}})
}

// slogPrintf adapts a *slog.Logger to robfig/cron's PrintfLogger,
// which expects a value with a Printf(string, ...any) method.
type slogPrintf struct{ logger *slog.Logger }

func (s slogPrintf) Printf(format string, args ...any) {
	s.logger.Info("cron." + fmt.Sprintf(format, args...))
}
