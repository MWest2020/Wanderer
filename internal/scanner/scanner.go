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
	"sync"
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

// Publisher is the optional post-scan hook: a completed, persisted
// scan's artefacts are handed off to an external sink (e.g. the
// Nextcloud WebDAV exporter). Implementations MUST NOT block scan
// completion or surface errors — the scan is already persisted when
// Publish is called, so a publish failure is the publisher's own
// concern to log. Nil disables the hook.
type Publisher interface {
	Publish(scanID string)
}

// Scanner runs the configured probes against a target.
type Scanner struct {
	Store  *store.Store
	Probes []probe.Probe
	Config probe.Config
	Logger *slog.Logger

	GlobalBudget time.Duration

	// Publisher, when set, receives each completed scan's ID after
	// the scan is persisted. Opt-in via serve.yaml's nextcloud: block.
	Publisher Publisher
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

	// Two-pass execution. Pass 1 (every probe except `ip`) runs
	// concurrently with errgroup; pass 2 (the `ip` probe) sees a
	// Target enriched with hosts the pass-1 probes discovered.
	var pass1, pass2 []probe.Probe
	for _, p := range s.Probes {
		if p.ID() == "ip" {
			pass2 = append(pass2, p)
		} else {
			pass1 = append(pass1, p)
		}
	}

	pass1Findings, pass1Failed := s.runPassConcurrent(rootCtx, pass1, target, scan.ID, logger)
	scan.Findings = append(scan.Findings, pass1Findings...)

	enriched := expandRelatedFromFindings(target, scan.Findings)
	pass2Findings, pass2Failed := s.runPassConcurrent(rootCtx, pass2, enriched, scan.ID, logger)
	scan.Findings = append(scan.Findings, pass2Findings...)

	// Synthesis: correlate dns.mx × ip.asn into one observed
	// mail-routing Finding ("inbound mail lands at <operator>
	// (<country>)"). Runs after pass 2 because it needs the ip.asn
	// lookups the IP probe ran on the MX hosts pass 1 discovered. The
	// observed fact leads; the wand rule annotates the score.
	if mr, ok := synthesiseMailRouting(enriched, scan.Findings); ok {
		scan.Findings = append(scan.Findings, mr)
		if err := s.Store.AppendFindings(rootCtx, scan.ID, []models.Finding{mr}); err != nil {
			logger.Error("scan.persist_failed", "probe", mr.ProbeID, "err", err)
		}
	}

	// Synthesis: correlate dns.ns × ip.asn into one observed DNS-hosting
	// Finding ("DNS for X is run by <operator> (<country>)"). Same shape
	// as mail routing — the observed fact leads; the wand NS rule
	// annotates the EEA-jurisdiction score.
	if dh, ok := synthesiseDNSHosting(enriched, scan.Findings); ok {
		scan.Findings = append(scan.Findings, dh)
		if err := s.Store.AppendFindings(rootCtx, scan.ID, []models.Finding{dh}); err != nil {
			logger.Error("scan.persist_failed", "probe", dh.ProbeID, "err", err)
		}
	}

	// Synthesis: correlate the apex dns.a/dns.aaaa addresses × ip.asn into
	// one observed hosting-identity Finding ("X is hosted at <operator>
	// (<country>)"). The fourth who/where twin — the observed fact leads;
	// the wand apex rule annotates the EEA-jurisdiction score.
	if hi, ok := synthesiseHostingIdentity(enriched, scan.Findings); ok {
		scan.Findings = append(scan.Findings, hi)
		if err := s.Store.AppendFindings(rootCtx, scan.ID, []models.Finding{hi}); err != nil {
			logger.Error("scan.persist_failed", "probe", hi.ProbeID, "err", err)
		}
	}

	// Synthesis: correlate http.third_party (+ resource kinds) × ip.asn
	// into one observed origin-map Finding ("X loads fonts from Google
	// (US), …"), grouped by vendor. The first Wave-2 surface signal — the
	// observed map leads; the wand third-party rule annotates the
	// in/out-EEA count and names the non-EEA vendors.
	if om, ok := synthesiseOriginMap(enriched, scan.Findings); ok {
		scan.Findings = append(scan.Findings, om)
		if err := s.Store.AppendFindings(rootCtx, scan.ID, []models.Finding{om}); err != nil {
			logger.Error("scan.persist_failed", "probe", om.ProbeID, "err", err)
		}
	}

	// Synthesis: correlate the apex ip.asn × http.response (server header)
	// × tls.issuer into one observed CDN-front Finding ("X's apex is
	// fronted by Cloudflare (US)") — reframing a fronted apex that the
	// hosting signal reads as "hosted at". The observed fact leads; the
	// wand hyperscaler rule annotates the US-reach score.
	if cf, ok := synthesiseCDNFront(enriched, scan.Findings); ok {
		scan.Findings = append(scan.Findings, cf)
		if err := s.Store.AppendFindings(rootCtx, scan.ID, []models.Finding{cf}); err != nil {
			logger.Error("scan.persist_failed", "probe", cf.ProbeID, "err", err)
		}
	}

	totalProbes := len(pass1) + len(pass2)
	failed := pass1Failed + pass2Failed
	completed := totalProbes - failed
	anyCompleted := completed > 0
	anyFailed := failed > 0
	allFailed := completed == 0

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

	// Post-scan publication hook. Runs after the scan is fully
	// persisted, so a publish failure never changes the scan's
	// recorded status; the publisher owns its own timeout + logging.
	if s.Publisher != nil {
		s.Publisher.Publish(scan.ID)
	}
	return scan, nil
}

// runPassConcurrent runs every probe in `probes` concurrently against
// target, persisting findings as they arrive. It returns the
// aggregated findings and the number of probes whose Run errored.
// Panics inside a probe are converted by runOne into errors so a
// single faulty probe cannot abort the pass.
func (s *Scanner) runPassConcurrent(ctx context.Context, probes []probe.Probe, target models.Target, scanID string, logger *slog.Logger) ([]models.Finding, int) {
	if len(probes) == 0 {
		return nil, 0
	}
	type result struct {
		findings []models.Finding
		failed   bool
	}
	results := make([]result, len(probes))
	var wg sync.WaitGroup
	for i, p := range probes {
		i, p := i, p
		probeLogger := logger.With("probe", p.ID())
		wg.Add(1)
		go func() {
			defer wg.Done()
			findings, err := s.runOne(ctx, p, target, probeLogger)
			if err != nil {
				findings = append(findings, models.Finding{
					ProbeID:    p.ID() + ".error",
					Subject:    target.Domain,
					Severity:   models.SeverityConcern,
					Attributes: map[string]any{"error": err.Error()},
				})
				results[i] = result{failed: true, findings: findings}
			} else {
				results[i] = result{findings: findings}
			}
			// Persist each probe's findings the moment it finishes,
			// not after the whole pass. The pass is gated by its
			// slowest probe (transit waits up to 30s), so a deferred
			// write left the findings table empty until the end and
			// made live views — the UI scan-status count — sit at 0
			// and then jump. Writes serialise on the single DB conn.
			if len(findings) > 0 {
				if err := s.Store.AppendFindings(ctx, scanID, findings); err != nil {
					probeLogger.Error("scan.persist_failed", "err", err)
				}
			}
		}()
	}
	wg.Wait()

	var all []models.Finding
	failedCount := 0
	for _, r := range results {
		if r.failed {
			failedCount++
		}
		all = append(all, r.findings...)
	}
	return all, failedCount
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
		case "dns.ns":
			// Nameserver hosts, so the IP probe geo-locates them and
			// the NS-jurisdiction rule can score who runs the DNS.
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
