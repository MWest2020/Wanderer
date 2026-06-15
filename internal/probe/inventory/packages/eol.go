package packages

import "strings"

// eolMinimums lists the lowest version we still consider "current"
// for a handful of well-known packages. Everything older surfaces as
// SeverityObservation so the assessor or report can call attention
// to it. The list is intentionally short — extend it deliberately.
var eolMinimums = map[string]string{
	"php":     "8.1",
	"php-cli": "8.1",
	"python3": "3.10",
	"openssl": "3.0",
	"nodejs":  "20.0",
}

// isEOL reports whether (name, version) is considered EOL according
// to eolMinimums. The comparison is lexicographic on the version's
// dotted prefix — sufficient for major-version gating, which is all
// the table expresses.
func isEOL(name, version string) bool {
	minVer, ok := eolMinimums[strings.ToLower(name)]
	if !ok {
		return false
	}
	return versionLess(version, minVer)
}

// versionLess returns true when a < b under dotted-numeric ordering,
// dropping any non-numeric suffix on each segment. Suitable for the
// minor-major comparisons eolMinimums expresses.
func versionLess(a, b string) bool {
	pa := splitDotted(a)
	pb := splitDotted(b)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] < pb[i] {
			return true
		}
		if pa[i] > pb[i] {
			return false
		}
	}
	return len(pa) < len(pb)
}

func splitDotted(s string) []int {
	var out []int
	cur := 0
	seen := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			cur = cur*10 + int(c-'0')
			seen = true
			continue
		}
		if c == '.' {
			if seen {
				out = append(out, cur)
			} else {
				out = append(out, 0)
			}
			cur, seen = 0, false
			continue
		}
		// Stop at first non-numeric, non-dot character (e.g. "-" or "+").
		break
	}
	if seen {
		out = append(out, cur)
	}
	return out
}
