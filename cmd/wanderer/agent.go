package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/MWest2020/wanderer/internal/agent"
	"github.com/MWest2020/wanderer/internal/probe/egress"
	egressscanners "github.com/MWest2020/wanderer/internal/probe/egress/scanners"
	"github.com/MWest2020/wanderer/internal/probe/inventory"
	"github.com/MWest2020/wanderer/internal/probe/inventory/docker"
	"github.com/MWest2020/wanderer/internal/probe/inventory/nextcloud"
	"github.com/MWest2020/wanderer/internal/probe/inventory/packages"
	"github.com/MWest2020/wanderer/internal/probe/inventory/systemd"
	ipprobe "github.com/MWest2020/wanderer/internal/probe/ip"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runAgent executes `wanderer agent --config <file>`. It runs the
// inventory inspectors and the egress probe on a loop and ships
// findings either to a local SQLite store or to a remote core over
// HMAC-signed HTTPS.
func runAgent(args []string) int {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	cfgPath := fs.String("config", envOr("WANDERER_AGENT_CONFIG", "wanderer-agent.yaml"), "Path to wanderer-agent.yaml")
	once := fs.Bool("once", false, "Run inspectors once and exit")
	vendorsPath := fs.String("vendors", "", "Path to a custom egress vendors YAML (overrides the embedded list; falls back to WANDERER_VENDORS)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	logger := newLogger(true)
	slog.SetDefault(logger)

	if v, err := egress.LoadVendors(*vendorsPath); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent: %v\n", err)
		return 1
	} else {
		egress.Configure(v)
	}

	cfg, err := agent.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent: %v\n", err)
		return 1
	}

	inspectors := buildInspectors(cfg)
	egressProbe := buildEgressProbe(cfg, logger)
	timeout := cfg.Scan.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	interval := cfg.Scan.Interval
	if *once {
		interval = 0
	}

	switch cfg.Core.Mode {
	case "local":
		return runAgentLocal(logger, cfg, inspectors, egressProbe, timeout, interval)
	case "remote":
		return runAgentRemote(logger, cfg, inspectors, egressProbe, timeout, interval)
	}
	fmt.Fprintf(os.Stderr, "wanderer agent: unknown core.mode %q\n", cfg.Core.Mode)
	return 1
}

// buildEgressProbe wires the egress probe according to cfg. When no
// egress scanners are enabled the returned Probe has zero scanners
// and Inspect returns no findings — the agent simply emits no
// egress findings until the operator opts in.
func buildEgressProbe(cfg *agent.Config, logger *slog.Logger) egress.Probe {
	var scs []egressscanners.Scanner
	if cfg.Egress.ConfigFiles.Enabled {
		scs = append(scs, egressscanners.ConfigFiles{Paths: cfg.Egress.ConfigFiles.Paths})
	}
	if cfg.Egress.ProcEnv.Enabled {
		scs = append(scs, egressscanners.ProcEnv{})
	}
	if cfg.Egress.Systemd.Enabled {
		scs = append(scs, egressscanners.Systemd{UnitDirs: cfg.Egress.Systemd.Dirs})
	}
	var resolver egress.HostResolver
	if cfg.GeoIP.ASN != "" {
		ip, err := ipprobe.Open(cfg.GeoIP.ASN, cfg.GeoIP.Country)
		if err != nil {
			logger.Warn("agent.geoip", "err", err)
		} else {
			resolver = egress.IPResolver{IP: ip}
		}
	}
	return egress.Probe{Scanners: scs, Resolver: resolver}
}

func buildInspectors(cfg *agent.Config) []inventory.Inspector {
	var out []inventory.Inspector
	if c := cfg.Inspectors["systemd"]; c.Enabled {
		out = append(out, systemd.Systemd{})
	}
	if c := cfg.Inspectors["docker"]; c.Enabled {
		out = append(out, docker.Docker{Socket: c.Socket})
	}
	if c := cfg.Inspectors["packages"]; c.Enabled {
		managers := c.Managers
		if len(managers) == 0 {
			managers = []string{"dpkg", "rpm"}
		}
		for _, m := range managers {
			switch strings.ToLower(m) {
			case "dpkg":
				out = append(out, packages.Dpkg{})
			case "rpm":
				out = append(out, packages.Rpm{})
			}
		}
	}
	if c := cfg.Inspectors["nextcloud"]; c.Enabled {
		out = append(out, nextcloud.Nextcloud{OccPath: c.OccPath, RunAs: c.RunAs})
	}
	return out
}

func runAgentLocal(logger *slog.Logger, cfg *agent.Config, inspectors []inventory.Inspector, egressProbe egress.Probe, timeout, interval time.Duration) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st, err := store.Open(ctx, "file:"+filepath.Clean(cfg.Core.DB))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	tgt := &models.Target{Domain: cfg.Hostname, Kind: models.TargetKindHost}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent: upsert target: %v\n", err)
		return 1
	}

	return loop(logger, interval, func() {
		runCtx, runCancel := context.WithTimeout(ctx, timeout)
		defer runCancel()
		scan, err := st.CreateScan(runCtx, tgt.ID)
		if err != nil {
			logger.Error("agent.scan_create", "err", err)
			return
		}
		findings := inventory.Inspect(runCtx, inspectors)
		findings = append(findings, egressProbe.Inspect(runCtx)...)
		if err := st.AppendFindings(runCtx, scan.ID, findings); err != nil {
			logger.Error("agent.persist", "err", err)
		}
		if err := st.FinishScan(runCtx, scan.ID, models.ScanStatusComplete, ""); err != nil {
			logger.Error("agent.finish", "err", err)
		}
		logger.Info("agent.tick", "scan", scan.ID, "findings", len(findings))
	})
}

func runAgentRemote(logger *slog.Logger, cfg *agent.Config, inspectors []inventory.Inspector, egressProbe egress.Probe, timeout, interval time.Duration) int {
	secret, err := os.ReadFile(cfg.Core.HMACSecretFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer agent: read hmac secret: %v\n", err)
		return 1
	}
	r := &agent.Remote{
		BaseURL:  cfg.Core.URL,
		Secret:   secret,
		Hostname: cfg.Hostname,
	}
	return loop(logger, interval, func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		findings := inventory.Inspect(ctx, inspectors)
		findings = append(findings, egressProbe.Inspect(ctx)...)
		if err := r.Send(ctx, cfg.Core.TargetID, findings); err != nil {
			logger.Error("agent.send", "err", err)
		} else {
			logger.Info("agent.tick", "findings", len(findings))
		}
	})
}

// loop calls do once when interval is 0, otherwise every interval
// until the process receives SIGINT/SIGTERM.
func loop(logger *slog.Logger, interval time.Duration, do func()) int {
	do()
	if interval == 0 {
		return 0
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-t.C:
			do()
		case <-sig:
			logger.Info("agent.stop", "reason", "signal")
			return 0
		}
	}
}
