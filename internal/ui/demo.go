// demo.go renders the public, read-only /demo route (openspec change
// 2026-09-21-demo-pagina): the latest completed scan of one
// pre-configured target, reusing BuildAnswerVerdict for the headline
// and BuildFlowAnswers for the onderbouwing — the same building blocks
// the login-gated assessment page uses, not a second implementation
// (tasks.md 1.2 "Hergebruik de bestaande weergave; bouw geen tweede").
//
// DemoHandler is mounted directly on cmd/wanderer's root router, never
// under /ui — it carries no OrgSlug, no ReportURL, no nav, and no way
// to reach any other target's data (tasks.md 1.3), and it never
// touches a Scanner, so no request to it can start a scan (tasks.md
// 1.4).
//
// The headline also carries the x/n score (docs/tasks/2026-09-22-demo-score.md
// 1.1/1.2): composeDemoHeadline reuses BuildFleetScore — the same x/n
// the login-gated answer page shows — rather than a third count, and
// leads with it so a reader sees how much scores well before the
// ja/nee/onbekend sentence that names the heaviest open point.
//
// "Latest" means latest assessed (docs/tasks/2026-09-22-demo-zonder-beoordeling.md
// 1.1): latestCompletedScan skips a voltooide scan that has no
// beoordeling yet rather than landing on it, so a bare `POST /scans`
// on the live target can never blank out the page's last real story.
package ui

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// demoView is demo.tmpl's shape. HasScan false means "nog geen
// meting" — either no scan has completed yet, or one has but nothing
// was ever assessed.
type demoView struct {
	Domain      string
	HasScan     bool
	ScannedAt   string
	Verdict     string
	Headline    string
	FlowAnswers []AccountabilityAnswer
}

// DemoHandler renders the demo page for target. It only reads: the
// store's ListScans/GetScan/ListAssessmentsForScan, nothing that could
// enqueue work.
func DemoHandler(st *store.Store, tmpl *template.Template, target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		view := demoView{Domain: target}
		scanID, startedAt, found, err := latestCompletedScan(ctx, st, target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			render(w, tmpl, "demo.tmpl", view)
			return
		}
		scan, err := st.GetScan(ctx, scanID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assessments, err := st.ListAssessmentsForScan(ctx, scan.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		findingsByID := make(map[string]models.Finding, len(scan.Findings))
		for _, f := range scan.Findings {
			findingsByID[f.ID] = f
		}
		v := BuildAnswerVerdict(assessments, findingsByID)
		fs := BuildFleetScore(assessments, findingsByID)
		view.HasScan = true
		view.ScannedAt = startedAt.UTC().Format(time.RFC3339)
		view.Verdict = v.Verdict
		view.Headline = composeDemoHeadline(v, fs)
		view.FlowAnswers = BuildFlowAnswers(assessments, findingsByID, target)
		render(w, tmpl, "demo.tmpl", view)
	}
}

// composeDemoHeadline leads the demo page's headline with the x/n
// score (docs/tasks/2026-09-22-demo-score.md 1.2 "leidt met de score,
// niet met een kale nee"), so a reader sees in one sentence how much
// scores well before the existing ja/nee/onbekend sentence that names
// the heaviest open point (BuildAnswerVerdict's Headline, unchanged).
// fs.N is 0 exactly when v.Verdict is "onbekend" — no flow answered or
// afhankelijk — so there is no score to lead with and v.Headline is
// returned as-is.
func composeDemoHeadline(v AnswerVerdict, fs FleetScore) string {
	if fs.N == 0 {
		return v.Headline
	}
	score := fmt.Sprintf("%d/%d", fs.X, fs.N)
	if fs.Unanswered > 0 {
		score += fmt.Sprintf(" · %d onbekend", fs.Unanswered)
	}
	return score + " — " + v.Headline
}

// latestCompletedScan finds target's most recent scan that is not
// still running, did not fail outright, and has a beoordeling — a
// "voltooide scan" in tasks.md's sense: Complete (every probe
// succeeded) or Partial (some did, so there is still something real to
// show), AND assessed. POST /scans never assesses — that is the
// separate POST /scans/{id}/assessments route
// (docs/tasks/2026-09-22-demo-zonder-beoordeling.md) — so a voltooide
// scan can still have no verdict; this walks candidates newest-first
// and skips those instead of landing on one, so a scan without a
// beoordeling can never bump an older, assessed scan off the page.
// Domain matching is case-insensitive, mirroring scanStatusHandler's
// lookup.
func latestCompletedScan(ctx context.Context, st *store.Store, domain string) (scanID string, startedAt time.Time, found bool, err error) {
	scans, err := st.ListScans(ctx, store.Selectors{})
	if err != nil {
		return "", time.Time{}, false, err
	}
	var candidates []store.ScanRow
	for _, s := range scans {
		if !strings.EqualFold(s.Domain, domain) {
			continue
		}
		switch models.ScanStatus(s.Status) {
		case models.ScanStatusComplete, models.ScanStatusPartial:
		default:
			continue
		}
		candidates = append(candidates, s)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].StartedAt.After(candidates[j].StartedAt)
	})
	for _, s := range candidates {
		assessments, aerr := st.ListAssessmentsForScan(ctx, s.ID)
		if aerr != nil {
			return "", time.Time{}, false, aerr
		}
		if len(assessments) == 0 {
			continue
		}
		return s.ID, s.StartedAt, true, nil
	}
	return "", time.Time{}, false, nil
}
