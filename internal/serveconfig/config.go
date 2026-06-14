// Package serveconfig parses the optional YAML config file that
// `wanderer serve --config <path>` reads. Every field is optional;
// the resolver in resolve.go applies the precedence
// flag > env > yaml > default. When --config is unset, the binary
// behaves byte-identically to today.
package serveconfig

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v2"
)

// Config mirrors the wanderer serve flag surface in YAML form.
// A zero Config (or an empty file) is equivalent to passing no
// flags — every value falls through to its hard-coded default
// via the resolver helpers.
type Config struct {
	Listen    string      `yaml:"listen,omitempty"`
	DB        string      `yaml:"db,omitempty"`
	GeoIP     GeoIP       `yaml:"geoip,omitempty"`
	UI        UI          `yaml:"ui,omitempty"`
	OIDC      OIDC        `yaml:"oidc,omitempty"`
	Schedules string      `yaml:"schedules,omitempty"`
	Scan      ScanSection `yaml:"scan,omitempty"`
}

// GeoIP collects the GeoLite2 mmdb paths and the "missing is
// fine" toggle — equivalent to --geoip / --geoip-country / --no-geoip.
type GeoIP struct {
	ASN      string `yaml:"asn,omitempty"`
	Country  string `yaml:"country,omitempty"`
	Optional bool   `yaml:"optional,omitempty"`
}

// UI carries the --ui / --ui-htpasswd toggles.
type UI struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Htpasswd string `yaml:"htpasswd,omitempty"`
}

// OIDC configures Nextcloud (or any OIDC provider) as the login
// for the read-only UI. An empty block leaves OIDC disabled and
// the UI falls back to htpasswd (or open, in dev mode). The secret
// itself is never in YAML — ClientSecretFile points at a file
// mirroring the existing hmac_secret_file convention.
type OIDC struct {
	ProviderURL      string `yaml:"provider_url,omitempty"`
	ClientID         string `yaml:"client_id,omitempty"`
	ClientSecretFile string `yaml:"client_secret_file,omitempty"`
	RedirectURL      string `yaml:"redirect_url,omitempty"`
	// Scopes overrides the default openid/profile/email set.
	Scopes []string `yaml:"scopes,omitempty"`
	// SessionTTL is the hard lifetime of a login session
	// (default 12h when zero).
	SessionTTL time.Duration `yaml:"session_ttl,omitempty"`
	// RevalidateInterval bounds how often a live session is
	// re-checked against the provider's userinfo endpoint. Zero
	// (the default) revalidates on every request so a Nextcloud
	// disable cuts access immediately.
	RevalidateInterval time.Duration `yaml:"revalidate_interval,omitempty"`
	// CookieSecure sets the Secure flag on the session cookie.
	// Defaults to true; set false only for local plain-http use.
	CookieSecure *bool `yaml:"cookie_secure,omitempty"`
}

// Enabled reports whether the operator filled in the OIDC block.
// ProviderURL is the load-bearing field; the rest is validated by
// the oidc package when the authenticator is built.
func (o OIDC) Enabled() bool { return o.ProviderURL != "" }

// ScanSection mirrors the scan-tunable flags that apply to every
// scan dispatched through `wanderer serve`.
type ScanSection struct {
	PerProbeTimeout     time.Duration `yaml:"per_probe_timeout,omitempty"`
	Budget              time.Duration `yaml:"budget,omitempty"`
	UserAgent           string        `yaml:"user_agent,omitempty"`
	AllowPrivateTargets bool          `yaml:"allow_private_targets,omitempty"`
	// Organisation is the fallback slug for scans that don't
	// specify one — schedules without an `organisation:` key, and
	// API POST /scans calls without an `organisation` body field.
	// Empty falls through to the seeded `default` organisation.
	Organisation string `yaml:"organisation,omitempty"`
}

// Load reads and parses a YAML config file. A non-existent path
// is a hard error — the operator pointed `--config` at it
// explicitly, so silently treating "missing file" as "empty
// config" would be the surprising failure mode.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("serveconfig: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses raw YAML using strict unmarshal so any unknown
// field — typically a typo like `htpasswrd` for `htpasswd` —
// surfaces as a parse error at startup rather than being silently
// dropped onto a zero default.
func Parse(data []byte) (*Config, error) {
	var c Config
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, fmt.Errorf("serveconfig: parse: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate enforces cross-field invariants beyond what type
// parsing already gives. Most of the YAML is permissive — every
// field is optional with a sensible zero — but a partially filled
// oidc: block is a configuration mistake worth catching at startup
// rather than at first login.
func (c *Config) Validate() error {
	if c.OIDC.Enabled() {
		if c.OIDC.ClientID == "" {
			return fmt.Errorf("serveconfig: oidc.client_id is required when oidc.provider_url is set")
		}
		if c.OIDC.ClientSecretFile == "" {
			return fmt.Errorf("serveconfig: oidc.client_secret_file is required when oidc.provider_url is set")
		}
		if c.OIDC.RedirectURL == "" {
			return fmt.Errorf("serveconfig: oidc.redirect_url is required when oidc.provider_url is set")
		}
	}
	return nil
}
