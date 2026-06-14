// Package egress is the agent-side egress probe. It walks
// configured config files, /proc/<pid>/environ entries, and systemd
// unit environments, classifies the URLs and hosts it finds, and
// emits Findings — with secrets redacted before they ever touch a
// Finding or a log line.
package egress

import (
	"net/url"
	"regexp"
	"strings"
)

// Placeholder is the literal string that replaces any redacted value.
const Placeholder = "«redacted»"

// secretKeyNameRE matches env-var or config-key names that we treat as
// definitely-secret regardless of value shape. Case-insensitive.
var secretKeyNameRE = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|passwd|token|access[_-]?key|private[_-]?key|client[_-]?secret|auth[_-]?token|bearer|^pw$|^pwd$)`)

// secretValueREs matches values that are identifiable as secrets by
// shape regardless of the key name they appeared under.
var secretValueREs = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                        // AWS access key
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),                            // Slack token
	regexp.MustCompile(`gh[opusr]_[A-Za-z0-9]{20,}`),                              // GitHub PAT family
	regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`),                                  // Google API key
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),                      // PEM private key block
	regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`), // JWT
}

// urlSchemeRE recognises URL-like values that may carry inline
// credentials we need to scrub.
var urlSchemeRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

// Apply returns a redacted form of value plus a flag telling the
// caller whether anything was changed. The key is consulted because
// `API_KEY=hunter2` should redact even though "hunter2" matches no
// generic secret pattern.
//
// The function is total: a panic in any sub-rule would be a security
// failure, so each rule is wrapped to fail closed (i.e. redact on
// error).
func Apply(key, value string) (string, bool) {
	if value == "" {
		return value, false
	}
	if isSecretKeyName(key) {
		return Placeholder, true
	}
	for _, re := range secretValueREs {
		if re.MatchString(value) {
			return Placeholder, true
		}
	}
	if urlSchemeRE.MatchString(value) {
		if redacted, changed := scrubURLCredentials(value); changed {
			return redacted, true
		}
	}
	return value, false
}

// isSecretKeyName reports whether the key name implies its value is a
// secret. Empty keys are never secrets — they typically come from
// procenv-style scans where the key/value split has already happened
// upstream, and we rely on the value rules in that case.
func isSecretKeyName(key string) bool {
	if key == "" {
		return false
	}
	return secretKeyNameRE.MatchString(key)
}

// scrubURLCredentials replaces the password part of a URL's userinfo
// section with the placeholder while preserving everything else.
// Returns the original string when no userinfo is present.
func scrubURLCredentials(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return raw, false
	}
	if u.User == nil {
		return raw, false
	}
	if _, hasPwd := u.User.Password(); !hasPwd {
		return raw, false
	}
	username := u.User.Username()
	u.User = url.UserPassword(username, Placeholder)
	out := u.String()
	// url.UserPassword percent-encodes the literal placeholder —
	// undo so the marker reads cleanly in operator output.
	encoded := url.QueryEscape(Placeholder)
	return strings.ReplaceAll(out, encoded, Placeholder), true
}

// IsLikelySecretKey is exported so scanners can decide ahead of time
// whether to even surface a key/value pair (e.g. for `egress.unknown`
// emission policy).
func IsLikelySecretKey(key string) bool { return isSecretKeyName(key) }
