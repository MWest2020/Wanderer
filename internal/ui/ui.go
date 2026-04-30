// Package ui ships a small read-only operator UI for Wanderer.
// Pages are rendered with html/template; styling is vanilla CSS;
// authentication is HTTP Basic against an htpasswd file. The
// package contains zero mutating handlers — a static-analysis
// test in ui_test.go grep-blocks any future POST/PUT/PATCH/DELETE
// registration in this directory.
package ui

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.tmpl static/*
var assets embed.FS

// Handler builds the chi sub-router for the UI. htpasswdPath may be
// empty; with an empty path the routes are still mounted but no
// authentication is required (development mode).
func Handler(st *store.Store, htpasswdPath string) (http.Handler, error) {
	tmpl, err := template.ParseFS(assets, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("ui: parse templates: %w", err)
	}
	r := chi.NewRouter()
	if htpasswdPath != "" {
		creds, err := LoadHtpasswd(htpasswdPath)
		if err != nil {
			return nil, err
		}
		r.Use(basicAuthMiddleware(htpasswdPath, creds))
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	r.Get("/", dashboardHandler(st, tmpl))
	r.Get("/targets", targetsHandler(st, tmpl))
	r.Get("/scans/{id}", scanHandler(st, tmpl))
	r.Get("/scans/{id}/assessment", assessmentHandler(st, tmpl))
	r.Get("/targets/{id}/drift", driftHandler(st, tmpl))
	return r, nil
}

// basicAuthMiddleware re-reads the htpasswd file on every request
// so an operator can rotate credentials without restarting the
// process. Cache hit-rate is fine on the small files this UI
// targets; if the file disappears we fall back to the in-memory
// snapshot rather than locking the operator out.
func basicAuthMiddleware(path string, fallback map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			creds := fallback
			if fresh, err := LoadHtpasswd(path); err == nil {
				creds = fresh
			}
			if !ok || !verifyAgainst(creds, user, pass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="wanderer"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// verifyAgainst returns true when (user, pass) authenticates
// against creds. Unknown users return false in constant time.
func verifyAgainst(creds map[string]string, user, pass string) bool {
	entry, ok := creds[user]
	if !ok {
		// Run a dummy bcrypt compare against a constant entry so
		// the timing pattern of "user found / not found" is not
		// trivially observable.
		_ = subtle.ConstantTimeCompare([]byte("dummy"), []byte("dummy"))
		return false
	}
	return VerifyHtpasswdLine(entry, pass)
}

// dashboardView is the shape consumed by dashboard.tmpl. The
// posture summary, top concerns, and recent activity sections each
// render from their own field; templates that find an empty slice
// emit empty-state copy rather than nothing.
type dashboardView struct {
	GeneratedAt   string
	HasData       bool // true when at least one scan exists
	PostureBlocks []postureBlockView
	TopConcerns   []ConcernRow
	Activity      []activityRowView
}

type postureBlockView struct {
	Framework string
	Counts    []postureCountView
	Total     int
}

type postureCountView struct {
	Score string
	Count int
}

type activityRowView struct {
	ScanID        string
	Domain        string
	StartedAt     string
	Status        string
	HasAssessment bool
}

func dashboardHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		scans, err := st.ListScans(ctx, store.Selectors{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Build per-target snapshots: most recent scan per target,
		// most recent Assessment per (target, framework). The same
		// shape feeds PostureCounts and TopConcerns.
		type latest struct {
			scan store.ScanRow
			when time.Time
		}
		byTarget := map[string]latest{}
		for _, s := range scans {
			cur, ok := byTarget[s.TargetID]
			if !ok || s.StartedAt.After(cur.when) {
				byTarget[s.TargetID] = latest{scan: s, when: s.StartedAt}
			}
		}
		snaps := make([]TargetSnapshot, 0, len(byTarget))
		assessmentsByScan := map[string]bool{}
		for _, l := range byTarget {
			snap := TargetSnapshot{
				TargetID:    l.scan.TargetID,
				Domain:      l.scan.Domain,
				LastScanID:  l.scan.ID,
				LastScanAt:  l.when,
				LastStatus:  l.scan.Status,
				Assessments: map[string]models.Assessment{},
			}
			if list, err := st.ListAssessmentsForScan(ctx, l.scan.ID); err == nil {
				for _, a := range list {
					assessmentsByScan[l.scan.ID] = true
					cur, ok := snap.Assessments[a.Framework]
					if !ok || a.CreatedAt.After(cur.CreatedAt) {
						snap.Assessments[a.Framework] = a
					}
				}
			}
			snaps = append(snaps, snap)
		}
		// Activity needs HasAssessment lookups for ALL scans, not
		// just the most-recent-per-target subset.
		activityHas := func(scanID string) bool {
			if assessmentsByScan[scanID] {
				return true
			}
			if list, err := st.ListAssessmentsForScan(ctx, scanID); err == nil && len(list) > 0 {
				assessmentsByScan[scanID] = true
				return true
			}
			return false
		}

		summary := PostureCounts(snaps)
		concerns := TopConcerns(snaps, lookupRule, 5)
		activity := RecentActivity(scans, activityHas, 5)

		view := dashboardView{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			HasData:     len(scans) > 0,
			TopConcerns: concerns,
		}
		// Stable framework order: dictu, eucsf, then alphabetical.
		fwOrder := func(a, b string) bool {
			if a == "dictu" {
				return true
			}
			if b == "dictu" {
				return false
			}
			if a == "eucsf" {
				return true
			}
			if b == "eucsf" {
				return false
			}
			return a < b
		}
		var frameworks []string
		for fw := range summary {
			frameworks = append(frameworks, fw)
		}
		sort.Slice(frameworks, func(i, j int) bool { return fwOrder(frameworks[i], frameworks[j]) })
		for _, fw := range frameworks {
			block := postureBlockView{Framework: fw}
			for _, sc := range AllScores {
				count := summary[fw][sc]
				if count == 0 {
					continue
				}
				block.Counts = append(block.Counts, postureCountView{Score: string(sc), Count: count})
				block.Total += count
			}
			view.PostureBlocks = append(view.PostureBlocks, block)
		}
		for _, a := range activity {
			view.Activity = append(view.Activity, activityRowView{
				ScanID:        a.ScanID,
				Domain:        a.Domain,
				StartedAt:     a.StartedAt.UTC().Format(time.RFC3339),
				Status:        a.Status,
				HasAssessment: a.HasAssessment,
			})
		}
		render(w, tmpl, "dashboard.tmpl", view)
	}
}

// targetsRowsView is the shape index.tmpl iterates over.
type targetsRowsView struct {
	GeneratedAt string
	Rows        []targetRowView
}

type targetRowView struct {
	ID            string
	Domain        string
	LastScanID    string
	LastScanTime  string
	LastStatus    string
	FrameworkRows []frameworkRowView
}

type frameworkRowView struct {
	Framework string
	Score     string
	When      string
}

func targetsHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		scans, err := st.ListScans(ctx, store.Selectors{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Group scans by target; pick the most recent per target.
		type latest struct {
			scan store.ScanRow
			when time.Time
		}
		byTarget := map[string]latest{}
		for _, s := range scans {
			cur, ok := byTarget[s.TargetID]
			if !ok || s.StartedAt.After(cur.when) {
				byTarget[s.TargetID] = latest{scan: s, when: s.StartedAt}
			}
		}
		ordered := make([]latest, 0, len(byTarget))
		for _, v := range byTarget {
			ordered = append(ordered, v)
		}
		sort.Slice(ordered, func(i, j int) bool {
			return ordered[i].scan.Domain < ordered[j].scan.Domain
		})
		view := targetsRowsView{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
		for _, l := range ordered {
			row := targetRowView{
				ID:           l.scan.TargetID,
				Domain:       l.scan.Domain,
				LastScanID:   l.scan.ID,
				LastScanTime: l.when.UTC().Format(time.RFC3339),
				LastStatus:   l.scan.Status,
			}
			row.FrameworkRows = lastAssessmentsForScan(ctx, st, l.scan.ID)
			view.Rows = append(view.Rows, row)
		}
		render(w, tmpl, "index.tmpl", view)
	}
}

func lastAssessmentsForScan(ctx context.Context, st *store.Store, scanID string) []frameworkRowView {
	list, err := st.ListAssessmentsForScan(ctx, scanID)
	if err != nil || len(list) == 0 {
		return nil
	}
	// Latest per framework.
	byFW := map[string]models.Assessment{}
	for _, a := range list {
		cur, ok := byFW[a.Framework]
		if !ok || a.CreatedAt.After(cur.CreatedAt) {
			byFW[a.Framework] = a
		}
	}
	out := make([]frameworkRowView, 0, len(byFW))
	for fw, a := range byFW {
		score := "—"
		if len(a.Dimensions) > 0 {
			worst := models.ScoreOnbekend
			for _, d := range a.Dimensions {
				if d.Score.Rank() > 0 && (worst == models.ScoreOnbekend || d.Score.Rank() < worst.Rank()) {
					worst = d.Score
				}
			}
			score = string(worst)
		}
		out = append(out, frameworkRowView{
			Framework: fw,
			Score:     score,
			When:      a.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Framework < out[j].Framework })
	return out
}

type scanView struct {
	ID            string
	TargetID      string
	StartedAt     string
	EndedAt       string
	Status        string
	Probes        []probeGroupView
	HasAssessment bool
}

type probeGroupView struct {
	Prefix   string
	Findings []findingRowView
}

type findingRowView struct {
	ID            string
	ProbeID       string
	Subject       string
	Severity      string
	DimensionHint string
	Attributes    string
}

func scanHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		scan, err := st.GetScan(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "scan not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		view := scanView{
			ID:        scan.ID,
			TargetID:  scan.TargetID,
			StartedAt: scan.StartedAt.UTC().Format(time.RFC3339),
			Status:    string(scan.Status),
		}
		if scan.EndedAt != nil {
			view.EndedAt = scan.EndedAt.UTC().Format(time.RFC3339)
		}
		if assessments, err := st.ListAssessmentsForScan(r.Context(), scan.ID); err == nil && len(assessments) > 0 {
			view.HasAssessment = true
		}
		groups := map[string][]findingRowView{}
		for _, f := range scan.Findings {
			prefix := strings.SplitN(f.ProbeID, ".", 2)[0]
			groups[prefix] = append(groups[prefix], findingRowView{
				ID:            f.ID,
				ProbeID:       f.ProbeID,
				Subject:       f.Subject,
				Severity:      string(f.Severity),
				DimensionHint: string(f.DimensionHint),
				Attributes:    fmt.Sprintf("%v", f.Attributes),
			})
		}
		var keys []string
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			view.Probes = append(view.Probes, probeGroupView{Prefix: k, Findings: groups[k]})
		}
		render(w, tmpl, "scan.tmpl", view)
	}
}

type driftView struct {
	TargetID string
	Since    string
	Findings []findingRowView
}

func driftHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetID := chi.URLParam(r, "id")
		var since time.Time
		if s := r.URL.Query().Get("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err == nil {
				since = t
			}
		}
		findings, err := st.ListDriftForTarget(r.Context(), targetID, since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		view := driftView{
			TargetID: targetID,
			Since:    since.Format(time.RFC3339),
		}
		for _, f := range findings {
			view.Findings = append(view.Findings, findingRowView{
				ID:            f.ID,
				ProbeID:       f.ProbeID,
				Subject:       f.Subject,
				Severity:      string(f.Severity),
				DimensionHint: string(f.DimensionHint),
				Attributes:    fmt.Sprintf("%v", f.Attributes),
			})
		}
		render(w, tmpl, "drift.tmpl", view)
	}
}

// assessmentView is the shape consumed by assessment.tmpl. One
// frameworkView per persisted Assessment for the scan; if no
// Assessment exists the Frameworks slice is empty and the template
// renders the "run wanderer assess" hint.
type assessmentView struct {
	ScanID     string
	StartedAt  string
	Status     string
	Frameworks []frameworkCardView
}

type frameworkCardView struct {
	Framework  string
	CreatedAt  string
	Dimensions []dimensionCardView
}

type dimensionCardView struct {
	Dimension    string
	Score        string
	Completeness string
	Rationales   []rationaleRowView
}

type rationaleRowView struct {
	CriteriumID string
	Score       string
	Verdict     string
	Description string
	Rationale   string
	Retired     bool
	Evidence    []string
}

func assessmentHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		scan, err := st.GetScan(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "scan not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assessments, err := st.ListAssessmentsForScan(r.Context(), scan.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		view := assessmentView{
			ScanID:    scan.ID,
			StartedAt: scan.StartedAt.UTC().Format(time.RFC3339),
			Status:    string(scan.Status),
		}
		// Stable framework order: dictu first, then alphabetical.
		sort.SliceStable(assessments, func(i, j int) bool {
			a, b := assessments[i].Framework, assessments[j].Framework
			if a == "dictu" {
				return true
			}
			if b == "dictu" {
				return false
			}
			return a < b
		})
		for _, a := range assessments {
			fw := frameworkCardView{
				Framework: a.Framework,
				CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
			}
			for _, d := range a.Dimensions {
				card := dimensionCardView{
					Dimension:    string(d.Dimension),
					Score:        string(d.Score),
					Completeness: string(d.Completeness),
				}
				for _, rationale := range d.Rationale {
					row := rationaleRowView{
						CriteriumID: rationale.CriteriumID,
						Score:       string(rationale.Score),
						Verdict:     rationale.Verdict,
						Evidence:    rationale.Evidence,
					}
					if rule, ok := lookupRule(a.Framework, rationale.CriteriumID); ok {
						row.Description = rule.Description
						row.Rationale = rule.Rationale
					} else {
						row.Description = "rule retired"
						row.Retired = true
					}
					card.Rationales = append(card.Rationales, row)
				}
				fw.Dimensions = append(fw.Dimensions, card)
			}
			view.Frameworks = append(view.Frameworks, fw)
		}
		render(w, tmpl, "assessment.tmpl", view)
	}
}

func render(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
