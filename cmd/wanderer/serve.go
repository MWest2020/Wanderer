package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/scheduler"
	"github.com/MWest2020/wanderer/internal/store"
)

// runServe starts the HTTP API and (optionally) the cron scheduler.
func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", envOr("WANDERER_LISTEN", ":8080"), "HTTP listen address")
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	geoipPath := fs.String("geoip", envOr("WANDERER_GEOIP_ASN", ""), "Path to GeoLite2-ASN mmdb")
	geoipCountry := fs.String("geoip-country", envOr("WANDERER_GEOIP_COUNTRY", ""), "Optional GeoLite2-Country mmdb")
	perProbe := fs.Duration("per-probe-timeout", scanner.DefaultPerProbeTimeout, "Per-probe timeout")
	globalTO := fs.Duration("budget", scanner.DefaultGlobalBudget, "Global scan timeout budget")
	ua := fs.String("user-agent", "Wanderer/0.x", "User-Agent for HTTP probes")
	allowPrivate := fs.Bool("allow-private-targets", false, "Allow scanning RFC1918 / loopback / cloud-metadata addresses (default off)")
	schedulesPath := fs.String("schedules", envOr("WANDERER_SCHEDULES", ""), "Optional cron schedules YAML file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	logger := newLogger(true)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
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
	sc := scanner.New(st, probes, probe.Config{PerProbeTimeout: *perProbe, UserAgent: *ua, AllowPrivateTargets: *allowPrivate})
	sc.Logger = logger
	sc.GlobalBudget = *globalTO

	// Schedules: load and validate before listening so a bad cron
	// expression fails the process at startup, not silently.
	var sched *scheduler.Scheduler
	if *schedulesPath != "" {
		cfg, err := scheduler.LoadConfig(*schedulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: schedules: %v\n", err)
			return 1
		}
		sched = scheduler.New(st, sc, logger)
		if err := sched.Reload(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: schedules: %v\n", err)
			return 1
		}
		sched.Start()
		defer sched.Stop(context.Background())
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.Router(st, sc, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	hup := make(chan os.Signal, 1)
	if sched != nil {
		signal.Notify(hup, syscall.SIGHUP)
		go func() {
			for range hup {
				cfg, err := scheduler.LoadConfig(*schedulesPath)
				if err != nil {
					logger.Error("schedules.reload_error", "err", err)
					continue
				}
				if err := sched.Reload(cfg); err != nil {
					logger.Error("schedules.reload_error", "err", err)
					continue
				}
				sched.Start()
				logger.Info("schedules.reloaded", "schedules", len(cfg.Schedules))
			}
		}()
	}

	go func() {
		logger.Info("serve.start", "addr", *addr, "schedules", *schedulesPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve.error", "err", err)
			cancel()
		}
	}()
	select {
	case <-sig:
		logger.Info("serve.stop", "reason", "signal")
	case <-ctx.Done():
		logger.Info("serve.stop", "reason", "ctx")
	}
	shutdownCtx, scancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer scancel()
	_ = srv.Shutdown(shutdownCtx)
	return 0
}
