package models

import (
	"encoding/json"
	"errors"
	"time"
)

// Score is the verdict a rule or dimension receives. Kept to a small
// enum on purpose — finer granularity hides behind rule count, not
// behind extra score levels.
type Score string

const (
	// ScoreOnbekend means no evidence-backed verdict could be produced.
	// Either the rule did not match anything or the required attribute
	// was missing.
	ScoreOnbekend Score = "onbekend"
	// ScoreAfhankelijk is low sovereignty — the subject depends on an
	// out-of-jurisdiction or non-sovereign party.
	ScoreAfhankelijk Score = "afhankelijk"
	// ScoreVoldoende is adequate — no red flags, but not actively
	// sovereign.
	ScoreVoldoende Score = "voldoende"
	// ScoreSoeverein is the best verdict this assessor will issue.
	ScoreSoeverein Score = "soeverein"
)

// Valid reports whether s is one of the defined scores.
func (s Score) Valid() bool {
	switch s {
	case ScoreOnbekend, ScoreAfhankelijk, ScoreVoldoende, ScoreSoeverein:
		return true
	}
	return false
}

// Rank returns an ordering for aggregation: higher is better. Unknown
// returns 0 and is excluded from "worst score" aggregation.
func (s Score) Rank() int {
	switch s {
	case ScoreAfhankelijk:
		return 1
	case ScoreVoldoende:
		return 2
	case ScoreSoeverein:
		return 3
	}
	return 0
}

// Completeness describes how much of a dimension's rule set was
// actually answerable by the evidence in a scan.
type Completeness string

const (
	// CompletenessComplete means every rule for this dimension had
	// evidence to work with.
	CompletenessComplete Completeness = "complete"
	// CompletenessPartial means some rules had evidence and some did
	// not (typically because a probe relevant to that rule was not run).
	CompletenessPartial Completeness = "partial"
	// CompletenessIncomplete means no rule for this dimension could be
	// answered from the scan.
	CompletenessIncomplete Completeness = "incomplete"
)

// Valid reports whether c is one of the defined completeness values.
func (c Completeness) Valid() bool {
	switch c {
	case CompletenessComplete, CompletenessPartial, CompletenessIncomplete:
		return true
	}
	return false
}

// Rationale is one rule-level line of reasoning in an Assessment.
type Rationale struct {
	// CriteriumID is a rule identifier, usually a DICTU criterium
	// reference such as "dictu.1.1".
	CriteriumID string `json:"criterium_id"`
	// Verdict is a short human-readable sentence explaining the score.
	Verdict string `json:"verdict"`
	// Score is the rule's verdict for the subject.
	Score Score `json:"score"`
	// Evidence lists Finding.ID values that back this rationale. Empty
	// means the rule was considered but had no evidence to act on; such
	// entries feed the dimension's Completeness calculation rather than
	// its score.
	Evidence []string `json:"evidence"`
}

// DimensionScore is the aggregate verdict for one DICTU dimension.
type DimensionScore struct {
	Dimension    DimensionHint `json:"dimension"`
	Score        Score         `json:"score"`
	Completeness Completeness  `json:"completeness"`
	Rationale    []Rationale   `json:"rationale"`
}

// Assessment is the output of running a rule set against a Scan's
// Findings. It is deterministic: the same Findings produce the same
// Assessment modulo ID and CreatedAt.
type Assessment struct {
	ID         string           `json:"id,omitempty"`
	ScanID     string           `json:"scan_id"`
	Framework  string           `json:"framework"`
	CreatedAt  time.Time        `json:"created_at"`
	Dimensions []DimensionScore `json:"dimensions"`
	Report     string           `json:"report,omitempty"`
}

// Validate checks that an Assessment is well-formed enough to persist.
// It mutates nothing.
func (a *Assessment) Validate() error {
	if a.ScanID == "" {
		return errors.New("assessment: ScanID is required")
	}
	if a.Framework == "" {
		return errors.New("assessment: Framework is required")
	}
	if len(a.Dimensions) == 0 {
		return errors.New("assessment: at least one DimensionScore is required")
	}
	for i := range a.Dimensions {
		d := &a.Dimensions[i]
		if !d.Dimension.Valid() {
			return errors.New("assessment: invalid DimensionHint")
		}
		if !d.Score.Valid() {
			return errors.New("assessment: invalid Score")
		}
		if !d.Completeness.Valid() {
			return errors.New("assessment: invalid Completeness")
		}
		for j := range d.Rationale {
			r := &d.Rationale[j]
			if r.CriteriumID == "" {
				return errors.New("assessment: Rationale.CriteriumID is required")
			}
			if !r.Score.Valid() {
				return errors.New("assessment: Rationale.Score is invalid")
			}
		}
	}
	if _, err := json.Marshal(a); err != nil {
		return errors.New("assessment: not JSON-serialisable")
	}
	return nil
}
