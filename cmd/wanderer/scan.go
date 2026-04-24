package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/MWest2020/wanderer/internal/probe"
	dnsprobe "github.com/MWest2020/wanderer/internal/probe/dns"
	httpprobe "github.com/MWest2020/wanderer/internal/probe/http"
	ipprobe "github.com/MWest2020/wanderer/internal/probe/ip"
	tlsprobe "github.com/MWest2020/wanderer/internal/probe/tls"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runScan executes `wanderer scan <domain>` and returns the intended
// process exit code.
func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	dbPath := fs.String("db", "wanderer.db", "Path to SQLite database")
	geoipPath := fs.String("geoip", envOr("WANDERER_GEOIP_ASN", ""), "Path to GeoLite2-ASN mmdb (ASN + country)")
	geoipCountry := fs.String("geoip-country", envOr("WANDERER_GEOIP_COUNTRY", ""), "Optional GeoLite2-Country mmdb (defaults to --geoip)")
	perProbe := fs.Duration("per-probe-timeout", scanner.DefaultPerProbeTimeout, "Per-probe timeout")
	globalTO := fs.Duration("budget", scanner.DefaultGlobalBudget, "Global scan timeout budget")
	ua := fs.String("user-agent", "Wanderer/0.x", "User-Agent for HTTP probes")
	jsonLogs := fs.Bool("json-logs", false, "Emit logs as JSON (default text)")
	positional, err := parseFlagsInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "usage: wanderer scan [flags] <domain>")
		return 2
	}
	domain := positional[0]

	logger := newLogger(*jsonLogs)
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), *globalTO+5*time.Second)
	defer cancel()

	st, err := store.Open(ctx, "file:"+filepath.Clean(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	probes, err := buildProbes(*geoipPath, *geoipCountry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: %v\n", err)
		return 1
	}

	sc := scanner.New(st, probes, probe.Config{
		PerProbeTimeout: *perProbe,
		UserAgent:       *ua,
	})
	sc.Logger = logger
	sc.GlobalBudget = *globalTO

	scan, err := sc.Scan(ctx, models.Target{Domain: domain})
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: scan: %v\n", err)
		return 1
	}
	printScan(os.Stdout, scan)
	switch scan.Status {
	case models.ScanStatusFailed:
		return 1
	default:
		return 0
	}
}

func buildProbes(geoipPath, geoipCountry string) ([]probe.Probe, error) {
	var ipp *ipprobe.Probe
	if geoipPath != "" {
		p, err := ipprobe.Open(geoipPath, geoipCountry)
		if err != nil {
			return nil, fmt.Errorf("ip probe: %w", err)
		}
		ipp = p
	} else {
		// Degraded mode: no DB available. The probe surfaces a single
		// "ip.unavailable" finding so the absence is visible in
		// output rather than silently missing.
		ipp = &ipprobe.Probe{}
	}
	return []probe.Probe{
		dnsprobe.New(),
		tlsprobe.New(),
		ipp,
		httpprobe.New(),
	}, nil
}

// parseFlagsInterspersed parses a flag.FlagSet over args while allowing
// positional arguments to appear anywhere, not only after all flags.
// Go's stdlib flag package stops at the first non-flag token; this
// helper keeps calling Parse on the remainder and collects positionals.
func parseFlagsInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	remaining := args
	for {
		if err := fs.Parse(remaining); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		remaining = rest[1:]
	}
}

func printScan(w *os.File, scan *models.Scan) {
	fmt.Fprintf(w, "Scan %s  status=%s\n", scan.ID, scan.Status)
	fmt.Fprintf(w, "Started: %s\n", scan.StartedAt.Format(time.RFC3339))
	if scan.EndedAt != nil {
		fmt.Fprintf(w, "Ended:   %s\n", scan.EndedAt.Format(time.RFC3339))
	}
	if scan.Error != "" {
		fmt.Fprintf(w, "Error:   %s\n", scan.Error)
	}
	// Group findings by probe prefix for readability.
	byProbe := map[string][]models.Finding{}
	var probeOrder []string
	for _, f := range scan.Findings {
		prefix := firstSegment(f.ProbeID)
		if _, ok := byProbe[prefix]; !ok {
			probeOrder = append(probeOrder, prefix)
		}
		byProbe[prefix] = append(byProbe[prefix], f)
	}
	sort.Strings(probeOrder)
	for _, p := range probeOrder {
		fmt.Fprintf(w, "\n== %s ==\n", p)
		for _, f := range byProbe[p] {
			fmt.Fprintf(w, "  [%s] %s  subject=%s", f.Severity, f.ProbeID, f.Subject)
			if f.DimensionHint != "" {
				fmt.Fprintf(w, "  dim=%s", f.DimensionHint)
			}
			fmt.Fprintln(w)
			for k, v := range f.Attributes {
				fmt.Fprintf(w, "      %s: %v\n", k, v)
			}
		}
	}
	fmt.Fprintf(w, "\nTotal findings: %d\n", len(scan.Findings))
}

func firstSegment(s string) string {
	for i, r := range s {
		if r == '.' {
			return s[:i]
		}
	}
	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newLogger(jsonLogs bool) *slog.Logger {
	var h slog.Handler
	if jsonLogs {
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	return slog.New(h)
}
