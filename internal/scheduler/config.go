// Package scheduler runs cron-driven scans inside `wanderer serve`.
// Schedules are read from a YAML file at startup and re-read on
// SIGHUP. Each tick invokes the same scanner the CLI uses; the
// scheduler does not own its own scanner pipeline.
package scheduler

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"go.yaml.in/yaml/v2"
)

// Config is the parsed schedules file.
type Config struct {
	Schedules []Schedule `yaml:"schedules"`
}

// Schedule is one cron entry plus its target.
type Schedule struct {
	Name    string        `yaml:"name"`
	Target  Target        `yaml:"target"`
	Cron    string        `yaml:"cron"`
	Probes  []string      `yaml:"probes,omitempty"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
	// Organisation is the slug this scheduled scan attaches to.
	// Optional: empty falls back to the serve.yaml `scan.organisation`
	// field (or, if that is also empty, the seeded `default` org).
	// Validate emits a clear error if neither layer is set, so an
	// operator can never silently lose track of which organisation
	// a recurring scan belongs to.
	Organisation string `yaml:"organisation,omitempty"`
}

// Target is a thin YAML-friendly mirror of models.Target.
type Target struct {
	Domain  string   `yaml:"domain"`
	Related []string `yaml:"related,omitempty"`
}

// LoadConfig reads and validates a schedules file. Validation rejects
// the file on the first problem so an operator sees the offending
// entry rather than silently mis-scheduled jobs.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scheduler: read %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig parses raw YAML and validates it.
func ParseConfig(data []byte) (*Config, error) {
	var c Config
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, fmt.Errorf("scheduler: parse: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate runs the same checks Run will rely on. Exposed for tests.
func (c *Config) Validate() error {
	parser := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)
	seen := map[string]bool{}
	for i := range c.Schedules {
		s := &c.Schedules[i]
		if s.Name == "" {
			return fmt.Errorf("scheduler: schedules[%d]: name is required", i)
		}
		if seen[s.Name] {
			return fmt.Errorf("scheduler: duplicate schedule name %q", s.Name)
		}
		seen[s.Name] = true
		if s.Target.Domain == "" {
			return fmt.Errorf("scheduler: schedule %q: target.domain is required", s.Name)
		}
		if s.Cron == "" {
			return fmt.Errorf("scheduler: schedule %q: cron is required", s.Name)
		}
		if _, err := parser.Parse(s.Cron); err != nil {
			return fmt.Errorf("scheduler: schedule %q: invalid cron %q: %w", s.Name, s.Cron, err)
		}
	}
	return nil
}

// ErrEmpty is returned by LoadConfig when the file parses but contains
// no schedules — let the caller decide whether that is fatal.
var ErrEmpty = errors.New("scheduler: no schedules defined")
