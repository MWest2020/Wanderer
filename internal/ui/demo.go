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
package ui

import (
	"context"
	"html/template"
	"net/http"
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
		if len(assessments) == 0 {
			// A completed scan nobody ever ran `wanderer assess` against
			// has nothing to answer yet either — the same "nog geen
			// meting" state as no scan at all, not a false onbekend.
			render(w, tmpl, "demo.tmpl", view)
			return
		}
		findingsByID := make(map[string]models.Finding, len(scan.Findings))
		for _, f := range scan.Findings {
			findingsByID[f.ID] = f
		}
		v := BuildAnswerVerdict(assessments, findingsByID)
		view.HasScan = true
		view.ScannedAt = startedAt.UTC().Format(time.RFC3339)
		view.Verdict = v.Verdict
		view.Headline = v.Headline
		view.FlowAnswers = BuildFlowAnswers(assessments, findingsByID, target)
		render(w, tmpl, "demo.tmpl", view)
	}
}

// latestCompletedScan finds target's most recent scan that is not
// still running and did not fail outright — a "voltooide scan" in
// tasks.md's sense: Complete (every probe succeeded) or Partial (some
// did, so there is still something real to show). Domain matching is
// case-insensitive, mirroring scanStatusHandler's lookup.
func latestCompletedScan(ctx context.Context, st *store.Store, domain string) (scanID string, startedAt time.Time, found bool, err error) {
	scans, err := st.ListScans(ctx, store.Selectors{})
	if err != nil {
		return "", time.Time{}, false, err
	}
	for _, s := range scans {
		if !strings.EqualFold(s.Domain, domain) {
			continue
		}
		switch models.ScanStatus(s.Status) {
		case models.ScanStatusComplete, models.ScanStatusPartial:
		default:
			continue
		}
		if !found || s.StartedAt.After(startedAt) {
			scanID, startedAt, found = s.ID, s.StartedAt, true
		}
	}
	return scanID, startedAt, found, nil
}
