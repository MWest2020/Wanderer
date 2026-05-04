package models

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// TargetKind distinguishes a public-domain Target (the perimeter
// scanner's input shape) from a host-only Target (the agent modus,
// where the subject is a Linux hostname that may not carry a TLD).
type TargetKind string

const (
	// TargetKindDomain is the default. Domain must be a public domain
	// with at least one dot (the existing TLD validation).
	TargetKindDomain TargetKind = "domain"
	// TargetKindHost relaxes the TLD requirement: bare hostnames like
	// `webapp-01` are accepted. Used by `wanderer agent` whose
	// hostname is the host's local identity, not a public domain.
	TargetKindHost TargetKind = "host"
)

// Target is the input to a scan: a single apex domain, optionally with
// related domains that will be treated as part of the same
// organisational footprint (e.g. the .com variant of a .nl primary).
type Target struct {
	ID             string     `json:"id,omitempty"`
	Domain         string     `json:"domain"`
	Related        []string   `json:"related,omitempty"`
	Kind           TargetKind `json:"kind,omitempty"`
	OrganisationID string     `json:"organisation_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
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

// NormaliseHost applies the same lowercase / trim / control-char
// hygiene as NormaliseDomain but skips the TLD requirement, so a
// bare Linux hostname (e.g. `webapp-01`) round-trips. URL-like input
// (a slash, scheme, or port suffix) is rejected — the caller should
// have a hostname, not a URL.
func NormaliseHost(in string) (string, error) {
	s := strings.TrimSpace(in)
	if s == "" {
		return "", errors.New("host: empty")
	}
	s = strings.ToLower(s)
	if strings.ContainsAny(s, "/?#") || strings.Contains(s, "://") {
		return "", fmt.Errorf("host: %q contains URL syntax", in)
	}
	if ip := net.ParseIP(s); ip != nil {
		return "", fmt.Errorf("host: %q is an IP, not a host name", in)
	}
	for _, r := range s {
		if r <= 0x20 || r == 0x7f {
			return "", fmt.Errorf("host: %q contains control characters", in)
		}
	}
	return s, nil
}

// Validate checks that a Target is usable as a scan input. An empty
// Kind defaults to TargetKindDomain so existing callers (the perimeter
// scanner, the API) keep their previous behaviour.
func (t *Target) Validate() error {
	switch t.Kind {
	case "":
		t.Kind = TargetKindDomain
		fallthrough
	case TargetKindDomain:
		d, err := NormaliseDomain(t.Domain)
		if err != nil {
			return err
		}
		t.Domain = d
	case TargetKindHost:
		h, err := NormaliseHost(t.Domain)
		if err != nil {
			return err
		}
		t.Domain = h
	default:
		return fmt.Errorf("target: unknown kind %q", t.Kind)
	}
	for i, r := range t.Related {
		rd, err := NormaliseDomain(r)
		if err != nil {
			return fmt.Errorf("related[%d]: %w", i, err)
		}
		t.Related[i] = rd
	}
	return nil
}
