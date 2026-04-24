// Package models defines the shared data types exchanged between probes,
// the scanner, the store, and the API. These types are the contract: a
// probe must not return anything richer than a Finding, and the assessor
// (future work) must not reach past the Finding into probe packages.
package models

import (
	"encoding/json"
	"errors"
	"time"
)

// Severity classifies what a Finding says about the target. These values
// are deliberately coarse. Fine-grained scoring belongs to the assessor,
// not the probe.
type Severity string

const (
	// SeverityInfo is a neutral fact. Not a problem, not praise.
	SeverityInfo Severity = "info"
	// SeverityObservation is a noteworthy fact that may matter in context.
	SeverityObservation Severity = "observation"
	// SeverityConcern is a fact that likely bears on sovereignty posture.
	SeverityConcern Severity = "concern"
	// SeverityFinding is a fact that the assessor should almost certainly
	// reflect in the final score.
	SeverityFinding Severity = "finding"
)

// Valid reports whether s is one of the defined severities.
func (s Severity) Valid() bool {
	switch s {
	case SeverityInfo, SeverityObservation, SeverityConcern, SeverityFinding:
		return true
	}
	return false
}

// DimensionHint names a DICTU Toetsingsinstrument dimension that this
// Finding informs. An empty value means the Finding is a raw observation
// with no direct dimensional meaning on its own.
type DimensionHint string

const (
	DimensionNone         DimensionHint = ""
	DimensionJuridisch    DimensionHint = "juridisch"
	DimensionTechnologie  DimensionHint = "technologie"
	DimensionDataAI       DimensionHint = "data_ai"
	DimensionOperationeel DimensionHint = "operationeel"
	DimensionMens         DimensionHint = "mens"
)

// Valid reports whether d is one of the defined dimensions (the empty
// value is valid — it means "no hint").
func (d DimensionHint) Valid() bool {
	switch d {
	case DimensionNone,
		DimensionJuridisch,
		DimensionTechnologie,
		DimensionDataAI,
		DimensionOperationeel,
		DimensionMens:
		return true
	}
	return false
}

// Finding is the single output type every probe produces. The scanner
// collects these; the store serialises them; the assessor (future) reads
// them. No probe-specific structure is allowed past this boundary:
// probe-specific data goes into Attributes.
type Finding struct {
	// ID is a ULID assigned when the Finding is persisted. Empty before
	// persistence.
	ID string `json:"id,omitempty"`

	// ScanID is the parent scan's ID. Populated by the scanner.
	ScanID string `json:"scan_id,omitempty"`

	// ProbeID identifies which probe produced this Finding. Stable string,
	// e.g. "dns.mx", "tls.issuer", "http.third_party".
	ProbeID string `json:"probe_id"`

	// DimensionHint is the DICTU dimension this Finding informs, if any.
	DimensionHint DimensionHint `json:"dimension_hint,omitempty"`

	// CriteriumHint is an optional DICTU criterium reference
	// (e.g. "1.2" or a short slug).
	CriteriumHint string `json:"criterium_hint,omitempty"`

	// Subject is the entity being described — a domain, an IP, a host.
	Subject string `json:"subject"`

	// Severity is the coarse classification.
	Severity Severity `json:"severity"`

	// Attributes holds structured probe-specific data. Must be
	// JSON-serialisable.
	Attributes map[string]any `json:"attributes"`

	// Evidence is raw source material so downstream consumers can audit
	// without re-scanning (e.g. certificate PEM, verbatim TXT record).
	Evidence []byte `json:"evidence,omitempty"`

	// CreatedAt is when the Finding was persisted.
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Validate checks that a Finding is well-formed enough to persist.
func (f *Finding) Validate() error {
	if f.ProbeID == "" {
		return errors.New("finding: ProbeID is required")
	}
	if f.Subject == "" {
		return errors.New("finding: Subject is required")
	}
	if !f.Severity.Valid() {
		return errors.New("finding: Severity is invalid")
	}
	if !f.DimensionHint.Valid() {
		return errors.New("finding: DimensionHint is invalid")
	}
	if f.Attributes == nil {
		f.Attributes = map[string]any{}
	}
	if _, err := json.Marshal(f.Attributes); err != nil {
		return errors.New("finding: Attributes not JSON-serialisable")
	}
	return nil
}
