package models

// SealLevel is the EU Cloud Sovereignty Framework (SEAL) score
// scale: 0 (unknown / no evidence) up to 4 (sovereign). It exists
// alongside the existing Score type so the SEAL rule pack can
// emit framework-native verdicts; the dimensional aggregation in
// the assessor still happens via Score, which a small mapping
// function (LevelToScore below) translates back to.
type SealLevel string

const (
	// SEAL0 means no evidence-backed verdict could be produced.
	SEAL0 SealLevel = "seal_0"
	// SEAL1 means a verdict that fails the framework outright.
	SEAL1 SealLevel = "seal_1"
	// SEAL2 means a verdict with notable dependence on a non-EU party.
	SEAL2 SealLevel = "seal_2"
	// SEAL3 means a verdict that is adequate.
	SEAL3 SealLevel = "seal_3"
	// SEAL4 means full sovereignty.
	SEAL4 SealLevel = "seal_4"
)

// Valid reports whether l is one of the defined SEAL levels.
func (l SealLevel) Valid() bool {
	switch l {
	case SEAL0, SEAL1, SEAL2, SEAL3, SEAL4:
		return true
	}
	return false
}

// LevelToScore maps a SealLevel onto the existing Score enum so the
// SEAL rule pack plugs into the same per-dimension aggregation the
// assessor already does.
func (l SealLevel) ToScore() Score {
	switch l {
	case SEAL4:
		return ScoreSoeverein
	case SEAL3:
		return ScoreVoldoende
	case SEAL2, SEAL1:
		return ScoreAfhankelijk
	}
	return ScoreOnbekend
}

// Framework names a rule pack. The Assessment.Framework field stays
// a free-form string for forward compatibility, but the values used
// in code SHOULD come from this enum.
type Framework string

const (
	// FrameworkWand is Conduction's first-party rule pack, formerly
	// known as DICTU. The DICTU Toetsingsinstrument Soevereiniteit
	// Clouddiensten inspired the rule semantics; the implementation
	// and the `wand` (Wanderer-NL) name are Conduction's. See ADR-0011.
	FrameworkWand  Framework = "wand"
	FrameworkEUCSF Framework = "eucsf"
)

// Valid reports whether f is one of the defined frameworks.
func (f Framework) Valid() bool {
	switch f {
	case FrameworkWand, FrameworkEUCSF:
		return true
	}
	return false
}
