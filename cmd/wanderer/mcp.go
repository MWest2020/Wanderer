package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/MWest2020/wanderer/internal/mcp"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
)

// runMCP starts the MCP server over stdin/stdout. Logs go to stderr
// so the protocol channel stays clean.
func runMCP(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	dbPath := fs.String("db", envOr("WANDERER_DB", "wanderer.db"), "Path to SQLite database")
	geoipPath := fs.String("geoip", envOr("WANDERER_GEOIP_ASN", ""), "Path to GeoLite2-ASN mmdb")
	geoipCountry := fs.String("geoip-country", envOr("WANDERER_GEOIP_COUNTRY", ""), "Optional GeoLite2-Country mmdb")
	perProbe := fs.Duration("per-probe-timeout", scanner.DefaultPerProbeTimeout, "Per-probe timeout")
	globalTO := fs.Duration("budget", scanner.DefaultGlobalBudget, "Global scan timeout budget")
	ua := fs.String("user-agent", "Wanderer/0.x", "User-Agent for HTTP probes")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
	sc := scanner.New(st, probes, probe.Config{PerProbeTimeout: *perProbe, UserAgent: *ua})
	sc.Logger = logger
	sc.GlobalBudget = *globalTO

	deps := mcp.Deps{Store: st, Scanner: sc}
	static, patterns := mcp.BuildResources(deps)
	srv := &mcp.Server{
		Tools:    mcp.BuildTools(deps),
		Static:   static,
		Patterns: patterns,
		Logger:   logger,
	}

	// Cancel on SIGINT/SIGTERM so an in-flight scan unblocks.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	logger.Info("mcp.start", "db", *dbPath)
	if err := srv.Run(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: mcp: %v\n", err)
		return 1
	}
	return 0
}
