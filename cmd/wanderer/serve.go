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
	"strings"
	"syscall"
	"time"

	"github.com/MWest2020/wanderer/internal/api"
	"github.com/MWest2020/wanderer/internal/auth/oidc"
	"github.com/MWest2020/wanderer/internal/export/nextcloud"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/scheduler"
	"github.com/MWest2020/wanderer/internal/serveconfig"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/internal/ui"
	"github.com/MWest2020/wanderer/pkg/models"
)

// runServe starts the HTTP API and (optionally) the cron scheduler.
//
// Settings come from four layers, applied highest-precedence first:
// CLI flag > env var > YAML config (via --config) > hard default.
// When --config is unset, the binary behaves byte-identically to
// the no-YAML form — no surprise behaviour for operators who never
// adopt the file.
func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	// Empty / zero defaults so the resolver can distinguish
	// "operator passed the flag" from "operator left it alone".
	addr := fs.String("addr", "", "HTTP listen address (default :8080)")
	dbPath := fs.String("db", "", "Path to SQLite database (default wanderer.db)")
	geoipPath := fs.String("geoip", "", "Path to GeoLite2-ASN mmdb (ASN + country)")
	geoipCountry := fs.String("geoip-country", "", "Optional GeoLite2-Country mmdb (defaults to --geoip)")
	noGeoIP := fs.Bool("no-geoip", false, "Silence the startup warning when GeoLite2 is intentionally absent (CI, offline labs)")
	perProbe := fs.Duration("per-probe-timeout", 0, "Per-probe timeout (default "+scanner.DefaultPerProbeTimeout.String()+")")
	globalTO := fs.Duration("budget", 0, "Global scan timeout budget (default "+scanner.DefaultGlobalBudget.String()+")")
	ua := fs.String("user-agent", "", "User-Agent for HTTP probes (default Wanderer/0.x)")
	allowPrivate := fs.Bool("allow-private-targets", false, "Allow scanning RFC1918 / loopback / cloud-metadata addresses (default off)")
	schedulesPath := fs.String("schedules", "", "Optional cron schedules YAML file")
	uiEnabled := fs.Bool("ui", false, "Mount the read-only UI at /ui/ (default off)")
	uiHtpasswd := fs.String("ui-htpasswd", "", "Path to an htpasswd file (bcrypt entries) protecting /ui/")
	uiAllowScan := fs.Bool("ui-allow-scan", false, "Dev mode: enable the UI 'Scan a target' form (default off = read-only). Gate behind --ui-htpasswd/oidc when exposed.")
	configPath := fs.String("config", envOr("WANDERER_CONFIG", ""), "Optional YAML config file (see docs/operator.md)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	setFlags := serveconfig.SetFlags(fs)

	// Load the YAML if --config was given; missing path is a hard
	// error (operator pointed at it explicitly).
	var cfg *serveconfig.Config
	if *configPath != "" {
		var err error
		cfg, err = serveconfig.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: %v\n", err)
			return 1
		}
	}

	// Resolve every setting through the precedence stack. cfg may
	// be nil; the helpers below treat a nil cfg as "no YAML layer".
	listen := serveconfig.ResolveString(setFlags, "addr", *addr, "WANDERER_LISTEN", cfgListen(cfg), ":8080")
	db := serveconfig.ResolveString(setFlags, "db", *dbPath, "WANDERER_DB", cfgDB(cfg), "wanderer.db")
	asn := serveconfig.ResolveString(setFlags, "geoip", *geoipPath, "WANDERER_GEOIP_ASN", cfgGeoIPASN(cfg), "")
	country := serveconfig.ResolveString(setFlags, "geoip-country", *geoipCountry, "WANDERER_GEOIP_COUNTRY", cfgGeoIPCountry(cfg), "")
	skipGeoWarn := serveconfig.ResolveBool(setFlags, "no-geoip", *noGeoIP, "WANDERER_GEOIP_OPTIONAL", cfgGeoIPOptional(cfg), cfg != nil, false)
	probeTO := serveconfig.ResolveDuration(setFlags, "per-probe-timeout", *perProbe, "WANDERER_PER_PROBE_TIMEOUT", cfgPerProbe(cfg), scanner.DefaultPerProbeTimeout)
	budget := serveconfig.ResolveDuration(setFlags, "budget", *globalTO, "WANDERER_BUDGET", cfgBudget(cfg), scanner.DefaultGlobalBudget)
	userAgent := serveconfig.ResolveString(setFlags, "user-agent", *ua, "WANDERER_USER_AGENT", cfgUserAgent(cfg), "Wanderer/0.x")
	allowPrivateTargets := serveconfig.ResolveBool(setFlags, "allow-private-targets", *allowPrivate, "WANDERER_ALLOW_PRIVATE_TARGETS", cfgAllowPrivate(cfg), cfg != nil, false)
	schedules := serveconfig.ResolveString(setFlags, "schedules", *schedulesPath, "WANDERER_SCHEDULES", cfgSchedules(cfg), "")
	uiOn := serveconfig.ResolveBool(setFlags, "ui", *uiEnabled, "WANDERER_UI_ENABLED", cfgUIEnabled(cfg), cfg != nil, false)
	htpasswd := serveconfig.ResolveString(setFlags, "ui-htpasswd", *uiHtpasswd, "WANDERER_UI_HTPASSWD", cfgUIHtpasswd(cfg), "")
	defaultOrgSlug := serveconfig.ResolveString(setFlags, "organisation", "", "WANDERER_ORGANISATION", cfgOrganisation(cfg), models.DefaultOrganisationSlug)

	warnIfGeoIPMissing(os.Stderr, asn, skipGeoWarn)

	logger := newLogger(true)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st, err := store.Open(ctx, "file:"+filepath.Clean(db))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: open store: %v\n", err)
		return 1
	}
	defer st.Close()

	probes, err := buildProbes(asn, country)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wanderer: %v\n", err)
		return 1
	}
	sc := scanner.New(st, probes, probe.Config{PerProbeTimeout: probeTO, UserAgent: userAgent, AllowPrivateTargets: allowPrivateTargets})
	sc.Logger = logger
	sc.GlobalBudget = budget

	// Optional Nextcloud WebDAV publisher (opt-in via the nextcloud:
	// block). Wired as the scanner's post-scan hook so every completed
	// scan drops a JSON-LD + Markdown bundle into the customer's
	// Nextcloud; a publish failure never fails the scan.
	if cfg != nil && cfg.Nextcloud.Enabled {
		pub, err := buildNextcloudPublisher(st, cfg, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: nextcloud: %v\n", err)
			return 1
		}
		sc.Publisher = pub
		logger.Info("nextcloud.publisher.enabled", "url", cfg.Nextcloud.URL)
	}

	// Schedules: load and validate before listening so a bad cron
	// expression fails the process at startup, not silently.
	var sched *scheduler.Scheduler
	if schedules != "" {
		schedCfg, err := scheduler.LoadConfig(schedules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: schedules: %v\n", err)
			return 1
		}
		sched = scheduler.New(st, sc, logger)
		sched.SetDefaultOrganisation(defaultOrgSlug)
		if err := sched.Reload(schedCfg); err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: schedules: %v\n", err)
			return 1
		}
		sched.Start()
		defer sched.Stop(context.Background())
	}

	root := http.NewServeMux()
	root.Handle("/", api.Router(st, sc, logger))
	if uiOn {
		uiOpts := ui.Options{HtpasswdPath: htpasswd, MountPrefix: "/ui"}
		if serveconfig.ResolveBool(setFlags, "ui-allow-scan", *uiAllowScan, "WANDERER_UI_ALLOW_SCAN", false, false, false) {
			// Dev mode: let the UI trigger scans. The scanner is the
			// same one the API uses.
			uiOpts.Scanner = sc
			if htpasswd == "" && !oidcEnabled(cfg) {
				logger.Warn("ui.allow_scan.unauthenticated", "msg", "--ui-allow-scan is on with no UI auth; do not expose this beyond localhost")
			}
		}
		if oidcEnabled(cfg) {
			auth, err := buildOIDC(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wanderer: oidc: %v\n", err)
				return 1
			}
			uiOpts.Auth = auth
			uiOpts.SessionTTL = cfg.OIDC.SessionTTL
			uiOpts.RevalidateInterval = cfg.OIDC.RevalidateInterval
			uiOpts.CookieSecure = cfg.OIDC.CookieSecure == nil || *cfg.OIDC.CookieSecure
		}
		uiHandler, err := ui.Handler(st, uiOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wanderer: ui: %v\n", err)
			return 1
		}
		root.Handle("/ui/", http.StripPrefix("/ui", uiHandler))
		logger.Info("ui.mounted", "htpasswd", htpasswd != "", "oidc", uiOpts.Auth != nil)
	}
	srv := &http.Server{
		Addr:              listen,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	hup := make(chan os.Signal, 1)
	if sched != nil {
		signal.Notify(hup, syscall.SIGHUP)
		go func() {
			for range hup {
				schedCfg, err := scheduler.LoadConfig(schedules)
				if err != nil {
					logger.Error("schedules.reload_error", "err", err)
					continue
				}
				if err := sched.Reload(schedCfg); err != nil {
					logger.Error("schedules.reload_error", "err", err)
					continue
				}
				sched.Start()
				logger.Info("schedules.reloaded", "schedules", len(schedCfg.Schedules))
			}
		}()
	}

	go func() {
		logger.Info("serve.start", "addr", listen, "schedules", schedules)
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

// cfgX accessors return the YAML value for one setting, or the
// zero value when no config file was loaded. The resolver layer
// treats zero as "not set" so a nil cfg falls cleanly through to
// env / default.
func cfgListen(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.Listen
}

func cfgDB(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.DB
}

func cfgGeoIPASN(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.GeoIP.ASN
}

func cfgGeoIPCountry(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.GeoIP.Country
}

func cfgGeoIPOptional(c *serveconfig.Config) bool {
	if c == nil {
		return false
	}
	return c.GeoIP.Optional
}

func cfgPerProbe(c *serveconfig.Config) time.Duration {
	if c == nil {
		return 0
	}
	return c.Scan.PerProbeTimeout
}

func cfgBudget(c *serveconfig.Config) time.Duration {
	if c == nil {
		return 0
	}
	return c.Scan.Budget
}

func cfgUserAgent(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.Scan.UserAgent
}

func cfgAllowPrivate(c *serveconfig.Config) bool {
	if c == nil {
		return false
	}
	return c.Scan.AllowPrivateTargets
}

func cfgSchedules(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.Schedules
}

func cfgUIEnabled(c *serveconfig.Config) bool {
	if c == nil {
		return false
	}
	return c.UI.Enabled
}

func cfgUIHtpasswd(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.UI.Htpasswd
}

func cfgOrganisation(c *serveconfig.Config) string {
	if c == nil {
		return ""
	}
	return c.Scan.Organisation
}

// oidcEnabled reports whether a usable oidc: block was loaded. OIDC
// is YAML-only (no flag surface) so it requires --config; a nil cfg
// means the operator never opted in.
func oidcEnabled(c *serveconfig.Config) bool {
	return c != nil && c.OIDC.Enabled()
}

// buildNextcloudPublisher reads the app password from disk and
// constructs the WebDAV publisher. TargetDir defaults to "Wanderer".
func buildNextcloudPublisher(st *store.Store, c *serveconfig.Config, logger *slog.Logger) (*nextcloud.Publisher, error) {
	pw, err := os.ReadFile(c.Nextcloud.AppPasswordFile)
	if err != nil {
		return nil, fmt.Errorf("read app_password_file: %w", err)
	}
	targetDir := c.Nextcloud.TargetDir
	if targetDir == "" {
		targetDir = "Wanderer"
	}
	client := nextcloud.NewClient(nextcloud.Config{
		URL:         c.Nextcloud.URL,
		Username:    c.Nextcloud.Username,
		AppPassword: strings.TrimSpace(string(pw)),
		TargetDir:   targetDir,
	})
	return nextcloud.NewPublisher(st, client, logger), nil
}

// buildOIDC reads the client secret from disk (mirroring the
// hmac_secret_file convention) and constructs the authenticator.
// Discovery is deferred, so a provider that is down right now does
// not stop the server from starting.
func buildOIDC(c *serveconfig.Config) (*oidc.Authenticator, error) {
	secret, err := os.ReadFile(c.OIDC.ClientSecretFile)
	if err != nil {
		return nil, fmt.Errorf("read client_secret_file: %w", err)
	}
	return oidc.New(oidc.Config{
		ProviderURL:  c.OIDC.ProviderURL,
		ClientID:     c.OIDC.ClientID,
		ClientSecret: strings.TrimSpace(string(secret)),
		RedirectURL:  c.OIDC.RedirectURL,
		Scopes:       c.OIDC.Scopes,
	})
}
