// Package scanners contains the per-source code paths for the
// egress probe: walking config files, reading /proc environs,
// parsing systemd unit files. Each scanner returns Candidates that
// the egress probe then classifies and redacts.
package scanners

import "context"

// Candidate is a raw key/value pair the scanner found, with a hint
// about where it came from. Redaction has not yet happened — that
// is the egress probe's job upstream of Findings emission.
type Candidate struct {
	Source string // path or pid the value came from
	Key    string // env-var name or config key
	Value  string // raw value, possibly secret
}

// Scanner is the contract every scanner implements.
type Scanner interface {
	ID() string
	Available() (bool, string)
	Scan(ctx context.Context) ([]Candidate, error)
}
