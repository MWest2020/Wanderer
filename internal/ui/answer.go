package ui

import (
	"fmt"

	"github.com/MWest2020/wanderer/pkg/models"
)

// AnswerVerdict is the answer page's headline (proposal.md "The
// answer"): the four-value score collapsed to ja/nee/onbekend, in the
// register the accountability dimension already uses. onbekend is
// never promoted to ja (spec.md "Unknown is not a yes"); a "nee"
// always names the flow whose verdict decided it.
type AnswerVerdict struct {
	Verdict    string // "ja" | "nee" | "onbekend"
	Headline   string // the rendered Dutch sentence
	Unanswered int    // flows that fired but scored onbekend

	// DecidingFlow and DecidingVerdict are set only when Verdict is
	// "nee": the flow label and its observed verdict text — the single
	// observation that decided it, so the headline can name it.
	DecidingFlow    string
	DecidingVerdict string
}

// BuildAnswerVerdict collapses a scan's assessments into the answer
// page's headline. It reuses SovereigntyFlows for the grouping
// (hosting/mail/DNS/transit/CDN/third-parties/certificate) instead of
// re-deriving a second grouping from the raw rationales. A flow
// absent from that list — a rule that never fired because the scan
// predates the dimension — contributes neither a verdict nor an
// unanswered count: it was never asked, so it can't count as "nee"
// or as unanswered. findingsByID resolves a deciding flow's Evidence
// so the "nee" headline can name the observed fact alongside the
// Dutch verdict (SovereigntyFlows' dutchFlowVerdict); nil is fine when
// no findings are available.
func BuildAnswerVerdict(assessments []models.Assessment, findingsByID map[string]models.Finding) AnswerVerdict {
	flows := SovereigntyFlows(assessments, findingsByID)
	afhankelijk, unanswered, answered := classifyFlows(flows)
	haveAnswered := answered > 0

	switch {
	case len(afhankelijk) > 0:
		deciding := afhankelijk[0]
		return AnswerVerdict{
			Verdict:  "nee",
			Headline: renderAnswerCopy("nee", map[string]string{"flow": deciding.Label, "verdict": deciding.Verdict}),
			// Unanswered is still reported alongside a "nee" — the count
			// of unrelated questions we couldn't answer doesn't change
			// because one question already came back "nee".
			Unanswered:      unanswered,
			DecidingFlow:    deciding.Label,
			DecidingVerdict: deciding.Verdict,
		}
	case haveAnswered && unanswered == 0:
		return AnswerVerdict{Verdict: "ja", Headline: renderAnswerCopy("ja", nil)}
	case haveAnswered:
		return AnswerVerdict{
			Verdict:    "ja",
			Headline:   renderAnswerCopy("ja_unanswered", map[string]string{"clause": unansweredClause(unanswered)}),
			Unanswered: unanswered,
		}
	default:
		return AnswerVerdict{
			Verdict:    "onbekend",
			Headline:   renderAnswerCopy("onbekend", map[string]string{"clause": unansweredClause(unanswered)}),
			Unanswered: unanswered,
		}
	}
}

// unansweredClause renders the Dutch count-agreement clause for n
// flows that fired but scored onbekend. Computed here rather than in
// answer_nl.yaml because Dutch plural verb agreement ("kon" vs
// "konden") needs the count itself, not just a placeholder for it.
func unansweredClause(n int) string {
	switch {
	case n == 0:
		return "geen van de vragen kon worden beantwoord"
	case n == 1:
		return "1 vraag kon niet worden beantwoord"
	default:
		return fmt.Sprintf("%d vragen konden niet worden beantwoord", n)
	}
}
