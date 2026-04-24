package models

import "time"

// ScanStatus is the lifecycle state of a scan.
type ScanStatus string

const (
	// ScanStatusRunning means the scan is in flight.
	ScanStatusRunning ScanStatus = "running"
	// ScanStatusComplete means every probe ran to completion.
	ScanStatusComplete ScanStatus = "complete"
	// ScanStatusPartial means at least one probe failed or timed out, but
	// the scan still produced meaningful findings. Partial is a first-
	// class result, not an error.
	ScanStatusPartial ScanStatus = "partial"
	// ScanStatusFailed means the scan could not produce meaningful output
	// (e.g. the domain did not resolve).
	ScanStatusFailed ScanStatus = "failed"
)

// Valid reports whether s is one of the defined statuses.
func (s ScanStatus) Valid() bool {
	switch s {
	case ScanStatusRunning, ScanStatusComplete, ScanStatusPartial, ScanStatusFailed:
		return true
	}
	return false
}

// Scan records one invocation of the probe suite against a Target.
type Scan struct {
	ID        string     `json:"id"`
	TargetID  string     `json:"target_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Status    ScanStatus `json:"status"`
	Error     string     `json:"error,omitempty"`
	Findings  []Finding  `json:"findings,omitempty"`
}
