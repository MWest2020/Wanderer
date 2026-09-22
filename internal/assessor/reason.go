package assessor

// ReasonClass classifies how a Rationale's reason code affects
// dimension aggregation. The mechanism is generic — shared by every
// pack and dimension, not specific to any one of them.
type ReasonClass string

const (
	// ReasonStructural means the rule does not apply to this target
	// (renders "n.v.t."). Excluded from both the worst-score
	// computation and the completeness denominator.
	ReasonStructural ReasonClass = "structural"
	// ReasonGap means a measurement hole (renders "onbekend"). Counts
	// as a missing observation, same as an evidence-less rule today.
	ReasonGap ReasonClass = "gap"
)

// ReasonSubject names who a reason code's verdict is actually about.
// A "scanner" subject is a limitation of Wanderer's own environment
// and must never render as a property of the target.
type ReasonSubject string

const (
	// ReasonSubjectTarget is the default: the verdict says something
	// about the scanned domain.
	ReasonSubjectTarget ReasonSubject = "target"
	// ReasonSubjectScanner means the verdict says something about
	// Wanderer's own environment, not the target.
	ReasonSubjectScanner ReasonSubject = "scanner"
)

// Reason codes seeded by the accountability dimension. Codes are
// strings (not a Go type) because they are persisted as-is in
// models.Rationale.Reason; the constants exist so rules reference
// them without retyping the string.
const (
	ReasonRegistryRedacted       = "registry_redacted"
	ReasonNotPublishedByRegistry = "not_published_by_registry"
	ReasonScannerNoIPv6          = "scanner_no_ipv6"
	ReasonProbeUnavailable       = "probe_unavailable"
	// ReasonScannerUnreadableOutput means an inspector's own output
	// could not be parsed (malformed / truncated / prefixed with
	// noise) — a hole in our measurement, not a statement about the
	// target. Seeded by the Nextcloud inspector
	// (internal/probe/inventory/nextcloud/nextcloud.go) but generic
	// across inspectors.
	ReasonScannerUnreadableOutput = "scanner_unreadable_output"
	// ReasonNotMeasured means no relevant external measurement was
	// ever performed (no import, or the measurement's own tests came
	// back not_tested/error) — seeded by the standards dimension
	// (internal/assessor/wand/standards_rules.go), task 3.1.
	ReasonNotMeasured = "not_measured"
	// ReasonMeasurementStale means a relevant measurement exists but
	// is older than the rule's configured staleness boundary —
	// seeded by the standards dimension, task 3.1/3.2.
	ReasonMeasurementStale = "measurement_stale"
)

type reasonInfo struct {
	class   ReasonClass
	subject ReasonSubject
}

// reasonRegistry is the single source of truth for every reason code
// a Rule may emit, across every dimension and pack. A code absent
// here is a bug in the rule that emitted it — see ReasonInfo.
var reasonRegistry = map[string]reasonInfo{
	ReasonRegistryRedacted:        {class: ReasonStructural, subject: ReasonSubjectTarget},
	ReasonNotPublishedByRegistry:  {class: ReasonStructural, subject: ReasonSubjectTarget},
	ReasonScannerNoIPv6:           {class: ReasonStructural, subject: ReasonSubjectScanner},
	ReasonProbeUnavailable:        {class: ReasonGap, subject: ReasonSubjectTarget},
	ReasonScannerUnreadableOutput: {class: ReasonGap, subject: ReasonSubjectScanner},
	ReasonNotMeasured:             {class: ReasonGap, subject: ReasonSubjectTarget},
	ReasonMeasurementStale:        {class: ReasonGap, subject: ReasonSubjectTarget},
}

// ReasonInfo looks up the class and subject registered for code. It
// panics when code is not registered: a rule emitting an
// unregistered reason code is a programming error that must fail
// loudly (and fail tests), not silently default to some class.
func ReasonInfo(code string) (class ReasonClass, subject ReasonSubject) {
	info, ok := reasonRegistry[code]
	if !ok {
		panic("assessor: unregistered reason code " + code)
	}
	return info.class, info.subject
}
