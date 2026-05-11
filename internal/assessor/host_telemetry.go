package assessor

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed host_telemetry.yaml
var hostTelemetryYAML []byte

// HostTelemetryAgent is one entry from host_telemetry.yaml.
type HostTelemetryAgent struct {
	Prefix string `yaml:"prefix"`
	Vendor string `yaml:"vendor"`
}

type hostTelemetryFile struct {
	Agents []HostTelemetryAgent `yaml:"agents"`
}

var (
	hostTelemetryOnce  sync.Once
	hostTelemetryList  []HostTelemetryAgent
	hostTelemetryErr   error
	hostTelemetryLower []string
)

func loadHostTelemetry() ([]HostTelemetryAgent, error) {
	hostTelemetryOnce.Do(func() {
		var f hostTelemetryFile
		if err := yaml.UnmarshalStrict(hostTelemetryYAML, &f); err != nil {
			hostTelemetryErr = fmt.Errorf("assessor: parse host_telemetry.yaml: %w", err)
			return
		}
		hostTelemetryList = f.Agents
		hostTelemetryLower = make([]string, len(f.Agents))
		for i, a := range f.Agents {
			hostTelemetryLower[i] = strings.ToLower(a.Prefix)
		}
	})
	return hostTelemetryList, hostTelemetryErr
}

// HostTelemetryMatch returns the matched agent entry when the
// given subject (package name or systemd unit name) prefixes
// any of the host_telemetry.yaml entries. Case-insensitive.
// The second return value reports whether a match was found.
func HostTelemetryMatch(subject string) (HostTelemetryAgent, bool) {
	if _, err := loadHostTelemetry(); err != nil {
		return HostTelemetryAgent{}, false
	}
	s := strings.ToLower(subject)
	for i, p := range hostTelemetryLower {
		if strings.HasPrefix(s, p) {
			return hostTelemetryList[i], true
		}
	}
	return HostTelemetryAgent{}, false
}

// HostTelemetryEntries returns the loaded list (for tests + the
// /ui/reporting rule-description text).
func HostTelemetryEntries() []HostTelemetryAgent {
	list, _ := loadHostTelemetry()
	return list
}
