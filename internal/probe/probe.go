// Package probe defines the shared contract every probe must satisfy.
//
// A probe takes a target and a configuration and returns a slice of
// Findings. That is the only shape the scanner needs to know about. The
// four concrete probes (dns, tls, ip, http) live in sub-packages and
// have no knowledge of each other.
package probe

import (
	"context"
	"net/http"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Probe is implemented by every probe package. A probe MUST honour
// ctx.Done() and MUST NOT panic (the scanner installs a recover, but
// probes should not rely on it).
type Probe interface {
	// ID returns a stable short identifier, e.g. "dns", "tls", "ip",
	// "http". It is used in logging, metrics, and partial-scan
	// accounting.
	ID() string

	// Run executes the probe against target with the given config and
	// returns findings. An error is returned only for catastrophic
	// failures — an inability to return a meaningful Finding at all.
	// Expected adverse conditions (NXDOMAIN, handshake timeout, etc.)
	// should be reported as Findings, not as errors.
	Run(ctx context.Context, target models.Target, cfg Config) ([]models.Finding, error)
}

// Config is the cross-probe configuration injected by the scanner.
// Probe-specific fields live in their own sub-packages and are
// constructed by the scanner from application config.
type Config struct {
	// HTTPClient is used by any probe that needs outbound HTTP
	// (e.g. tls/crt.sh, http/apex fetch). Nil means use a default
	// client with sensible timeouts.
	HTTPClient *http.Client

	// UserAgent is the User-Agent header for any outbound HTTP. Defaults
	// to "Wanderer/0.x".
	UserAgent string

	// PerProbeTimeout is the upper bound for any single probe. The
	// scanner wraps each probe's context with this deadline.
	PerProbeTimeout time.Duration

	// GeoIPPath is the filesystem path to the GeoLite2 database for
	// the IP probe. Empty means the IP probe runs in degraded mode.
	GeoIPPath string

	// GeoIPCountryPath optionally overrides the country DB path; if
	// empty, GeoIPPath is used for both ASN and country.
	GeoIPCountryPath string
}
