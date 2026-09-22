package ui

import (
	"sort"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// TargetSnapshot is the dashboard's per-target input row: the
// most recent scan plus the most recent Assessment per framework.
type TargetSnapshot struct {
	TargetID    string
	Domain      string
	Kind        models.TargetKind // "domain" (perimeter) or "host" (agent)
	LastScanID  string
	LastScanAt  time.Time
	LastStatus  string
	Assessments map[string]models.Assessment
}

// Headline is the pontificaal-section data shape: what coverage
// the instance has at a glance. Populated from data the store
// already holds; no new query, no new probe.
type Headline struct {
	LastScanAt       time.Time // zero time when no scans recorded
	TotalScans       int
	PerimeterTargets int      // unique TargetIDs with Kind=domain
	AgentHostTargets int      // unique TargetIDs with Kind=host
	Frameworks       []string // sorted list of frameworks with at least one Assessment
}

// BuildHeadline derives the headline counts from the scans list
// (for total + last-scan timestamp) and the per-target snapshots
// (for kind counts and framework presence).
func BuildHeadline(snaps []TargetSnapshot, scans []store.ScanRow) Headline {
	h := Headline{TotalScans: len(scans)}
	for _, s := range scans {
		if s.StartedAt.After(h.LastScanAt) {
			h.LastScanAt = s.StartedAt
		}
	}
	frameworks := map[string]struct{}{}
	for _, sn := range snaps {
		switch sn.Kind {
		case models.TargetKindHost:
			h.AgentHostTargets++
		default: // empty kind defaults to domain in pkg/models.Target
			h.PerimeterTargets++
		}
		for fw := range sn.Assessments {
			frameworks[fw] = struct{}{}
		}
	}
	for fw := range frameworks {
		h.Frameworks = append(h.Frameworks, fw)
	}
	sort.Strings(h.Frameworks)
	return h
}

// PostureCountsByKind returns the same shape as PostureCounts but
// limited to snapshots whose Kind matches `kind`. Empty result
// when no snapshot of that kind exists — let the renderer pick
// the empty-state copy.
func PostureCountsByKind(snaps []TargetSnapshot, kind models.TargetKind) PostureSummary {
	filtered := make([]TargetSnapshot, 0, len(snaps))
	for _, s := range snaps {
		// Empty kind defaults to domain to match pkg/models.Target.
		actual := s.Kind
		if actual == "" {
			actual = models.TargetKindDomain
		}
		if actual == kind {
			filtered = append(filtered, s)
		}
	}
	return PostureCounts(filtered)
}

// PostureSummary is per-framework counts of targets bucketed by
// their worst-dimension score. The outer key is the framework
// name (e.g. "dictu", "eucsf"); the inner map's keys are
// `models.Score` values.
type PostureSummary map[string]map[models.Score]int

// ConcernRow is one concern that scored `afhankelijk` on at least one
// target's most recent Assessment. TargetCount counts distinct target
// IDs, not finding occurrences — see Decision 2 in the
// add-posture-dashboard design. Total is how many distinct targets the
// underlying rule(s) actually evaluated (any score), so a reader can
// read TargetCount/Total as "faalt op X van Y domeinen" (run 03 task
// 2.4) rather than a bare count with no denominator. Framework and
// CriteriumID identify the primary rule — the one the reporting link
// points to and Description/Rationale come from; Frameworks lists
// every framework that surfaced this same concern (run 03 task 2.5:
// two frameworks flagging the same certificate-issuer jurisdiction
// gap must read as one finding, not two).
type ConcernRow struct {
	Framework   string
	CriteriumID string
	Description string
	Rationale   string
	TargetCount int
	Total       int
	Frameworks  []string
}

// concernRuleRef is one (framework, criteriumID) pair contributing to
// a ConcernRow.
type concernRuleRef struct{ framework, criteriumID string }

// concernTopics merges (framework, criteriumID) pairs that are the
// same real-world concern surfaced separately per framework. Today
// that is only the TLS-issuer-jurisdiction check: wand's
// cert_issuer_eea and eucsf's cert_issuer_eu both flag a non-EEA/EU
// certificate authority. Absent from this map, a rule merges with
// nothing — it keeps its own row, keyed by its own (framework, id).
var concernTopics = map[concernRuleRef]string{
	{"wand", "wand.juridisch.cert_issuer_eea"}: "cert_issuer_jurisdiction",
	{"eucsf", "eucsf.sov2.cert_issuer_eu"}:      "cert_issuer_jurisdiction",
}

// concernTopic returns the merge key for ref: the shared topic from
// concernTopics when one is declared, otherwise a key unique to this
// one rule (unchanged behaviour for every rule not in that table).
func concernTopic(ref concernRuleRef) string {
	if topic, ok := concernTopics[ref]; ok {
		return topic
	}
	return ref.framework + "|" + ref.criteriumID
}

// ActivityRow is one scan in the dashboard's recent-activity feed.
type ActivityRow struct {
	ScanID        string
	Domain        string
	StartedAt     time.Time
	Status        string
	HasAssessment bool
}

// WorstScore returns the worst score across the given dimensions,
// ignoring `onbekend` dimensions. If every dimension is `onbekend`
// (or the slice is empty) the result is `onbekend` — a target we
// cannot evaluate is unknown, not "worst".
func WorstScore(dims []models.DimensionScore) models.Score {
	score, _ := WorstScoreCovering(dims)
	return score
}

// WorstScoreCovering returns the same worst score as WorstScore, plus
// the sorted list of dimension names that contributed to it — every
// dimension with a rated (non-onbekend) score. A dimension marked
// NotApplicable always scores onbekend (engine.go) and is excluded
// the same way; a dimension absent from `dims` entirely (a scan that
// predates it) is excluded by simply never being iterated. Callers
// display the covered list alongside the score so "afhankelijk"
// never reads as "afhankelijk across the whole pack" when only one
// dimension actually fired.
func WorstScoreCovering(dims []models.DimensionScore) (models.Score, []string) {
	worst := models.ScoreOnbekend
	haveAny := false
	var covered []string
	for _, d := range dims {
		if d.Score == models.ScoreOnbekend || d.Score.Rank() == 0 {
			continue
		}
		covered = append(covered, string(d.Dimension))
		if !haveAny || d.Score.Rank() < worst.Rank() {
			worst = d.Score
			haveAny = true
		}
	}
	if !haveAny {
		return models.ScoreOnbekend, nil
	}
	sort.Strings(covered)
	return worst, covered
}

// AccountabilityPillView is the Overview's per-target accountability
// indicator (design.md "Overview surfaces accountability without
// adding a tab"): coloured like the dimension's own score, plus two
// states outside the four-value scale — "n.v.t." when every rule is
// structural, and "niet beoordeeld" when the Assessment predates the
// dimension entirely (no accountability entry in dims at all).
type AccountabilityPillView struct {
	Label string // score word, "n.v.t.", or "niet beoordeeld"
	Class string // CSS suffix: "soeverein"/"voldoende"/"afhankelijk"/"onbekend", "nvt", or "unassessed"
}

// scannerReasonWarnings maps a scanner-subject reason code to the
// operator-facing warning shown next to the dimension it affects
// (design.md "UI direction": a scanner limitation is never a property
// of the target). One entry today; a future scanner-subject reason
// code needs an entry here or it renders with no banner.
var scannerReasonWarnings = map[string]string{
	assessor.ReasonScannerNoIPv6: "scanner heeft geen IPv6 — v6-paden niet gemeten",
}

// DimensionScannerWarnings returns the distinct operator-environment
// warnings for dim — one per scanner-subject reason code present
// across its Rationale, sorted for stable rendering.
func DimensionScannerWarnings(dim models.DimensionScore) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range dim.Rationale {
		if r.Reason == "" {
			continue
		}
		_, subject := assessor.ReasonInfo(r.Reason)
		if subject != assessor.ReasonSubjectScanner {
			continue
		}
		msg, ok := scannerReasonWarnings[r.Reason]
		if !ok || seen[msg] {
			continue
		}
		seen[msg] = true
		out = append(out, msg)
	}
	sort.Strings(out)
	return out
}

// AccountabilityPill derives the pill for one target from its
// Assessment's dimensions. Pass nil (or an Assessment without the
// dimension) for a scan that predates the accountability change.
func AccountabilityPill(dims []models.DimensionScore) AccountabilityPillView {
	for _, d := range dims {
		if d.Dimension != models.DimensionAccountability {
			continue
		}
		if d.NotApplicable {
			return AccountabilityPillView{Label: "n.v.t.", Class: "nvt"}
		}
		return AccountabilityPillView{Label: string(d.Score), Class: string(d.Score)}
	}
	return AccountabilityPillView{Label: "niet beoordeeld", Class: "unassessed"}
}

// PostureCounts buckets each target's worst-dimension score by
// framework. Targets without any persisted Assessment for a given
// framework do not contribute to that framework's counts — they
// are simply absent rather than counted as `onbekend`.
func PostureCounts(snaps []TargetSnapshot) PostureSummary {
	out := PostureSummary{}
	for _, s := range snaps {
		for fw, a := range s.Assessments {
			score := WorstScore(a.Dimensions)
			if out[fw] == nil {
				out[fw] = map[models.Score]int{}
			}
			out[fw][score]++
		}
	}
	return out
}

// TopConcerns returns the concerns whose `afhankelijk` rationales span
// the most distinct targets, sorted descending by target-count (ties
// broken by CriteriumID for stable rendering), capped at `maxRows`. A
// concern spanning two frameworks (concernTopics) counts each target
// once even if both frameworks flagged it, and reports the union of
// frameworks that fired — never as two separate rows for the same
// underlying gap.
func TopConcerns(snaps []TargetSnapshot, ruleLookup func(framework, criteriumID string) (assessor.Rule, bool), maxRows int) []ConcernRow {
	afhankelijkTargets := map[string]map[string]struct{}{}
	// allTargets tracks every target the rule fired against, any
	// score — the denominator for "faalt op X van Y domeinen". It is
	// a separate tally from afhankelijkTargets; nothing here changes
	// which targets count as afhankelijk.
	allTargets := map[string]map[string]struct{}{}
	rulesByTopic := map[string][]concernRuleRef{}
	seenRule := map[string]map[concernRuleRef]bool{}
	for _, s := range snaps {
		for fw, a := range s.Assessments {
			for _, d := range a.Dimensions {
				for _, r := range d.Rationale {
					ref := concernRuleRef{fw, r.CriteriumID}
					topic := concernTopic(ref)
					if allTargets[topic] == nil {
						allTargets[topic] = map[string]struct{}{}
					}
					allTargets[topic][s.TargetID] = struct{}{}
					if seenRule[topic] == nil {
						seenRule[topic] = map[concernRuleRef]bool{}
					}
					if !seenRule[topic][ref] {
						seenRule[topic][ref] = true
						rulesByTopic[topic] = append(rulesByTopic[topic], ref)
					}
					if r.Score != models.ScoreAfhankelijk {
						continue
					}
					if afhankelijkTargets[topic] == nil {
						afhankelijkTargets[topic] = map[string]struct{}{}
					}
					afhankelijkTargets[topic][s.TargetID] = struct{}{}
				}
			}
		}
	}
	rows := make([]ConcernRow, 0, len(afhankelijkTargets))
	for topic, ts := range afhankelijkTargets {
		refs := rulesByTopic[topic]
		sort.Slice(refs, func(i, j int) bool {
			if reportingFrameworkRank(refs[i].framework) != reportingFrameworkRank(refs[j].framework) {
				return reportingFrameworkRank(refs[i].framework) < reportingFrameworkRank(refs[j].framework)
			}
			return refs[i].framework < refs[j].framework
		})
		primary := refs[0]
		frameworks := make([]string, 0, len(refs))
		for _, ref := range refs {
			frameworks = append(frameworks, ref.framework)
		}
		cr := ConcernRow{
			Framework:   primary.framework,
			CriteriumID: primary.criteriumID,
			TargetCount: len(ts),
			Total:       len(allTargets[topic]),
			Frameworks:  frameworks,
		}
		if ruleLookup != nil {
			if rule, ok := ruleLookup(primary.framework, primary.criteriumID); ok {
				cr.Description = rule.Description
				cr.Rationale = rule.Rationale
			}
		}
		rows = append(rows, cr)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TargetCount != rows[j].TargetCount {
			return rows[i].TargetCount > rows[j].TargetCount
		}
		return rows[i].CriteriumID < rows[j].CriteriumID
	})
	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return rows
}

// RecentActivity returns the most-recent `maxRows` scans across the
// estate, newest-first by StartedAt. The hasAssessment function
// reports whether a given scan has any persisted Assessment so the
// template can link to the Analysis page when one exists.
func RecentActivity(scans []store.ScanRow, hasAssessment func(scanID string) bool, maxRows int) []ActivityRow {
	all := append([]store.ScanRow(nil), scans...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].StartedAt.After(all[j].StartedAt)
	})
	if maxRows > 0 && len(all) > maxRows {
		all = all[:maxRows]
	}
	out := make([]ActivityRow, 0, len(all))
	for _, s := range all {
		row := ActivityRow{
			ScanID:    s.ID,
			Domain:    s.Domain,
			StartedAt: s.StartedAt,
			Status:    s.Status,
		}
		if hasAssessment != nil {
			row.HasAssessment = hasAssessment(s.ID)
		}
		out = append(out, row)
	}
	return out
}

// FrameworkVerdict is the Dashboard's "is dit goed of niet"
// signal per framework: the worst score reached across all
// targets in scope, plus how many targets are at that worst
// score. When zero targets have been assessed under the
// framework, Score is empty and TargetsAtWorst is 0.
type FrameworkVerdict struct {
	Framework      string
	Score          models.Score
	TargetsAtWorst int
	TotalAssessed  int
}

// WorstByFramework computes one FrameworkVerdict per framework
// present in the snapshots. Worst score uses Rank — the lowest
// non-onbekend rank wins. If every assessed target is onbekend,
// the verdict is onbekend with TargetsAtWorst = TotalAssessed.
func WorstByFramework(snaps []TargetSnapshot) []FrameworkVerdict {
	type bucket struct {
		worst   models.Score
		atWorst int
		total   int
	}
	buckets := map[string]*bucket{}
	for _, s := range snaps {
		for fw, a := range s.Assessments {
			b := buckets[fw]
			if b == nil {
				b = &bucket{worst: models.ScoreOnbekend}
				buckets[fw] = b
			}
			b.total++
			score := WorstScore(a.Dimensions)
			// Score with the lowest Rank (excluding 0 / onbekend
			// which Rank() returns for unknown) is the worst.
			if score.Rank() > 0 {
				if b.worst == models.ScoreOnbekend || score.Rank() < b.worst.Rank() {
					b.worst = score
					b.atWorst = 1
				} else if score == b.worst {
					b.atWorst++
				}
			} else if b.worst == models.ScoreOnbekend {
				b.atWorst++
			}
		}
	}
	out := make([]FrameworkVerdict, 0, len(buckets))
	for fw, b := range buckets {
		out = append(out, FrameworkVerdict{
			Framework:      fw,
			Score:          b.worst,
			TargetsAtWorst: b.atWorst,
			TotalAssessed:  b.total,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		// wand first, then eucsf, then alphabetical.
		rank := func(fw string) int {
			switch fw {
			case "wand":
				return 0
			case "eucsf":
				return 1
			default:
				return 2
			}
		}
		if rank(out[i].Framework) != rank(out[j].Framework) {
			return rank(out[i].Framework) < rank(out[j].Framework)
		}
		return out[i].Framework < out[j].Framework
	})
	return out
}

// AllScores is the canonical iteration order for posture-summary
// rendering: best to worst, with onbekend at the end so an
// unevaluated bucket reads as "we don't know yet" rather than
// "as bad as afhankelijk".
var AllScores = []models.Score{
	models.ScoreSoeverein,
	models.ScoreVoldoende,
	models.ScoreAfhankelijk,
	models.ScoreOnbekend,
}

// RuleSummaryRow is one row on /ui/reporting — a rule that has
// fired across the persisted Assessments, with distinct-target
// counts per score. A rule firing twice on the same target
// counts once (the dashboard's TopConcerns convention applied
// across every score, not just `afhankelijk`).
type RuleSummaryRow struct {
	Framework   string
	CriteriumID string
	Description string
	Counts      map[models.Score]int
}

// RuleTargetRow is one row on /ui/reporting/{framework}/{ruleID}
// — the per-rule deep dive. Each row links the operator back to
// the originating scan's assessment page.
type RuleTargetRow struct {
	TargetID string
	Domain   string
	ScanID   string
	Score    models.Score
	Verdict  string
	When     time.Time
}

// WorstScoreFromCounts returns the worst score with at least one
// target in `counts`, plus how many targets sit at that score.
// "Worst" follows Score.Rank() — lower rank wins, with onbekend
// (rank 0) treated as a fallback when no rated score is present.
func WorstScoreFromCounts(counts map[models.Score]int) (worst models.Score, atWorst int, total int) {
	worst = models.ScoreOnbekend
	for sc, n := range counts {
		total += n
		if n == 0 {
			continue
		}
		if sc.Rank() == 0 {
			continue
		}
		if worst == models.ScoreOnbekend || sc.Rank() < worst.Rank() {
			worst = sc
			atWorst = n
		} else if sc == worst {
			atWorst += n
		}
	}
	if worst == models.ScoreOnbekend {
		// No rated score reached; report the onbekend bucket itself.
		atWorst = counts[models.ScoreOnbekend]
	}
	return worst, atWorst, total
}

// reportingFrameworkRank gives wand priority over eucsf and
// alphabetical for any new pack. Used by RuleSummary's stable
// row order.
func reportingFrameworkRank(fw string) int {
	switch fw {
	case "wand":
		return 0
	case "eucsf":
		return 1
	default:
		return 2
	}
}

// RuleSummary returns one row per rule that has fired across the
// snapshots. ruleLookup attaches Description from the registry
// when the rule is registered (it always is for current frameworks
// but the helper is nil-safe for future packs the UI does not
// know about yet).
func RuleSummary(snaps []TargetSnapshot, ruleLookup func(framework, criteriumID string) (assessor.Rule, bool)) []RuleSummaryRow {
	type key struct{ fw, id string }
	// Per (fw, id, score) collect the set of distinct target IDs.
	// Sets, not bags — same convention as TopConcerns.
	type bucket struct {
		perScore map[models.Score]map[string]struct{}
	}
	buckets := map[key]*bucket{}
	for _, s := range snaps {
		for fw, a := range s.Assessments {
			for _, d := range a.Dimensions {
				for _, r := range d.Rationale {
					k := key{fw, r.CriteriumID}
					b := buckets[k]
					if b == nil {
						b = &bucket{perScore: map[models.Score]map[string]struct{}{}}
						buckets[k] = b
					}
					if b.perScore[r.Score] == nil {
						b.perScore[r.Score] = map[string]struct{}{}
					}
					b.perScore[r.Score][s.TargetID] = struct{}{}
				}
			}
		}
	}
	rows := make([]RuleSummaryRow, 0, len(buckets))
	for k, b := range buckets {
		row := RuleSummaryRow{
			Framework:   k.fw,
			CriteriumID: k.id,
			Counts:      map[models.Score]int{},
		}
		for sc, targetIDs := range b.perScore {
			row.Counts[sc] = len(targetIDs)
		}
		if ruleLookup != nil {
			if rule, ok := ruleLookup(k.fw, k.id); ok {
				row.Description = rule.Description
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if reportingFrameworkRank(rows[i].Framework) != reportingFrameworkRank(rows[j].Framework) {
			return reportingFrameworkRank(rows[i].Framework) < reportingFrameworkRank(rows[j].Framework)
		}
		if rows[i].Framework != rows[j].Framework {
			return rows[i].Framework < rows[j].Framework
		}
		return rows[i].CriteriumID < rows[j].CriteriumID
	})
	return rows
}

// RuleTargetRows returns the per-target rows for one rule on the
// detail page. A target with no Rationale for the rule is absent
// from the result. Newest-Assessment-wins is implicit: the
// snapshot's Assessments map already holds only the most recent
// Assessment per framework (built that way in dashboardHandler).
//
// Rows are ordered by score severity — afhankelijk first so the
// operator sees pain points immediately, then voldoende,
// soeverein, onbekend. Within a score bucket, alphabetical by
// domain.
func RuleTargetRows(snaps []TargetSnapshot, framework, ruleID string) []RuleTargetRow {
	out := make([]RuleTargetRow, 0)
	for _, s := range snaps {
		a, ok := s.Assessments[framework]
		if !ok {
			continue
		}
		for _, d := range a.Dimensions {
			for _, r := range d.Rationale {
				if r.CriteriumID != ruleID {
					continue
				}
				out = append(out, RuleTargetRow{
					TargetID: s.TargetID,
					Domain:   s.Domain,
					ScanID:   s.LastScanID,
					Score:    r.Score,
					Verdict:  r.Verdict,
					When:     s.LastScanAt,
				})
			}
		}
	}
	severityRank := map[models.Score]int{
		models.ScoreAfhankelijk: 0,
		models.ScoreVoldoende:   1,
		models.ScoreSoeverein:   2,
		models.ScoreOnbekend:    3,
	}
	sort.Slice(out, func(i, j int) bool {
		if severityRank[out[i].Score] != severityRank[out[j].Score] {
			return severityRank[out[i].Score] < severityRank[out[j].Score]
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}
