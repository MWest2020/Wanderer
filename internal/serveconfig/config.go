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
// parsing already gives. Currently the YAML is permissive —
// every field is optional with a sensible zero — so this is a
// hook for future invariants (e.g., "geoip.country requires
// geoip.asn"). Today it is a no-op so an empty config validates
// cleanly.
func (c *Config) Validate() error {
	return nil
}
