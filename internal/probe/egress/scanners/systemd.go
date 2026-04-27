package scanners

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Systemd reads systemd unit files under the configured roots and
// yields one Candidate per `Environment=KEY=VAL` directive and per
// KEY=VAL line of every `EnvironmentFile=...` referenced. Default
// roots cover the system and user-runtime paths a typical Linux
// installation uses.
type Systemd struct {
	// UnitDirs is the list of paths to walk. If empty, sensible
	// defaults are used.
	UnitDirs []string
}

func (Systemd) ID() string { return "systemd" }

func (s Systemd) Available() (bool, string) {
	for _, dir := range s.dirs() {
		if _, err := os.Stat(dir); err == nil {
			return true, ""
		}
	}
	return false, "no systemd unit directories found"
}

func (s Systemd) dirs() []string {
	if len(s.UnitDirs) > 0 {
		return s.UnitDirs
	}
	return []string{"/etc/systemd/system", "/lib/systemd/system", "/usr/lib/systemd/system"}
}

func (s Systemd) Scan(ctx context.Context) ([]Candidate, error) {
	var out []Candidate
	for _, dir := range s.dirs() {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(p, ".service") && !strings.HasSuffix(p, ".socket") && !strings.HasSuffix(p, ".timer") {
				return nil
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			out = append(out, parseUnit(p, string(body))...)
			return nil
		})
	}
	return out, nil
}

// parseUnit pulls Environment= and EnvironmentFile= directives.
func parseUnit(source, body string) []Candidate {
	var out []Candidate
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "Environment=") {
			rest := strings.TrimPrefix(line, "Environment=")
			rest = strings.Trim(rest, `"`)
			eq := strings.IndexByte(rest, '=')
			if eq <= 0 {
				continue
			}
			out = append(out, Candidate{
				Source: source,
				Key:    rest[:eq],
				Value:  rest[eq+1:],
			})
		}
		if strings.HasPrefix(line, "EnvironmentFile=") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "EnvironmentFile="))
			path = strings.TrimPrefix(path, "-")
			body, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			out = append(out, parseEnvFile(path, string(body))...)
		}
	}
	return out
}
