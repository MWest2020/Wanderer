// Package scanner orchestrates a single scan. It runs each probe
// sequentially, isolates panics and timeouts per probe, and writes
// findings to the store as they arrive. Partial scans are a first-class
// result.
package scanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/metrics"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Default timing budgets.
const (
	DefaultGlobalBudget    = 2 * time.Minute
	DefaultPerProbeTimeout = 30 * time.Second
)

// Scanner runs the configured probes against a target.
type Scanner struct {
	Store  *store.Store
	Probes []probe.Probe
	Config probe.Config
	Logger *slog.Logger

	GlobalBudget time.Duration
}

// New returns a scanner with sensible defaults. Callers MUST provide a
// non-nil store and at least one probe.
func New(s *store.Store, probes []probe.Probe, cfg probe.Config) *Scanner {
	if cfg.PerProbeTimeout == 0 {
		cfg.PerProbeTimeout = DefaultPerProbeTimeout
	}
	return &Scanner{
		Store:        s,
		Probes:       probes,
		Config:       cfg,
		Logger:       slog.Default(),
		GlobalBudget: DefaultGlobalBudget,
	}
}

// Scan runs the probes against target and persists the resulting Scan
// and Findings. It returns the final Scan record. The process-level
// caller (CLI or API) decides exit codes based on Scan.Status.
func (s *Scanner) Scan(ctx context.Context, target models.Target) (*models.Scan, error) {
	if s.Store == nil {
		return nil, errors.New("scanner: store is nil")
	}
	if len(s.Probes) == 0 {
		return nil, errors.New("scanner: no probes configured")
	}
	if err := target.Validate(); err != nil {
		return nil, fmt.Errorf("scanner: target: %w", err)
	}
	if err := s.Store.UpsertTarget(ctx, &target); err != nil {
		return nil, err
	}
	scan, err := s.Store.CreateScan(ctx, target.ID)
	if err != nil {
		return nil, err
	}

	logger := s.Logger.With("scan_id", scan.ID, "domain", target.Domain)
	logger.Info("scan.start", "probes", len(s.Probes))

	rootCtx, cancel := context.WithTimeout(ctx, s.GlobalBudget)
	defer cancel()

	anyCompleted := false
	anyFailed := false
	allFailed := true
	for _, p := range s.Probes {
		probeLogger := logger.With("probe", p.ID())
		// The IP probe correlates juridisch and technologie rules across
		// MX hosts and HTTP third parties. Those hosts are only known
		// after the DNS and HTTP probes have run, so for the IP probe we
		// pass an enriched Target whose Related list includes them.
		// Other probes see the original Target. See ADR-0009 (or the
		// CHANGELOG entry adding two-pass scanning) for rationale.
		runTarget := target
		if p.ID() == "ip" {
			runTarget = expandRelatedFromFindings(target, scan.Findings)
		}
		findings, err := s.runOne(rootCtx, p, runTarget, probeLogger)
		if err != nil {
			anyFailed = true
			findings = append(findings, models.Finding{
				ProbeID:    p.ID() + ".error",
				Subject:    target.Domain,
				Severity:   models.SeverityConcern,
				Attributes: map[string]any{"error": err.Error()},
			})
		} else {
			anyCompleted = true
			allFailed = false
		}
		if len(findings) > 0 {
			if err := s.Store.AppendFindings(rootCtx, scan.ID, findings); err != nil {
				probeLogger.Error("scan.persist_failed", "err", err)
				// Continue; persistence failure on one probe must not
				// take down the scan.
			}
			scan.Findings = append(scan.Findings, findings...)
		}
	}

	status := models.ScanStatusComplete
	var scanErr string
	switch {
	case allFailed:
		status = models.ScanStatusFailed
		scanErr = "all probes failed"
	case anyFailed:
		status = models.ScanStatusPartial
	case !anyCompleted:
		status = models.ScanStatusFailed
	}
	if ferr := s.Store.FinishScan(rootCtx, scan.ID, status, scanErr); ferr != nil {
		logger.Error("scan.finish_failed", "err", ferr)
	}
	scan.Status = status
	now := time.Now().UTC()
	scan.EndedAt = &now
	scan.Error = scanErr

	logger.Info("scan.end", "status", status, "findings", len(scan.Findings))
	return scan, nil
}

// expandRelatedFromFindings returns a copy of target with hosts
// discovered by previously-run probes (MX hosts from dns.mx, third-party
// hosts from http.third_party) appended to Related. Hosts already
// present in target.Domain or target.Related are skipped. The expansion
// is what lets the IP probe correlate jurisdiction across mail vendors
// and front-end third parties — without it, three of the ten DICTU
// rules silently return Onbekend.
func expandRelatedFromFindings(target models.Target, findings []models.Finding) models.Target {
	seen := map[string]bool{}
	norm := func(h string) string {
		return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(h), "."))
	}
	if d := norm(target.Domain); d != "" {
		seen[d] = true
	}
	for _, r := range target.Related {
		if h := norm(r); h != "" {
			seen[h] = true
		}
	}
	var extra []string
	for _, f := range findings {
		var host string
		switch f.ProbeID {
		case "dns.mx":
			if h, ok := f.Attributes["host"].(string); ok {
				host = norm(h)
			}
		case "http.third_party":
			host = norm(f.Subject)
		case "dns.subdomain":
			host = norm(f.Subject)
		}
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		extra = append(extra, host)
	}
	if len(extra) == 0 {
		return target
	}
	sort.Strings(extra)
	out := target
	out.Related = append(append([]string{}, target.Related...), extra...)
	return out
}

// runOne runs a single probe with its own timeout and a panic recover.
// It always returns: errors are expressed as a non-nil error, panics
// become errors too.
func (s *Scanner) runOne(ctx context.Context, p probe.Probe, target models.Target, logger *slog.Logger) (findings []models.Finding, err error) {
	metrics.ProbeRuns.WithLabelValues(p.ID()).Inc()
	ctx, cancel := context.WithTimeout(ctx, s.Config.PerProbeTimeout)
	defer cancel()
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			metrics.ProbeFailures.WithLabelValues(p.ID(), "panic").Inc()
			err = fmt.Errorf("probe %s panicked: %v", p.ID(), r)
			logger.Error("probe.panic", "panic", r, "stack", string(debug.Stack()))
			findings = append(findings, models.Finding{
				ProbeID:    p.ID() + ".panic",
				Subject:    target.Domain,
				Severity:   models.SeverityConcern,
				Attributes: map[string]any{"panic": fmt.Sprint(r)},
			})
		}
	}()
	logger.Debug("probe.start")
	findings, err = p.Run(ctx, target, s.Config)
	dur := time.Since(start)
	switch {
	case err == nil:
		logger.Info("probe.end", "findings", len(findings), "ms", dur.Milliseconds())
	case errors.Is(err, context.DeadlineExceeded):
		metrics.ProbeFailures.WithLabelValues(p.ID(), "timeout").Inc()
		logger.Warn("probe.timeout", "ms", dur.Milliseconds())
		findings = append(findings, models.Finding{
			ProbeID:    p.ID() + ".timeout",
			Subject:    target.Domain,
			Severity:   models.SeverityInfo,
			Attributes: map[string]any{"timeout": true, "budget_ms": s.Config.PerProbeTimeout.Milliseconds()},
		})
		// Timeout is not a "hard" failure for scan status purposes:
		// we clear err so the probe counts as completed-with-warning.
		err = nil
	default:
		metrics.ProbeFailures.WithLabelValues(p.ID(), "error").Inc()
		logger.Error("probe.error", "err", err, "ms", dur.Milliseconds())
	}
	return findings, err
}
