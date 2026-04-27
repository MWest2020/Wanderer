package scanners

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ProcEnv reads /proc/<pid>/environ for every PID the agent user can
// inspect. Permission denied on a specific PID is silently skipped —
// emitting a Finding for every unreadable PID would produce noise
// proportional to the host's process count.
type ProcEnv struct {
	// Root is the proc filesystem root. Defaults to "/proc". Tests
	// override.
	Root string
}

func (ProcEnv) ID() string { return "procenv" }

func (p ProcEnv) Available() (bool, string) {
	root := p.Root
	if root == "" {
		root = "/proc"
	}
	if _, err := os.Stat(root); err != nil {
		return false, "/proc not available: " + err.Error()
	}
	return true, ""
}

func (p ProcEnv) Scan(ctx context.Context) ([]Candidate, error) {
	root := p.Root
	if root == "" {
		root = "/proc"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Candidate
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !isAllDigits(name) {
			continue
		}
		path := filepath.Join(root, name, "environ")
		body, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
		}
		out = append(out, parseEnviron(name, body)...)
	}
	return out, nil
}

// parseEnviron parses the null-separated KEY=VALUE entries that the
// kernel exposes at /proc/<pid>/environ.
func parseEnviron(pid string, body []byte) []Candidate {
	var out []Candidate
	for _, raw := range strings.Split(string(body), "\x00") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		eq := strings.IndexByte(raw, '=')
		if eq <= 0 {
			continue
		}
		key := raw[:eq]
		val := raw[eq+1:]
		out = append(out, Candidate{Source: "/proc/" + pid + "/environ", Key: key, Value: val})
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
