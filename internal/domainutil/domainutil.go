// Package domainutil holds small DNS-name helpers shared by probes and
// the scanner that would otherwise be reimplemented per package.
package domainutil

import "strings"

// Registrable returns the last two DNS labels of name — e.g.
// "ns1.provider-a.nl" → "provider-a.nl". This is a heuristic, not a
// public-suffix-list lookup: wanderer vendors no PSL dependency, so a
// two-label ccTLD (e.g. "co.uk") is not special-cased.
func Registrable(name string) string {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return ""
	}
	return strings.Join(labels[len(labels)-2:], ".")
}
