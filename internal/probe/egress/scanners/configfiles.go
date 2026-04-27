package scanners

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFiles walks every path in Paths (recursively) and yields a
// Candidate per `KEY=VALUE` line in `.env`-style files and per
// `key: value` line in YAML/JSON files. The walker refuses to follow
// symlinks that point outside the configured root.
type ConfigFiles struct {
	Paths []string
}

func (ConfigFiles) ID() string { return "configfiles" }

func (c ConfigFiles) Available() (bool, string) {
	if len(c.Paths) == 0 {
		return false, "no paths configured"
	}
	return true, ""
}

func (c ConfigFiles) Scan(ctx context.Context) ([]Candidate, error) {
	var out []Candidate
	for _, root := range c.Paths {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		err = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, fs.ErrPermission) {
					return nil
				}
				return walkErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if d.IsDir() {
				return nil
			}
			// Reject symlinks that escape the configured root.
			real, err := filepath.EvalSymlinks(p)
			if err != nil {
				return nil
			}
			realAbs, _ := filepath.Abs(real)
			if !strings.HasPrefix(realAbs+string(filepath.Separator), absRoot+string(filepath.Separator)) &&
				realAbs != absRoot {
				return nil
			}
			cands, _ := scanFile(p)
			out = append(out, cands...)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return out, err
		}
	}
	return out, nil
}

// scanFile reads a single file and extracts KEY=VALUE / key: value
// pairs. Failures are silent — the file is simply skipped. The body
// is bounded to keep a misconfigured path from causing memory blow-up.
func scanFile(path string) ([]Candidate, error) {
	const maxBytes = 1 << 20
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > int64(maxBytes) {
		return nil, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".env", ".envrc", "":
		return parseEnvFile(path, string(body)), nil
	case ".yaml", ".yml", ".json", ".toml", ".ini", ".conf":
		return parseGenericKV(path, string(body)), nil
	}
	return nil, nil
}

// parseEnvFile parses dotenv-style files: lines of the form
// `KEY=VALUE`, with `#` comments, optional surrounding quotes, and
// blank lines.
func parseEnvFile(source, body string) []Candidate {
	var out []Candidate
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = trimQuotes(val)
		if key == "" {
			continue
		}
		out = append(out, Candidate{Source: source, Key: key, Value: val})
	}
	return out
}

// parseGenericKV is a heuristic walker for YAML/JSON/TOML/INI/.conf
// files. We do not attempt to honour their structure; we look for
// lines that *look like* key/value pairs and yield those. This is
// noisier than a real parser but pulls in zero new dependencies.
func parseGenericKV(source, body string) []Candidate {
	var out []Candidate
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Try `key: value` (YAML), then `key = value` (TOML/INI/.env), then `"key": "value"` (JSON).
		key, val := splitKV(line)
		if key == "" {
			continue
		}
		key = trimQuotes(strings.TrimSpace(key))
		val = trimQuotes(strings.TrimSpace(val))
		if val == "" {
			continue
		}
		out = append(out, Candidate{Source: source, Key: key, Value: val})
	}
	return out
}

func splitKV(line string) (string, string) {
	if i := strings.Index(line, ":"); i > 0 && (i+1 == len(line) || line[i+1] != '/') {
		return line[:i], strings.TrimSpace(line[i+1:])
	}
	if i := strings.IndexByte(line, '='); i > 0 {
		return line[:i], strings.TrimSpace(line[i+1:])
	}
	return "", ""
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, ",")
}
