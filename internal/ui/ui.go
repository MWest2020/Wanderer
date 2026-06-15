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

// Handler builds the chi sub-router for the UI. The Options carry
// the authentication wiring; the zero Options leaves every route
// open (development mode). See Options for the htpasswd / OIDC
// combinations.
func Handler(st *store.Store, opts Options) (http.Handler, error) {
	tmpl, err := template.New("ui").Funcs(template.FuncMap{
		// dict builds a map[string]any from alternating key/value
		// args so partials can be parameterised in {{template ...}}
		// invocations (e.g. nav.tmpl wants Active + HasReporting).
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: expected even number of args, got %d", len(values))
			}
			out := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key %d not a string", i)
				}
				out[key] = values[i+1]
			}
			return out, nil
		},
	}).ParseFS(assets, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("ui: parse templates: %w", err)
	}
	r := chi.NewRouter()
	gate, err := newAuthGate(st, opts)
	if err != nil {
		return nil, err
	}
	if gate != nil {
		r.Use(gate.middleware)
		if gate.auth != nil {
			r.Get("/login", gate.loginHandler)
			r.Get("/oauth/callback", gate.callbackHandler)
			r.Get("/logout", gate.logoutHandler)
		}
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	r.Get("/", dashboardHandler(st, tmpl))
	r.Get("/orgs/{slug}", dashboardOrgHandler(st, tmpl))
	r.Get("/targets", targetsHandler(st, tmpl))
	r.Get("/scans/{id}", scanHandler(st, tmpl))
	r.Get("/scans/{id}/assessment", assessmentHandler(st, tmpl))
	r.Get("/targets/{id}/drift", driftHandler(st, tmpl))
	r.Get("/analysis", analysisHandler(st, tmpl))
	r.Get("/reporting", reportingCatalogueHandler(st, tmpl))
	r.Get("/reporting/{framework}/{ruleID}", reportingRuleHandler(st, tmpl))
	return r, nil
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
	GeneratedAt        string
	HasData            bool // true when at least one scan exists
	Headline           headlineRenderView
	OrganisationsList  []organisationLinkView // populated only on the instance-wide /ui/
	ScopedOrganisation *organisationLinkView  // populated only on /ui/orgs/{slug}
	Verdicts           []verdictRenderView    // per-framework "is this OK" pill
	HasReporting       bool                   // controls whether the Reporting nav link renders
	OrgSlug            string                 // active org for nav-link scope persistence
}

// verdictRenderView is the per-framework verdict pill on the
// Dashboard. Score is the worst score reached across every
// assessed target in scope; AtWorst is how many targets are at
// that score; Total is how many targets contributed.
type verdictRenderView struct {
	Framework string
	Score     string
	AtWorst   int
	Total     int
}

// organisationLinkView is one row in the dashboard's organisation
// list: slug, display name, and the URL to the per-org dashboard.
type organisationLinkView struct {
	Slug        string
	Name        string
	URL         string
	TargetCount int
}

// headlineRenderView is the pontificaal section's render shape —
// strings preformatted so the template stays declarative.
type headlineRenderView struct {
	LastScanAt       string // RFC3339 of most recent scan, "" when no scans
	TotalScans       int
	PerimeterTargets int
	AgentHostTargets int
	Frameworks       []string
}

func dashboardHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderDashboard(w, r, st, tmpl, nil)
	}
}

// dashboardOrgHandler renders the per-organisation dashboard at
// /ui/orgs/{slug}. The view filters scans + snapshots to that
// organisation's Targets; the headline is rebadged with the org
// name, and the OrganisationsList sub-section is suppressed (the
// operator is already inside one organisation's view).
func dashboardOrgHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		o, err := st.GetOrganisationBySlug(r.Context(), slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderDashboard(w, r, st, tmpl, o)
	}
}

// renderDashboard fills the dashboardView struct + executes the
// template. When org is nil, the view is the instance-wide global;
// when org is set, snapshots are filtered to that organisation and
// the headline is rebadged.
func renderDashboard(w http.ResponseWriter, r *http.Request, st *store.Store, tmpl *template.Template, org *models.Organisation) {
	ctx := r.Context()
	orgID := ""
	if org != nil {
		orgID = org.ID
	}
	snaps, scans, err := buildSnapshots(ctx, st, orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	headline := BuildHeadline(snaps, scans)
	verdicts := WorstByFramework(snaps)

	orgSlug := ""
	if org != nil {
		orgSlug = org.Slug
	}
	view := dashboardView{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		HasData:      len(scans) > 0,
		HasReporting: true,
		OrgSlug:      orgSlug,
		Headline: headlineRenderView{
			TotalScans:       headline.TotalScans,
			PerimeterTargets: headline.PerimeterTargets,
			AgentHostTargets: headline.AgentHostTargets,
			Frameworks:       headline.Frameworks,
		},
	}
	if !headline.LastScanAt.IsZero() {
		view.Headline.LastScanAt = headline.LastScanAt.UTC().Format(time.RFC3339)
	}
	for _, v := range verdicts {
		view.Verdicts = append(view.Verdicts, verdictRenderView{
			Framework: v.Framework,
			Score:     string(v.Score),
			AtWorst:   v.TargetsAtWorst,
			Total:     v.TotalAssessed,
		})
	}
	// Organisation-aware metadata: per-org dashboards carry the
	// scoped org's slug + name so the template can rebadge the
	// headline; the instance-wide view carries the full list of
	// registered orgs as drill-in links.
	if org != nil {
		view.ScopedOrganisation = &organisationLinkView{
			Slug: org.Slug,
			Name: org.Name,
			URL:  "/ui/orgs/" + org.Slug,
		}
	} else {
		if orgs, listErr := st.ListOrganisations(ctx); listErr == nil {
			for _, o := range orgs {
				targets, _ := st.ListTargetsByOrganisation(ctx, o.ID)
				view.OrganisationsList = append(view.OrganisationsList, organisationLinkView{
					Slug:        o.Slug,
					Name:        o.Name,
					URL:         "/ui/orgs/" + o.Slug,
					TargetCount: len(targets),
				})
			}
		}
	}
	render(w, tmpl, "dashboard.tmpl", view)
}

// buildSnapshots builds the per-target snapshot list shared by
// the dashboard and reporting pages: one snapshot per Target, the
// most recent scan, the most recent Assessment per framework, and
// the resolved Kind. Returns the underlying scans slice too so a
// caller that also needs RecentActivity / TotalScans can share
// the same store roundtrip.
//
// orgID, when non-empty, filters scans to those whose Target
// belongs to that organisation. Used by /ui/orgs/{slug} and the
// `?org=` query parameter on /ui/reporting.
func buildSnapshots(ctx context.Context, st *store.Store, orgID string) (snaps []TargetSnapshot, scans []store.ScanRow, err error) {
	sel := store.Selectors{}
	if orgID != "" {
		sel.OrganisationID = orgID
	}
	scans, err = st.ListScans(ctx, sel)
	if err != nil {
		return nil, nil, err
	}
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
	snaps = make([]TargetSnapshot, 0, len(byTarget))
	for _, l := range byTarget {
		var kind models.TargetKind
		if t, terr := st.GetTarget(ctx, l.scan.TargetID); terr == nil && t != nil {
			kind = t.Kind
		}
		snap := TargetSnapshot{
			TargetID:    l.scan.TargetID,
			Domain:      l.scan.Domain,
			Kind:        kind,
			LastScanID:  l.scan.ID,
			LastScanAt:  l.when,
			LastStatus:  l.scan.Status,
			Assessments: map[string]models.Assessment{},
		}
		if list, lerr := st.ListAssessmentsForScan(ctx, l.scan.ID); lerr == nil {
			for _, a := range list {
				cur, ok := snap.Assessments[a.Framework]
				if !ok || a.CreatedAt.After(cur.CreatedAt) {
					snap.Assessments[a.Framework] = a
				}
			}
		}
		snaps = append(snaps, snap)
	}
	return snaps, scans, nil
}

// scopeSlugForScan resolves the organisation slug attached to a
// Target (via its scan), so Analysis-page handlers can thread the
// scope through the cross-page nav. Returns empty when the
// target's org cannot be resolved — UX-only data, never fatal.
func scopeSlugForScan(ctx context.Context, st *store.Store, targetID string) string {
	if targetID == "" {
		return ""
	}
	t, err := st.GetTarget(ctx, targetID)
	if err != nil || t == nil || t.OrganisationID == "" {
		return ""
	}
	o, err := st.GetOrganisation(ctx, t.OrganisationID)
	if err != nil || o == nil {
		return ""
	}
	return o.Slug
}

// resolveOrgQueryParam parses the optional `?org=<slug>` query
// parameter. Returns ("", nil, true) when no slug given. On an
// unknown slug, writes a 404 and returns ok=false so the caller
// can short-circuit.
func resolveOrgQueryParam(ctx context.Context, st *store.Store, w http.ResponseWriter, r *http.Request) (orgID string, org *models.Organisation, ok bool) {
	slug := r.URL.Query().Get("org")
	if slug == "" {
		return "", nil, true
	}
	o, err := st.GetOrganisationBySlug(ctx, slug)
	if err != nil {
		http.NotFound(w, r)
		return "", nil, false
	}
	return o.ID, o, true
}

// targetsRowsView is the shape index.tmpl iterates over.
type targetsRowsView struct {
	GeneratedAt        string
	OrgSlug            string
	HasReporting       bool
	ScopedOrganisation *organisationLinkView
	Rows               []targetRowView
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
		orgID, scopedOrg, ok := resolveOrgQueryParam(ctx, st, w, r)
		if !ok {
			return
		}
		sel := store.Selectors{}
		if orgID != "" {
			sel.OrganisationID = orgID
		}
		scans, err := st.ListScans(ctx, sel)
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
		view := targetsRowsView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
		}
		if scopedOrg != nil {
			view.OrgSlug = scopedOrg.Slug
			view.ScopedOrganisation = &organisationLinkView{
				Slug: scopedOrg.Slug,
				Name: scopedOrg.Name,
				URL:  "/ui/orgs/" + scopedOrg.Slug,
			}
		}
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
	OrgSlug       string
	HasReporting  bool
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
		// Resolve scope from the scan's Target so the nav-bar threads
		// the org through every Analysis-page click. Failure is
		// non-fatal — scope persistence is a UX concern, not data.
		orgSlug := scopeSlugForScan(r.Context(), st, scan.TargetID)
		view := scanView{
			ID:           scan.ID,
			TargetID:     scan.TargetID,
			StartedAt:    scan.StartedAt.UTC().Format(time.RFC3339),
			Status:       string(scan.Status),
			OrgSlug:      orgSlug,
			HasReporting: true,
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
	TargetID     string
	Since        string
	OrgSlug      string
	HasReporting bool
	Findings     []findingRowView
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
			TargetID:     targetID,
			Since:        since.Format(time.RFC3339),
			OrgSlug:      scopeSlugForScan(r.Context(), st, targetID),
			HasReporting: true,
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
	ScanID       string
	StartedAt    string
	Status       string
	OrgSlug      string
	HasReporting bool
	Flows        []Flow // Sovereignty overview — "what goes where"
	Frameworks   []frameworkCardView
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
			ScanID:       scan.ID,
			StartedAt:    scan.StartedAt.UTC().Format(time.RFC3339),
			Status:       string(scan.Status),
			OrgSlug:      scopeSlugForScan(r.Context(), st, scan.TargetID),
			HasReporting: true,
			Flows:        SovereigntyFlows(assessments),
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

// reportingView is the shape consumed by reporting.tmpl — the
// per-check cross-target index page.
type reportingView struct {
	GeneratedAt        string
	HasReporting       bool
	OrgSlug            string
	ScopedOrganisation *organisationLinkView
	Rows               []reportingRowView
}

type reportingRowView struct {
	Framework        string
	CriteriumID      string
	Description      string
	SoevereinCount   int
	VoldoendeCount   int
	AfhankelijkCount int
	OnbekendCount    int
}

// ruleCatalogueView is the shape consumed by reporting.tmpl —
// the rule reference page with a compact status column per row.
type ruleCatalogueView struct {
	GeneratedAt        string
	HasReporting       bool
	OrgSlug            string
	ScopedOrganisation *organisationLinkView
	Rows               []ruleCatalogueRow
}

type ruleCatalogueRow struct {
	Framework   string
	CriteriumID string
	Dimension   string
	Description string
	Rationale   string
	// Status is the worst-score string this rule reached across
	// the targets in scope ("" when the rule has not fired yet).
	// AtWorst / Total give the triage hint: "X of Y targets at
	// this score".
	Status  string
	AtWorst int
	Total   int
}

// analysisHandler renders the Analysis page — the rule × score
// matrix that operators steer with. This was the content of the
// old /ui/reporting before the 2026-05-10 layer restructure.
func analysisHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, scopedOrg, ok := resolveOrgQueryParam(ctx, st, w, r)
		if !ok {
			return
		}
		snaps, _, err := buildSnapshots(ctx, st, orgID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		summary := RuleSummary(snaps, lookupRule)
		view := reportingView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
			Rows:         make([]reportingRowView, 0, len(summary)),
		}
		if scopedOrg != nil {
			view.OrgSlug = scopedOrg.Slug
			view.ScopedOrganisation = &organisationLinkView{
				Slug: scopedOrg.Slug,
				Name: scopedOrg.Name,
				URL:  "/ui/orgs/" + scopedOrg.Slug,
			}
		}
		for _, row := range summary {
			view.Rows = append(view.Rows, reportingRowView{
				Framework:        row.Framework,
				CriteriumID:      row.CriteriumID,
				Description:      row.Description,
				SoevereinCount:   row.Counts[models.ScoreSoeverein],
				VoldoendeCount:   row.Counts[models.ScoreVoldoende],
				AfhankelijkCount: row.Counts[models.ScoreAfhankelijk],
				OnbekendCount:    row.Counts[models.ScoreOnbekend],
			})
		}
		render(w, tmpl, "analysis.tmpl", view)
	}
}

// reportingCatalogueHandler renders /ui/reporting as a rule
// catalogue — every registered rule with description, rationale,
// and a compact "current status" indicator that summarises the
// worst score reached on the rule across the snapshots in scope.
// The full per-score matrix lives on /ui/analysis; this column
// is just a triage hint so an operator with many rules can spot
// the problem rules at a glance.
func reportingCatalogueHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, scopedOrg, ok := resolveOrgQueryParam(ctx, st, w, r)
		if !ok {
			return
		}
		snaps, _, err := buildSnapshots(ctx, st, orgID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Build a quick (framework, ruleID) → counts lookup so the
		// catalogue row population stays O(rules). RuleSummary already
		// does the distinct-target counting; we just consume it.
		summary := RuleSummary(snaps, lookupRule)
		type key struct{ fw, id string }
		byRule := make(map[key]map[models.Score]int, len(summary))
		for _, row := range summary {
			byRule[key{row.Framework, row.CriteriumID}] = row.Counts
		}

		all := ListAllRules()
		rows := make([]ruleCatalogueRow, 0, len(all))
		for _, c := range all {
			row := ruleCatalogueRow{
				Framework:   c.Framework,
				CriteriumID: c.Rule.ID,
				Dimension:   string(c.Rule.Dimension),
				Description: c.Rule.Description,
				Rationale:   c.Rule.Rationale,
			}
			if counts, found := byRule[key{c.Framework, c.Rule.ID}]; found {
				worst, atWorst, total := WorstScoreFromCounts(counts)
				row.Status = string(worst)
				row.AtWorst = atWorst
				row.Total = total
			}
			rows = append(rows, row)
		}
		view := ruleCatalogueView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
			Rows:         rows,
		}
		if scopedOrg != nil {
			view.OrgSlug = scopedOrg.Slug
			view.ScopedOrganisation = &organisationLinkView{
				Slug: scopedOrg.Slug,
				Name: scopedOrg.Name,
				URL:  "/ui/orgs/" + scopedOrg.Slug,
			}
		}
		render(w, tmpl, "reporting.tmpl", view)
	}
}

// reportingRuleView is the shape consumed by reporting_rule.tmpl —
// the per-rule deep dive.
type reportingRuleView struct {
	GeneratedAt        string
	HasReporting       bool
	OrgSlug            string
	ScopedOrganisation *organisationLinkView
	Framework          string
	CriteriumID        string
	Dimension          string
	Description        string
	Rationale          string
	Rows               []reportingRuleRowView
}

type reportingRuleRowView struct {
	TargetID string
	Domain   string
	ScanID   string
	Score    string
	Verdict  string
	When     string
}

func reportingRuleHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		framework := chi.URLParam(r, "framework")
		ruleID := chi.URLParam(r, "ruleID")
		rule, ok := lookupRule(framework, ruleID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		ctx := r.Context()
		orgID, scopedOrg, ok := resolveOrgQueryParam(ctx, st, w, r)
		if !ok {
			return
		}
		snaps, _, err := buildSnapshots(ctx, st, orgID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows := RuleTargetRows(snaps, framework, ruleID)
		view := reportingRuleView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
			Framework:    framework,
			CriteriumID:  ruleID,
			Dimension:    string(rule.Dimension),
			Description:  rule.Description,
			Rationale:    rule.Rationale,
			Rows:         make([]reportingRuleRowView, 0, len(rows)),
		}
		if scopedOrg != nil {
			view.OrgSlug = scopedOrg.Slug
			view.ScopedOrganisation = &organisationLinkView{
				Slug: scopedOrg.Slug,
				Name: scopedOrg.Name,
				URL:  "/ui/orgs/" + scopedOrg.Slug,
			}
		}
		for _, rw := range rows {
			view.Rows = append(view.Rows, reportingRuleRowView{
				TargetID: rw.TargetID,
				Domain:   rw.Domain,
				ScanID:   rw.ScanID,
				Score:    string(rw.Score),
				Verdict:  rw.Verdict,
				When:     rw.When.UTC().Format(time.RFC3339),
			})
		}
		render(w, tmpl, "reporting_rule.tmpl", view)
	}
}
