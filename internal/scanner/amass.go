package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LoadAmassFQDNs reads an Amass `enum -json` output file and returns
// the FQDNs it carries. Malformed lines are logged at WARN level and
// skipped — a partial Amass file should not block the scan. The
// returned slice is deduplicated, lower-cased, and sorted.
func LoadAmassFQDNs(path string, logger *slog.Logger) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("amass: open %s: %w", path, err)
	}
	defer f.Close()
	if logger == nil {
		logger = slog.Default()
	}
	return parseAmass(f, logger)
}

func parseAmass(r io.Reader, logger *slog.Logger) ([]string, error) {
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row struct {
			Name   string `json:"name"`
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			logger.Warn("scanner.amass.malformed_line", "err", err.Error())
			continue
		}
		name := strings.TrimSpace(strings.ToLower(row.Name))
		name = strings.TrimSuffix(name, ".")
		if name == "" || !strings.Contains(name, ".") {
			continue
		}
		seen[name] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("amass: read: %w", err)
	}
	if len(seen) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	// Stable order so callers (and the eventual Target.Related slice)
	// stay deterministic.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out, nil
}
