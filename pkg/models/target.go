package models

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Target is the input to a scan: a single apex domain, optionally with
// related domains that will be treated as part of the same
// organisational footprint (e.g. the .com variant of a .nl primary).
type Target struct {
	ID        string    `json:"id,omitempty"`
	Domain    string    `json:"domain"`
	Related   []string  `json:"related,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// NormaliseDomain lower-cases, trims, and strips a leading scheme or
// trailing dot / path from a user-supplied domain. It does not accept
// IPs, IDNs without conversion, or URLs with paths — those are caller
// errors.
func NormaliseDomain(in string) (string, error) {
	s := strings.TrimSpace(in)
	if s == "" {
		return "", errors.New("domain: empty")
	}
	s = strings.ToLower(s)
	// Strip scheme if present (we accept "https://example.nl" as a
	// convenience even though a Target is a domain, not a URL).
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// Strip any path or query.
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	// Strip trailing dot (FQDN form).
	s = strings.TrimSuffix(s, ".")
	// Strip port if present.
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	if s == "" {
		return "", errors.New("domain: empty after normalisation")
	}
	if ip := net.ParseIP(s); ip != nil {
		return "", fmt.Errorf("domain: %q is an IP, not a domain", in)
	}
	// Minimal sanity: at least one dot, no spaces, printable ASCII.
	if !strings.Contains(s, ".") {
		return "", fmt.Errorf("domain: %q has no TLD", in)
	}
	for _, r := range s {
		if r <= 0x20 || r == 0x7f {
			return "", fmt.Errorf("domain: %q contains control characters", in)
		}
	}
	return s, nil
}

// Validate checks that a Target is usable as a scan input.
func (t *Target) Validate() error {
	d, err := NormaliseDomain(t.Domain)
	if err != nil {
		return err
	}
	t.Domain = d
	for i, r := range t.Related {
		rd, err := NormaliseDomain(r)
		if err != nil {
			return fmt.Errorf("related[%d]: %w", i, err)
		}
		t.Related[i] = rd
	}
	return nil
}
