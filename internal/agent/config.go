// Package agent supports `wanderer agent` — the host-side mode that
// runs inventory (and, in a follow-up, egress) inspectors and writes
// Findings either to a local SQLite store or to a remote core over
// HMAC-signed HTTPS. The config file format is documented in
// docs/agent.md.
package agent

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v2"
)

// Config is the parsed wanderer-agent.yaml.
type Config struct {
	Hostname   string                  `yaml:"hostname"`
	Core       CoreConfig              `yaml:"core"`
	Scan       ScanConfig              `yaml:"scan"`
	Inspectors map[string]InspectorCfg `yaml:"inspectors"`
	Egress     EgressConfig            `yaml:"egress"`
	GeoIP      GeoIPConfig             `yaml:"geoip"`
}

// EgressConfig groups the egress probe scanners. Each toggle is
// opt-in (default false) so a freshly-installed agent emits no
// egress findings until the operator names what it should look at.
type EgressConfig struct {
	ConfigFiles EgressConfigFiles `yaml:"configfiles"`
	ProcEnv     EgressProcEnv     `yaml:"procenv"`
	Systemd     EgressSystemd     `yaml:"systemd"`
	Flow        EgressFlow        `yaml:"flow"`
}

// EgressFlow toggles the eBPF-based runtime flow probe. Default off:
// loading a kernel program needs CAP_BPF / CAP_PERFMON (or root),
// which an operator must consciously accept. The eBPF kernel object
// is not yet shipped in this build — see ADR-0010 — so on every
// host the inspector currently surfaces as
// inventory.egress.flow.unavailable when enabled.
type EgressFlow struct {
	Enabled    bool                 `yaml:"enabled"`
	Window     time.Duration        `yaml:"window,omitempty"`
	ReverseDNS EgressFlowReverseDNS `yaml:"reverse_dns,omitempty"`
}

// EgressFlowReverseDNS toggles best-effort PTR annotation on flow
// Findings. Default off: a PTR query leaks the observation back
// through the host's resolver, which a sovereignty monitor cannot
// default-on without consent. See ADR-0010 for the tradeoff.
type EgressFlowReverseDNS struct {
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// EgressConfigFiles enumerates the directories the configfiles
// scanner walks.
type EgressConfigFiles struct {
	Enabled bool     `yaml:"enabled"`
	Paths   []string `yaml:"paths,omitempty"`
}

// EgressProcEnv toggles /proc/<pid>/environ scanning.
type EgressProcEnv struct {
	Enabled bool `yaml:"enabled"`
}

// EgressSystemd toggles unit-file scanning.
type EgressSystemd struct {
	Enabled bool     `yaml:"enabled"`
	Dirs    []string `yaml:"dirs,omitempty"`
}

// GeoIPConfig points at the same GeoLite2 mmdb files the perimeter
// IP probe uses. When set, the egress probe annotates each Finding
// with ASN/country information.
type GeoIPConfig struct {
	ASN     string `yaml:"asn,omitempty"`
	Country string `yaml:"country,omitempty"`
}

// CoreConfig is where the agent ships its findings.
type CoreConfig struct {
	Mode           string `yaml:"mode"`            // local | remote
	DB             string `yaml:"db,omitempty"`    // mode=local
	URL            string `yaml:"url,omitempty"`   // mode=remote
	HMACSecretFile string `yaml:"hmac_secret_file,omitempty"`
	TargetID       string `yaml:"target_id,omitempty"`
	// OutboxDir is the directory the remote-mode agent uses to
	// persist failed batches between ticks. Empty defaults to
	// `/var/lib/wanderer/agent/outbox`. Mode=local ignores this.
	OutboxDir string `yaml:"outbox_dir,omitempty"`
	// OutboxMaxBytes caps the on-disk outbox footprint. Zero defaults
	// to DefaultOutboxMaxBytes (100 MiB).
	OutboxMaxBytes int64 `yaml:"outbox_max_bytes,omitempty"`
}

// ScanConfig controls cadence.
type ScanConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// InspectorCfg is the per-inspector toggle and its inspector-specific
// fields. Unknown fields parse cleanly because each inspector reads
// only the keys it cares about.
type InspectorCfg struct {
	Enabled  bool     `yaml:"enabled"`
	Socket   string   `yaml:"socket,omitempty"`
	Managers []string `yaml:"managers,omitempty"`
	OccPath  string   `yaml:"occ_path,omitempty"`
	RunAs    string   `yaml:"run_as,omitempty"`
}

// LoadConfig reads and validates an agent config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agent: read %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig parses raw YAML.
func ParseConfig(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("agent: parse: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate enforces the cross-field invariants the config has.
func (c *Config) Validate() error {
	if c.Hostname == "" {
		return errors.New("agent: hostname is required")
	}
	switch c.Core.Mode {
	case "local":
		if c.Core.DB == "" {
			return errors.New("agent: core.db is required in mode=local")
		}
	case "remote":
		if c.Core.URL == "" {
			return errors.New("agent: core.url is required in mode=remote")
		}
		if c.Core.HMACSecretFile == "" {
			return errors.New("agent: core.hmac_secret_file is required in mode=remote")
		}
		if c.Core.TargetID == "" {
			return errors.New("agent: core.target_id is required in mode=remote")
		}
	default:
		return fmt.Errorf("agent: core.mode must be local or remote (got %q)", c.Core.Mode)
	}
	if c.Scan.Timeout < 0 {
		return errors.New("agent: scan.timeout must be >= 0")
	}
	return nil
}
