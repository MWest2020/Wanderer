// Package assessor turns a set of Findings into an Assessment: per-DICTU
// dimension scores, completeness flags, and per-rule rationale. The
// package is a pure consumer of models.Finding — it does not probe, log
// minimally, and is deterministic: the same Findings produce the same
// Assessment modulo ID and CreatedAt.
package assessor

import "github.com/MWest2020/wanderer/pkg/models"

// Rule is a single DICTU / SEAL criterium expressed as a Go function.
// Rules live in their own sub-package (e.g. internal/assessor/dictu)
// and are passed into Assess as a slice.
type Rule struct {
	// ID is the stable identifier, e.g. "dictu.juridisch.cert_issuer_eu".
	// It is recorded on every emitted models.Rationale as
	// Rationale.CriteriumID so a reader can trace a verdict back to
	// the rule that produced it.
	ID string
	// Dimension is the DICTU / SEAL dimension this rule contributes to.
	Dimension models.DimensionHint
	// Description is a one-line human-readable summary shown in the
	// markdown report and the UI's analysis card header.
	Description string
	// Rationale is a paragraph (1–4 sentences) of plain-language
	// context surfaced in the UI's analysis page: what the rule
	// observes, why it matters for sovereignty posture, and the
	// shape of the consequence when it fires `afhankelijk`. It is
	// the "why this matters" answer for a non-technical reader.
	// Required: every rule registered with DefaultRules() MUST set
	// a non-empty Rationale; the corresponding rule pack's
	// TestEveryRuleHasRationale fails the build otherwise.
	Rationale string
	// Match is the rule body. It receives every Finding in the scan
	// and returns a RuleResult. Rules SHOULD be total: defensively
	// handle missing or mistyped attributes by returning
	// ScoreOnbekend with an empty Evidence list, not by panicking.
	Match func(findings []models.Finding) RuleResult
}

// RuleResult is the outcome of running a single Rule. A result with a
// non-empty Evidence list is "evidence-backed" and contributes to the
// dimension score; a result with empty Evidence contributes only to
// the dimension's Completeness calculation.
type RuleResult struct {
	Score    models.Score
	Verdict  string
	Evidence []string
}

// IsEvidenceLike reports whether a Finding looks like positive evidence
// rather than a meta-finding (lookup error, explicit no-answer, probe
// unavailability). Rules that count or aggregate Findings by ProbeID
// SHOULD filter through this helper before treating a Finding as
// evidence; otherwise a non-resolvable domain or a missing probe can
// produce a positive verdict from rows that carry no real signal.
//
// A Finding is treated as meta when any of these attributes is set:
//   - "error"      (any non-empty value — convention is the error message)
//   - "no_answer"  (true)
//   - "unavailable" (true)
func IsEvidenceLike(f models.Finding) bool {
	if f.Attributes == nil {
		return true
	}
	if v, ok := f.Attributes["error"]; ok {
		if s, isStr := v.(string); !isStr || s != "" {
			return false
		}
	}
	if v, ok := f.Attributes["no_answer"].(bool); ok && v {
		return false
	}
	if v, ok := f.Attributes["unavailable"].(bool); ok && v {
		return false
	}
	return true
}
