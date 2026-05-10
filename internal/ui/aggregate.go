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

// ConcernRow is one rule that scored `afhankelijk` on at least one
// target's most recent Assessment. TargetCount counts distinct
// target IDs, not finding occurrences — see Decision 2 in the
// add-posture-dashboard design.
type ConcernRow struct {
	Framework   string
	CriteriumID string
	Description string
	Rationale   string
	TargetCount int
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
	worst := models.ScoreOnbekend
	haveAny := false
	for _, d := range dims {
		if d.Score == models.ScoreOnbekend || d.Score.Rank() == 0 {
			continue
		}
		if !haveAny || d.Score.Rank() < worst.Rank() {
			worst = d.Score
			haveAny = true
		}
	}
	if !haveAny {
		return models.ScoreOnbekend
	}
	return worst
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

// TopConcerns returns the rules whose `afhankelijk` rationales span
// the most distinct targets, sorted descending by target-count
// (ties broken by CriteriumID for stable rendering), capped at
// `maxRows`.
func TopConcerns(snaps []TargetSnapshot, ruleLookup func(framework, criteriumID string) (assessor.Rule, bool), maxRows int) []ConcernRow {
	type key struct{ fw, id string }
	targets := map[key]map[string]struct{}{}
	for _, s := range snaps {
		for fw, a := range s.Assessments {
			for _, d := range a.Dimensions {
				for _, r := range d.Rationale {
					if r.Score != models.ScoreAfhankelijk {
						continue
					}
					k := key{fw, r.CriteriumID}
					if targets[k] == nil {
						targets[k] = map[string]struct{}{}
					}
					targets[k][s.TargetID] = struct{}{}
				}
			}
		}
	}
	rows := make([]ConcernRow, 0, len(targets))
	for k, ts := range targets {
		cr := ConcernRow{
			Framework:   k.fw,
			CriteriumID: k.id,
			TargetCount: len(ts),
		}
		if ruleLookup != nil {
			if rule, ok := ruleLookup(k.fw, k.id); ok {
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
