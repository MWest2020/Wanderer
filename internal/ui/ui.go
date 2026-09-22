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
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/internal/scheduler"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.tmpl static/*
var assets embed.FS

// Templates parses the UI's embedded templates standalone. It is the
// seam a caller outside the login-gated Handler (the public /demo
// route — openspec change 2026-09-21-demo-pagina — lives on the root
// router, never under /ui) uses to render the same views with the
// same template set, without pulling in the auth gate or any of the
// chi routing below.
func Templates() (*template.Template, error) {
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
	return tmpl, nil
}

// Handler builds the chi sub-router for the UI. The Options carry
// the authentication wiring; the zero Options leaves every route
// open (development mode). See Options for the htpasswd / OIDC
// combinations.
func Handler(st *store.Store, opts Options) (http.Handler, error) {
	tmpl, err := Templates()
	if err != nil {
		return nil, err
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
	// The scan route is gated on a signed-in user, not on a dev-mode
	// flag: it mounts only when both a Scanner is wired AND the
	// instance has authentication configured (htpasswd or OIDC), since
	// that is the only thing standing between an anonymous request and
	// the handler once mounted (spec.md "UI surface stays read-only").
	// An instance with no authentication at all refuses the route and
	// says why, once, at startup — proposal.md "no anonymous scanning".
	allowScan := opts.Scanner != nil && gate != nil
	if opts.Scanner != nil && gate == nil {
		slog.Warn("ui.scan.disabled", "reason", "no authentication configured — set --ui-htpasswd or an oidc: block to enable scanning from the UI")
	}
	if allowScan {
		// The ONE sanctioned mutating route. The read-only test allows
		// exactly this POST and no other.
		r.Post("/scan", scanTriggerHandler(st, opts.Scanner))
		// Read-only poll page the POST bounces to while the background
		// scan runs; redirects to the assessment once it lands.
		r.Get("/scan-status", scanStatusHandler(st, tmpl))
	}
	r.Get("/", dashboardHandler(st, tmpl, allowScan))
	r.Get("/orgs/{slug}", dashboardOrgHandler(st, tmpl, allowScan))
	r.Get("/targets", targetsHandler(st, tmpl))
	r.Get("/scans/{id}", scanHandler(st, tmpl))
	r.Get("/scans/{id}/answer", answerHandler(st, tmpl))
	r.Get("/scans/{id}/assessment", assessmentHandler(st, tmpl))
	r.Get("/targets/{id}/drift", driftHandler(st, tmpl))
	r.Get("/trends", trendsHandler(st, tmpl))
	// The fleet page (spec.md "Domeinen zijn bij te houden als vloot")
	// is read-only for everyone, same as the rest of the UI. Adding
	// and removing a domain is gated the same way as scanning: a
	// signed-in user, not a dev-mode flag — gate != nil is exactly the
	// "authentication is configured" half of allowScan's condition
	// above (there is no Scanner-equivalent dependency here).
	allowFleetEdit := gate != nil
	r.Get("/orgs/{slug}/fleet", fleetHandler(st, tmpl, allowFleetEdit, opts.Schedules))
	if allowFleetEdit {
		// Two more sanctioned mutating routes, alongside /scan. Both
		// require {slug} to resolve to a real organisation before they
		// touch the store — see fleetAddHandler / fleetRemoveHandler.
		r.Post("/orgs/{slug}/fleet/domains", fleetAddHandler(st))
		r.Post("/orgs/{slug}/fleet/domains/{domain}/remove", fleetRemoveHandler(st))
	}
	// Retired layers: the Analysis matrix + Reporting catalogue
	// consolidated into Trends. Redirect so deep links survive.
	r.Get("/analysis", redirectToTrends())
	r.Get("/reporting", redirectToTrends())
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

// doorView is the shape consumed by dashboard.tmpl — the vloot-first
// entry surface (openspec change 2026-09-22-drie-lagen-ciso, spec.md
// "De vloot is de eerste laag"): the organisation's fleet score, the
// count of domains that are not soeverein, the per-stroom distribution,
// the three costliest rules, and the domains themselves sorted worst
// first. The scan input stays, but as an action inside the page, not
// its headline (spec.md "een invoerveld ... SHALL niet de hoofdzaak
// van de pagina zijn").
type doorView struct {
	GeneratedAt        string
	HasReporting       bool                   // controls whether the Trends nav link renders
	ScopedOrganisation *organisationLinkView  // populated only on /ui/orgs/{slug}
	OrganisationsList  []organisationLinkView // populated only when unscoped, so a multi-org instance can still reach a single vloot
	OrgSlug            string                 // active org for nav-link scope persistence
	AllowScan          bool                   // signed-in user: render the door's scan input
	AgentHosts         []string               // enrolled, non-revoked agent hostnames — selectable from the same input
	FleetManageURL     string                 // /ui/orgs/{slug}/fleet — "" when unscoped (task 2.3: management stays where it is)

	HasFleet bool // at least one domain with a scan to show

	// The vloot score itself (fleet_score.go's FleetSummary) —
	// proposal.md's valkuil: NotSovereign is shown right next to the
	// score so a high X/N can never hide a failing domain.
	X, N               int
	Unanswered         int
	NotSovereign       int
	DomainsWithoutScan int
	TopRules           []ConcernRow
	Flows              []FlowRollup

	Domains []doorDomainView
}

// doorDomainView is one row in the door's vloot list — the same x/n
// shape fleet.tmpl's row uses, without the schedule and delta columns:
// those stay on /ui/orgs/{slug}/fleet, the fleet management page task
// 2.3 leaves in place.
type doorDomainView struct {
	Domain     string
	Kind       string
	LastScanAt string
	AnswerURL  string // /ui/scans/{id}/answer

	HasScore     bool
	X, N         int
	Unanswered   int
	WorstFlow    string
	WorstVerdict string
}

// buildDoorDomains turns an organisation's (or the whole instance's)
// snapshots into the door's domain rows, sorted worst score first
// (spec.md "de lijst SHALL standaard gesorteerd zijn op de slechtste
// score eerst") via the same fleetScorePercent ranking fleetHandler's
// "sort=score" uses, so the two lists agree. A domain without a
// voltooide scan — the same gate BuildFleetSummary applies — carries
// no score and sorts after every scored domain, alphabetically.
func buildDoorDomains(snaps []TargetSnapshot) []doorDomainView {
	type scoredRow struct {
		view  doorDomainView
		score FleetScore
	}
	scored := make([]scoredRow, 0, len(snaps))
	var unscanned []doorDomainView

	for _, s := range snaps {
		row := doorDomainView{Domain: s.Domain, Kind: string(s.Kind)}
		if !s.LastScanAt.IsZero() {
			row.LastScanAt = s.LastScanAt.UTC().Format(time.RFC3339)
		}
		if s.LastScanID != "" {
			row.AnswerURL = "/ui/scans/" + s.LastScanID + "/answer"
		}
		switch models.ScanStatus(s.LastStatus) {
		case models.ScanStatusComplete, models.ScanStatusPartial:
		default:
			unscanned = append(unscanned, row)
			continue
		}
		assessments := make([]models.Assessment, 0, len(s.Assessments))
		for _, a := range s.Assessments {
			assessments = append(assessments, a)
		}
		fs := BuildFleetScore(assessments, nil)
		row.HasScore = true
		row.X, row.N, row.Unanswered = fs.X, fs.N, fs.Unanswered
		row.WorstFlow, row.WorstVerdict = fs.WorstFlow, fs.WorstVerdict
		scored = append(scored, scoredRow{view: row, score: fs})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return fleetScorePercent(scored[i].score) < fleetScorePercent(scored[j].score)
	})
	sort.Slice(unscanned, func(i, j int) bool { return unscanned[i].Domain < unscanned[j].Domain })

	out := make([]doorDomainView, 0, len(snaps))
	for _, s := range scored {
		out = append(out, s.view)
	}
	return append(out, unscanned...)
}

// dashboardTargetRow is the glanceable per-target line on the fleet
// table (/ui/trends) — domain, when it was last scanned, and its
// headline sovereignty verdict, linking straight to that scan's
// report.
type dashboardTargetRow struct {
	Domain              string
	Kind                string
	LastScanAt          string
	LastStatus          string
	Verdict             string // worst score across the preferred assessment; "" when not yet assessed
	VerdictDimensions   string // comma-joined dimensions the worst score was computed over; "" when Verdict is ""
	ReportURL           string // /ui/scans/{id}/assessment
	AccountabilityLabel string
	AccountabilityClass string
	AccountabilityLink  string // ReportURL + "#wand-accountability"; "" when there is no report yet
}

// verdictRenderView is the per-framework verdict pill on Trends.
// Score is the worst score reached across every assessed target in
// scope; AtWorst is how many targets are at that score; Total is how
// many targets contributed.
type verdictRenderView struct {
	Framework string
	Score     string
	AtWorst   int
	Total     int
}

// organisationLinkView is one row in the instance-wide organisation
// list (Trends) or the door/nav's scoped-org badge: slug, display
// name, and the URL to the per-org door.
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

// scanTriggerHandler is the sanctioned mutating route (POST /ui/scan),
// mounted only for a signed-in user (Handler mounts it only when both
// a Scanner and authentication are configured). It kicks the scan off
// in the background and bounces the browser to a status page that
// polls until the result is ready. Running the scan synchronously
// would hold the POST open for the full probe budget — the transit
// probe alone waits up to 30s for traceroute replies — so the browser
// appears frozen and the user re-submits. Detaching also keeps the DB
// writes off the request context, which a browser cancel would
// otherwise abort mid-scan ("begin tx: context canceled").
func scanTriggerHandler(st *store.Store, sc ScanTrigger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := strings.TrimSpace(r.FormValue("domain"))
		if domain == "" {
			http.Error(w, "domain is required", http.StatusBadRequest)
			return
		}
		target := models.Target{Domain: domain}
		// The door's one input doubles as the agent-host picker
		// (spec.md "selectable from the same input"): a submission
		// that names an enrolled, non-revoked agent hostname scans as
		// a host, not a public domain.
		if host, herr := models.NormaliseHost(domain); herr == nil {
			if ag, aerr := st.GetAgentByHostname(r.Context(), host); aerr == nil && ag != nil && !ag.Revoked() {
				target.Kind = models.TargetKindHost
			}
		}
		if err := target.Validate(); err != nil {
			// Reject bad input synchronously so we never launch a
			// background scan that can only ever fail.
			http.Error(w, "invalid domain: "+err.Error(), http.StatusBadRequest)
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			scan, err := sc.Scan(ctx, target)
			if err != nil {
				slog.Error("ui.scan.failed", "domain", domain, "err", err)
				return
			}
			// Assess with the wand pack so the Sovereignty overview +
			// diagram populate on the page the status poller lands on.
			findings, ferr := st.FindingsForAssessment(ctx, scan)
			if ferr != nil {
				slog.Error("ui.scan.assess_failed", "scan_id", scan.ID, "err", ferr)
				return
			}
			a := &models.Assessment{
				ScanID:     scan.ID,
				Framework:  "wand",
				Dimensions: assessor.Assess(findings, wand.DefaultRules()),
			}
			if err := st.CreateAssessment(ctx, a); err != nil {
				slog.Error("ui.scan.assess_failed", "scan_id", scan.ID, "err", err)
			}
		}()
		http.Redirect(w, r, "/ui/scan-status?domain="+url.QueryEscape(domain), http.StatusSeeOther)
	}
}

// scanStatusView is the shape consumed by scan-status.tmpl — the brief
// bridge page shown only until the background scan's row exists (see
// scanStatusHandler); once it does, the answer page takes over.
type scanStatusView struct {
	Domain       string
	HasReporting bool
}

// scanStatusHandler renders a self-refreshing page for a background
// scan keyed by domain. It finds the most recent scan for the domain
// and, once that scan row exists, redirects to its answer page — so
// the page polls (via an HTML meta-refresh, no JS) only for the brief
// window before the background goroutine has even created the scan.
// The answer page itself (answerHandler) takes over rendering progress
// from there; it does not wait for a persisted Assessment. Mounted
// alongside the scan form (dev mode only).
func scanStatusHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := strings.TrimSpace(r.URL.Query().Get("domain"))
		if domain == "" {
			http.Redirect(w, r, "/ui/", http.StatusSeeOther)
			return
		}
		scans, err := st.ListScans(r.Context(), store.Selectors{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var latest *store.ScanRow
		for i := range scans {
			if !strings.EqualFold(scans[i].Domain, domain) {
				continue
			}
			if latest == nil || scans[i].StartedAt.After(latest.StartedAt) {
				latest = &scans[i]
			}
		}
		if latest != nil {
			http.Redirect(w, r, "/ui/scans/"+latest.ID+"/answer", http.StatusSeeOther)
			return
		}
		render(w, tmpl, "scan-status.tmpl", scanStatusView{Domain: domain, HasReporting: true})
	}
}

func dashboardHandler(st *store.Store, tmpl *template.Template, allowScan bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderDoor(w, r, st, tmpl, nil, allowScan)
	}
}

// dashboardOrgHandler renders the per-organisation door at
// /ui/orgs/{slug}: the recently-answered list filters to that
// organisation's targets and the header rebadges with the org name.
func dashboardOrgHandler(st *store.Store, tmpl *template.Template, allowScan bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		o, err := st.GetOrganisationBySlug(r.Context(), slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderDoor(w, r, st, tmpl, o, allowScan)
	}
}

// enrolledAgentHostnames returns the hostnames of every enrolled,
// non-revoked agent — the door's datalist, so a host that reports
// through an agent is as easy to pick as typing a domain (spec.md
// "selectable from the same input"). ListAgents already orders by
// hostname.
func enrolledAgentHostnames(ctx context.Context, st *store.Store) []string {
	agents, err := st.ListAgents(ctx)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(agents))
	for _, a := range agents {
		if a.Revoked() {
			continue
		}
		out = append(out, a.Hostname)
	}
	return out
}

// renderDoor fills the doorView struct + executes dashboard.tmpl —
// the vloot-first entry surface. When org is nil the view covers every
// target in the instance (mirroring /ui/trends' unscoped mode, with
// the organisation list alongside so a multi-org instance can still
// reach a single vloot); when org is set, the view is that
// organisation's vloot alone.
func renderDoor(w http.ResponseWriter, r *http.Request, st *store.Store, tmpl *template.Template, org *models.Organisation, allowScan bool) {
	ctx := r.Context()
	orgID := ""
	if org != nil {
		orgID = org.ID
	}
	snaps, _, err := buildSnapshots(ctx, st, orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	orgSlug := ""
	if org != nil {
		orgSlug = org.Slug
	}
	summary := BuildFleetSummary(snaps, lookupRule)
	view := doorView{
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		HasReporting:       true,
		OrgSlug:            orgSlug,
		AllowScan:          allowScan,
		X:                  summary.X,
		N:                  summary.N,
		Unanswered:         summary.Unanswered,
		NotSovereign:       summary.NotSovereign,
		DomainsWithoutScan: summary.DomainsWithoutScan,
		TopRules:           summary.TopRules,
		Flows:              summary.Flows,
		Domains:            buildDoorDomains(snaps),
	}
	view.HasFleet = len(view.Domains) > 0
	if allowScan {
		view.AgentHosts = enrolledAgentHostnames(ctx, st)
	}
	if org != nil {
		view.ScopedOrganisation = &organisationLinkView{
			Slug: org.Slug,
			Name: org.Name,
			URL:  "/ui/orgs/" + org.Slug,
		}
		view.FleetManageURL = "/ui/orgs/" + org.Slug + "/fleet"
	} else if orgs, listErr := st.ListOrganisations(ctx); listErr == nil {
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

// fleetView is the shape fleet.tmpl renders: every domain currently in
// one organisation's fleet (spec.md "Domeinen zijn bij te houden als
// vloot"), each with its x/n score, its change since the previous
// scan, and its schedule. Domains is pre-sorted by SortKey; a domain
// that has never been scanned always sorts after every scanned one,
// regardless of SortKey — run 03's task-ref: a domain without a scan
// "verdwijnt niet naar onderen [per ongeluk] bij sorteren op score;
// zet die apart onderaan".
type fleetView struct {
	GeneratedAt        string
	HasReporting       bool
	OrgSlug            string
	ScopedOrganisation *organisationLinkView
	AllowEdit          bool // signed-in user: render the add/remove forms
	SortKey            string
	SortLinks          []fleetSortLinkView
	Domains            []fleetDomainView
}

// fleetSortLinkView is one of the "sort by" links fleet.tmpl renders —
// run 03's task-ref: "Sorteren ... via links met een query-parameter
// (geen JavaScript nodig)". Active marks the currently-applied sort so
// the choice stays visible on the page.
type fleetSortLinkView struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

// fleetDomainView is one row. LastScanAt is "" when the domain has
// never been scanned (fleet.tmpl renders "nog niet gescand" for that
// case, per spec.md's scenario) — HasScore is false in that case too,
// since there is nothing yet to score. ScheduleName is "" when no
// schedule in the schedules file targets this domain.
type fleetDomainView struct {
	Domain       string
	LastScanAt   string
	ScheduleName string
	ScheduleCron string

	// HasScore is true once the domain has at least one scan; the x/n
	// score can still read 0/0 if that scan has not been assessed yet.
	HasScore     bool
	X, N         int
	Unanswered   int
	WorstFlow    string
	WorstVerdict string

	// HasPrevious is true once there is a scan before the latest one
	// to diff against (spec.md run 03 task 3.2).
	HasPrevious  bool
	XDelta       int
	NDelta       int
	FlippedFlows []string
}

// fleetRow is fleetHandler's working shape before it is split into the
// scored/unscanned buckets and formatted for the template — it keeps
// LastScanAt as a time.Time so sorting by "last scan" doesn't need to
// re-parse the RFC3339 string fleet.tmpl gets.
type fleetRow struct {
	target       models.Target
	scheduleName string
	scheduleCron string
	hasScan      bool
	lastScanAt   time.Time
	score        FleetScore
	delta        FleetDelta
}

// fleetSortKeys enumerates the sortable columns (run 03's task-ref:
// "Sorteren op score, op verandering en op laatste scan") plus the
// default alphabetical order, in the order their links render.
var fleetSortKeys = []struct{ key, label string }{
	{"domain", "Domein"},
	{"score", "Score"},
	{"change", "Verandering"},
	{"last_scan", "Laatste scan"},
}

// fleetHandler renders /ui/orgs/{slug}/fleet: the organisation's fleet
// of domains, each scored with BuildFleetScore against its latest scan
// and diffed against the scan before that with BuildFleetDelta.
// Read-only — allowEdit only controls whether the template renders the
// add/remove forms; the mutating routes themselves are gated
// separately in Handler.
func fleetHandler(st *store.Store, tmpl *template.Template, allowEdit bool, schedules ScheduleSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slug := chi.URLParam(r, "slug")
		org, err := st.GetOrganisationBySlug(ctx, slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		domains, err := st.ListFleetDomains(ctx, org.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scans, err := st.ListScans(ctx, store.Selectors{OrganisationID: org.ID})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lastScanByTarget := map[string]store.ScanRow{}
		for _, sc := range scans {
			if cur, ok := lastScanByTarget[sc.TargetID]; !ok || sc.StartedAt.After(cur.StartedAt) {
				lastScanByTarget[sc.TargetID] = sc
			}
		}
		var scheds []scheduler.Schedule
		if schedules != nil {
			scheds = schedules.Schedules()
		}

		var scored, unscanned []fleetRow
		for _, t := range domains {
			row := fleetRow{target: t}
			row.scheduleName, row.scheduleCron = matchSchedule(scheds, t.Domain, org.Slug)
			last, ok := lastScanByTarget[t.ID]
			if !ok {
				unscanned = append(unscanned, row)
				continue
			}
			row.hasScan = true
			row.lastScanAt = last.StartedAt
			assessments, err := st.ListAssessmentsForScan(ctx, last.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			row.score = BuildFleetScore(assessments, nil)
			switch prevScan, perr := st.PreviousScanForTarget(ctx, t.ID, last.StartedAt); {
			case perr == nil:
				prevAssessments, aerr := st.ListAssessmentsForScan(ctx, prevScan.ID)
				if aerr != nil {
					http.Error(w, aerr.Error(), http.StatusInternalServerError)
					return
				}
				row.delta = BuildFleetDelta(true, prevAssessments, assessments)
			case errors.Is(perr, store.ErrNotFound):
				row.delta = BuildFleetDelta(false, nil, assessments)
			default:
				http.Error(w, perr.Error(), http.StatusInternalServerError)
				return
			}
			scored = append(scored, row)
		}

		sortKey := normaliseFleetSort(r.URL.Query().Get("sort"))
		sortFleetRows(scored, sortKey)
		sort.Slice(unscanned, func(i, j int) bool { return unscanned[i].target.Domain < unscanned[j].target.Domain })

		view := fleetView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
			OrgSlug:      org.Slug,
			AllowEdit:    allowEdit,
			SortKey:      sortKey,
			ScopedOrganisation: &organisationLinkView{
				Slug: org.Slug,
				Name: org.Name,
				URL:  "/ui/orgs/" + org.Slug,
			},
		}
		for _, sk := range fleetSortKeys {
			view.SortLinks = append(view.SortLinks, fleetSortLinkView{
				Key:    sk.key,
				Label:  sk.label,
				URL:    "/ui/orgs/" + org.Slug + "/fleet?sort=" + sk.key,
				Active: sk.key == sortKey,
			})
		}
		for _, row := range append(scored, unscanned...) {
			view.Domains = append(view.Domains, row.toView())
		}
		render(w, tmpl, "fleet.tmpl", view)
	}
}

// normaliseFleetSort maps an arbitrary `?sort=` value to a known
// fleetSortKeys entry, defaulting to "domain" for anything else —
// including an empty value, so a bare /fleet request reads as the same
// alphabetical order run 02 shipped.
func normaliseFleetSort(key string) string {
	for _, sk := range fleetSortKeys {
		if sk.key == key {
			return key
		}
	}
	return "domain"
}

// sortFleetRows orders rows in place; every row has hasScan=true — the
// never-scanned bucket is sorted and appended separately in
// fleetHandler so it never re-enters the ranking.
func sortFleetRows(rows []fleetRow, key string) {
	switch key {
	case "score":
		// Ascending: the worst, most-actionable domains surface first.
		// The percentage is sort-only (proposal.md "Een percentage MAY
		// getoond worden om op te sorteren") — x/n stays the displayed
		// value.
		sort.SliceStable(rows, func(i, j int) bool {
			return fleetScorePercent(rows[i].score) < fleetScorePercent(rows[j].score)
		})
	case "change":
		// Ascending: domains that regressed the most surface first.
		sort.SliceStable(rows, func(i, j int) bool {
			return fleetChangeRank(rows[i].delta) < fleetChangeRank(rows[j].delta)
		})
	case "last_scan":
		// Descending: most recently scanned first.
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].lastScanAt.After(rows[j].lastScanAt)
		})
	default:
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].target.Domain < rows[j].target.Domain
		})
	}
}

// fleetScorePercent is the sort-only percentage the proposal sanctions.
// A domain with n=0 (nothing could be answered yet) sorts as the worst
// case rather than dividing by zero.
func fleetScorePercent(s FleetScore) float64 {
	if s.N == 0 {
		return 0
	}
	return float64(s.X) / float64(s.N)
}

// fleetChangeRank ranks a delta from "got worse" to "got better",
// ascending. A domain with no previous scan ranks as unchanged (0),
// alongside domains that scanned twice with no change — there is
// nothing to contrast it against either way.
func fleetChangeRank(d FleetDelta) int {
	if !d.HasPrevious {
		return 0
	}
	return d.XDelta
}

// toView formats a fleetRow for fleet.tmpl. A row with no scan yet
// carries only its domain and schedule — the zero-value score/delta
// fields would otherwise read as a genuine (and misleading) 0/0.
func (row fleetRow) toView() fleetDomainView {
	v := fleetDomainView{
		Domain:       row.target.Domain,
		ScheduleName: row.scheduleName,
		ScheduleCron: row.scheduleCron,
	}
	if row.hasScan {
		v.LastScanAt = row.lastScanAt.UTC().Format(time.RFC3339)
		v.HasScore = true
		v.X = row.score.X
		v.N = row.score.N
		v.Unanswered = row.score.Unanswered
		v.WorstFlow = row.score.WorstFlow
		v.WorstVerdict = row.score.WorstVerdict
		v.HasPrevious = row.delta.HasPrevious
		v.XDelta = row.delta.XDelta
		v.NDelta = row.delta.NDelta
		v.FlippedFlows = row.delta.FlippedFlows
	}
	return v
}

// matchSchedule finds the cron schedule that governs domain under
// orgSlug, straight from the scheduler's loaded config (spec.md "Het
// schema komt nu uit het schedules-bestand"). A Schedule with no
// Organisation field falls back, at run time (scheduler.go's
// makeJob), to the serve-config default org or the seeded "default"
// org — the UI has no access to that resolved default, so it
// approximates with models.DefaultOrganisationSlug; an operator
// running a custom `--organisation` default alongside org-less
// schedule entries is the one case this can mismatch.
func matchSchedule(scheds []scheduler.Schedule, domain, orgSlug string) (name, cron string) {
	for _, s := range scheds {
		if s.Target.Domain != domain {
			continue
		}
		owner := s.Organisation
		if owner == "" {
			owner = models.DefaultOrganisationSlug
		}
		if owner == orgSlug {
			return s.Name, s.Cron
		}
	}
	return "", ""
}

// fleetAddHandler is a sanctioned mutating route (POST
// /ui/orgs/{slug}/fleet/domains): adds a domain to the org's fleet
// without scanning it. Mounted only for a signed-in user (see
// Handler's allowFleetEdit).
func fleetAddHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slug := chi.URLParam(r, "slug")
		org, err := st.GetOrganisationBySlug(ctx, slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		domain := strings.TrimSpace(r.FormValue("domain"))
		if domain == "" {
			http.Error(w, "domain is required", http.StatusBadRequest)
			return
		}
		if _, err := st.AddFleetDomain(ctx, org.ID, domain); err != nil {
			http.Error(w, "invalid domain: "+err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/ui/orgs/"+org.Slug+"/fleet", http.StatusSeeOther)
	}
}

// fleetRemoveHandler is a sanctioned mutating route (POST
// /ui/orgs/{slug}/fleet/domains/{domain}/remove): takes a domain out
// of the org's fleet. The domain's scans and assessments are
// untouched (store.RemoveFleetDomain only stamps removed_at).
// Mounted only for a signed-in user (see Handler's allowFleetEdit).
func fleetRemoveHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slug := chi.URLParam(r, "slug")
		org, err := st.GetOrganisationBySlug(ctx, slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		domain := chi.URLParam(r, "domain")
		if err := st.RemoveFleetDomain(ctx, org.ID, domain); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/ui/orgs/"+org.Slug+"/fleet", http.StatusSeeOther)
	}
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

// answerView is the shape consumed by answer.tmpl — the progressive
// answer page (spec.md "The answer fills in while the scan runs").
// Refreshing gates the meta-refresh tag: true while the scan can still
// produce more findings, false once it is done, so the page stops
// polling on its own.
type answerView struct {
	ScanID        string
	Domain        string
	OrgSlug       string
	HasReporting  bool
	Refreshing    bool
	JustStarted   bool
	Headline      string
	Verdict       string // ja | nee | onbekend; empty when JustStarted
	X, N          int    // BuildFleetScore's x/n for this one domain (spec.md "naast de oordeelzin")
	Unanswered    int
	Flows         []FlowState
	AssessmentURL string // one link to the reasoning (run 04's page does not exist yet)
}

// answerHandler renders /ui/scans/{id}/answer from the findings
// persisted so far — not from a persisted Assessment, which may not
// exist yet while the scan is still running (spec.md "the answer page
// renders from the findings that exist at that moment"). It assesses
// scan.Findings in memory with the same wand rule pack the background
// scan uses; this is a read, not a new storage mechanism.
func answerHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
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
		subject := scan.ID
		if t, terr := st.GetTarget(r.Context(), scan.TargetID); terr == nil && t != nil && t.Domain != "" {
			subject = t.Domain
		}
		done := scan.Status != models.ScanStatusRunning
		view := answerView{
			ScanID:        scan.ID,
			Domain:        subject,
			OrgSlug:       scopeSlugForScan(r.Context(), st, scan.TargetID),
			HasReporting:  true,
			Refreshing:    !done,
			AssessmentURL: "/ui/scans/" + scan.ID + "/assessment",
		}
		if len(scan.Findings) == 0 && !done {
			// A scan that has not produced a single finding yet has not
			// failed to answer anything — it just started. Rendering it
			// through BuildAnswerVerdict would read as a flat "onbekend"
			// for every flow, indistinguishable from a scan that ran to
			// completion and genuinely measured nothing.
			view.JustStarted = true
			view.Headline = renderAnswerCopy("net_begonnen", nil)
		} else {
			findings, ferr := st.FindingsForAssessment(r.Context(), scan)
			if ferr != nil {
				http.Error(w, ferr.Error(), http.StatusInternalServerError)
				return
			}
			assessments := []models.Assessment{{
				Framework:  "wand",
				Dimensions: assessor.Assess(findings, wand.DefaultRules()),
			}}
			findingsByID := make(map[string]models.Finding, len(findings))
			for _, f := range findings {
				findingsByID[f.ID] = f
			}
			v := BuildAnswerVerdict(assessments, findingsByID)
			view.Verdict = v.Verdict
			view.Headline = v.Headline
			fs := BuildFleetScore(assessments, findingsByID)
			view.X = fs.X
			view.N = fs.N
			view.Unanswered = fs.Unanswered
			view.Flows = BuildFlowStates(assessments, done, subject, findingsByID)
		}
		render(w, tmpl, "answer.tmpl", view)
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
	Domain       string // the subject domain — the identity users recognise
	StartedAt    string
	Status       string
	OrgSlug      string
	HasReporting bool
	AnswerURL    string // one link back to the answer page (spec.md "The reasoning is one click from the answer")
	Flows        []Flow // hub-and-spoke diagram data — "what goes where"
	Diagram      Diagram
	FlowAnswers  []AccountabilityAnswer // the seven flows as answer-sheet rows
	Frameworks   []frameworkCardView
	// FrameworkRuleCount is the total number of rows (accountability
	// answers + dimension rationales) rendered inside Frameworks — the
	// count the collapsed-by-default <details> summary names, so
	// "wat eronder zit" is legible before the reader clicks (run 05
	// task 5.1).
	FrameworkRuleCount int
}

type frameworkCardView struct {
	Framework  string
	CreatedAt  string
	Dimensions []dimensionCardView
	// Accountability renders in place of a dimensionCardView for the
	// accountability dimension (design.md "UI direction": an answer
	// sheet, not a rule dump). Set only on the "wand" framework, only
	// when the Assessment carries the dimension.
	Accountability *accountabilityDimensionView
}

type dimensionCardView struct {
	Dimension       string
	Score           string
	Completeness    string
	Rationales      []rationaleRowView
	ScannerWarnings []string // operator-environment notices — never a property of the target
}

// accountabilityDimensionView is the accountability answer sheet for
// one framework card — spec.md "Accountability renders as an answer
// sheet, not a rule dump".
type accountabilityDimensionView struct {
	Score           string
	Completeness    string
	ScannerWarnings []string
	Answers         []AccountabilityAnswer
}

type rationaleRowView struct {
	CriteriumID string
	Score       string
	Verdict     string
	Description string
	Rationale   string
	Retired     bool
	Evidence    []string
	// Remediation is set only when Score is "afhankelijk" — the one
	// concrete handeling for this domain (run 05 task 5.2), from
	// accountability_nl.yaml's handelingen table.
	Remediation string
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
		findingsByID := make(map[string]models.Finding, len(scan.Findings))
		for _, f := range scan.Findings {
			findingsByID[f.ID] = f
		}
		flows := SovereigntyFlows(assessments, findingsByID)
		subject := scan.ID
		if t, terr := st.GetTarget(r.Context(), scan.TargetID); terr == nil && t != nil && t.Domain != "" {
			subject = t.Domain
		}
		view := assessmentView{
			ScanID:       scan.ID,
			Domain:       subject,
			StartedAt:    scan.StartedAt.UTC().Format(time.RFC3339),
			Status:       string(scan.Status),
			OrgSlug:      scopeSlugForScan(r.Context(), st, scan.TargetID),
			HasReporting: true,
			AnswerURL:    "/ui/scans/" + scan.ID + "/answer",
			Flows:        flows,
			Diagram:      SovereigntyDiagram(subject, flows),
		}
		view.FlowAnswers = BuildFlowAnswers(assessments, findingsByID, subject)
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
		ruleCount := 0
		for _, a := range assessments {
			fw := frameworkCardView{
				Framework: a.Framework,
				CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
			}
			for _, d := range a.Dimensions {
				if d.Dimension == models.DimensionAccountability {
					// Answer sheet, not a rule dump (design.md "UI
					// direction") — replaces the generic dimension card
					// entirely for this dimension.
					answers := BuildAccountabilityAnswers(d, findingsByID, subject)
					fw.Accountability = &accountabilityDimensionView{
						Score:           string(d.Score),
						Completeness:    string(d.Completeness),
						ScannerWarnings: DimensionScannerWarnings(d),
						Answers:         answers,
					}
					ruleCount += len(answers)
					continue
				}
				card := dimensionCardView{
					Dimension:       string(d.Dimension),
					Score:           string(d.Score),
					Completeness:    string(d.Completeness),
					ScannerWarnings: DimensionScannerWarnings(d),
				}
				for _, rationale := range d.Rationale {
					if isSovereigntyFlowRule(rationale.CriteriumID) {
						// Rendered exclusively via FlowAnswers above —
						// showing it again here would be a second,
						// rule-ID-in-the-open view of the same fact.
						continue
					}
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
					if rationale.Score == models.ScoreAfhankelijk {
						if h, ok := wand.HandelingFor(rationale.CriteriumID); ok {
							row.Remediation = fillParams(h, map[string]string{"domein": subject})
						}
					}
					card.Rationales = append(card.Rationales, row)
				}
				ruleCount += len(card.Rationales)
				fw.Dimensions = append(fw.Dimensions, card)
			}
			view.Frameworks = append(view.Frameworks, fw)
		}
		view.FrameworkRuleCount = ruleCount
		render(w, tmpl, "assessment.tmpl", view)
	}
}

func render(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

// trendsView is the shape consumed by trends.tmpl — the consolidated
// Farmer layer. Catalogue is the index (every rule + worst-score
// status hint); Matrix is the per-rule cross-target score counts.
type trendsView struct {
	GeneratedAt        string
	HasReporting       bool
	OrgSlug            string
	ScopedOrganisation *organisationLinkView
	Headline           headlineRenderView
	Targets            []dashboardTargetRow   // the fleet: one row per target, alphabetical — moved off the door (spec.md 2.1)
	Verdicts           []verdictRenderView    // per-framework "is this OK" pill
	FlowRollup         []FlowRollup           // Sovereignty-by-flow roll-up across targets
	OrganisationsList  []organisationLinkView // populated only on the instance-wide /ui/trends
	Catalogue          []ruleCatalogueRow
	Matrix             []reportingRowView
}

// trendsHandler renders /ui/trends — the single Farmer surface: rules
// across the fleet. It merges what used to be two tabs (the Analysis
// steering matrix and the Reporting rule catalogue) into one page so
// the nav can collapse to Overview + Trends.
func trendsHandler(st *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, scopedOrg, ok := resolveOrgQueryParam(ctx, st, w, r)
		if !ok {
			return
		}
		snaps, scans, err := buildSnapshots(ctx, st, orgID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		summary := RuleSummary(snaps, lookupRule)

		// Matrix: one row per rule that has fired, with per-score
		// distinct-target counts. Also index the counts for the
		// catalogue's status hint so we walk the summary once.
		type key struct{ fw, id string }
		byRule := make(map[key]map[models.Score]int, len(summary))
		matrix := make([]reportingRowView, 0, len(summary))
		for _, row := range summary {
			byRule[key{row.Framework, row.CriteriumID}] = row.Counts
			matrix = append(matrix, reportingRowView{
				Framework:        row.Framework,
				CriteriumID:      row.CriteriumID,
				Description:      row.Description,
				SoevereinCount:   row.Counts[models.ScoreSoeverein],
				VoldoendeCount:   row.Counts[models.ScoreVoldoende],
				AfhankelijkCount: row.Counts[models.ScoreAfhankelijk],
				OnbekendCount:    row.Counts[models.ScoreOnbekend],
			})
		}

		// Catalogue: every registered rule, with a worst-score status
		// hint when it has fired in scope.
		all := ListAllRules()
		catalogue := make([]ruleCatalogueRow, 0, len(all))
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
			catalogue = append(catalogue, row)
		}

		headline := BuildHeadline(snaps, scans)
		view := trendsView{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			HasReporting: true,
			Catalogue:    catalogue,
			Matrix:       matrix,
			Headline: headlineRenderView{
				TotalScans:       headline.TotalScans,
				PerimeterTargets: headline.PerimeterTargets,
				AgentHostTargets: headline.AgentHostTargets,
				Frameworks:       headline.Frameworks,
			},
			Targets:    buildFleetRows(snaps),
			FlowRollup: SovereigntyFlowRollup(snaps),
		}
		if !headline.LastScanAt.IsZero() {
			view.Headline.LastScanAt = headline.LastScanAt.UTC().Format(time.RFC3339)
		}
		for _, v := range WorstByFramework(snaps) {
			view.Verdicts = append(view.Verdicts, verdictRenderView{
				Framework: v.Framework,
				Score:     string(v.Score),
				AtWorst:   v.TargetsAtWorst,
				Total:     v.TotalAssessed,
			})
		}
		if scopedOrg != nil {
			view.OrgSlug = scopedOrg.Slug
			view.ScopedOrganisation = &organisationLinkView{
				Slug: scopedOrg.Slug,
				Name: scopedOrg.Name,
				URL:  "/ui/orgs/" + scopedOrg.Slug,
			}
		} else if orgs, listErr := st.ListOrganisations(ctx); listErr == nil {
			// The full org list is only meaningful unscoped — a scoped
			// view is already inside one organisation.
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
		render(w, tmpl, "trends.tmpl", view)
	}
}

// buildFleetRows is the fleet table shared by /ui/trends: every
// target with its last scan and a one-glance verdict, linking to the
// report. Moved here from the door (spec.md 2.1 — "the fleet table
// ... move[s] off this first screen").
func buildFleetRows(snaps []TargetSnapshot) []dashboardTargetRow {
	rows := make([]dashboardTargetRow, 0, len(snaps))
	for _, s := range snaps {
		row := dashboardTargetRow{
			Domain:     s.Domain,
			Kind:       string(s.Kind),
			LastStatus: s.LastStatus,
		}
		if !s.LastScanAt.IsZero() {
			row.LastScanAt = s.LastScanAt.UTC().Format(time.RFC3339)
		}
		if s.LastScanID != "" {
			row.ReportURL = "/ui/scans/" + s.LastScanID + "/assessment"
		}
		// Prefer the wand pack for the headline verdict; fall back to
		// whatever framework was assessed.
		var dims []models.DimensionScore
		if a, ok := s.Assessments["wand"]; ok {
			dims = a.Dimensions
		} else {
			for _, a := range s.Assessments {
				dims = a.Dimensions
				break
			}
		}
		verdict, covered := WorstScoreCovering(dims)
		if !(verdict == models.ScoreOnbekend && len(covered) == 0 && len(dims) == 0) {
			row.Verdict = string(verdict)
		}
		if len(covered) > 0 {
			row.VerdictDimensions = strings.Join(covered, ", ")
		}
		pill := AccountabilityPill(dims)
		row.AccountabilityLabel = pill.Label
		row.AccountabilityClass = pill.Class
		if row.ReportURL != "" && pill.Class != "unassessed" {
			row.AccountabilityLink = row.ReportURL + "#wand-accountability"
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Domain < rows[j].Domain })
	return rows
}

// redirectToTrends 302-redirects the retired /ui/analysis and
// /ui/reporting routes to /ui/trends, preserving the org scope so
// existing deep links survive the nav collapse.
func redirectToTrends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := "/ui/trends"
		if org := strings.TrimSpace(r.URL.Query().Get("org")); org != "" {
			target += "?org=" + url.QueryEscape(org)
		}
		http.Redirect(w, r, target, http.StatusFound)
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
	Observation        string
	// Thresholds mirrors the rule's assessor.Threshold list (run 04);
	// empty when the rule is a presence/absence check with no
	// boundary — the template shows a fixed explanatory line instead
	// of an empty heading (run 05 task 5.1).
	Thresholds []reportingThresholdView
	Rows       []reportingRuleRowView
}

type reportingThresholdView struct {
	Name        string
	Value       string
	Unit        string
	Explanation string
}

type reportingRuleRowView struct {
	TargetID string
	Domain   string
	ScanID   string
	Score    string
	Verdict  string
	When     string
	// Remediation is set only when Score is "afhankelijk" — the one
	// concrete handeling for this domain (run 05 task 5.2), from
	// accountability_nl.yaml's handelingen table.
	Remediation string
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
			Observation:  rule.Observation,
			Rows:         make([]reportingRuleRowView, 0, len(rows)),
		}
		for _, th := range rule.Thresholds {
			view.Thresholds = append(view.Thresholds, reportingThresholdView{
				Name:        th.Name,
				Value:       strconv.FormatFloat(th.Value, 'f', -1, 64),
				Unit:        th.Unit,
				Explanation: th.Explanation,
			})
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
			row := reportingRuleRowView{
				TargetID: rw.TargetID,
				Domain:   rw.Domain,
				ScanID:   rw.ScanID,
				Score:    string(rw.Score),
				Verdict:  rw.Verdict,
				When:     rw.When.UTC().Format(time.RFC3339),
			}
			if rw.Score == models.ScoreAfhankelijk {
				if h, ok := wand.HandelingFor(ruleID); ok {
					row.Remediation = fillParams(h, map[string]string{"domein": rw.Domain})
				}
			}
			view.Rows = append(view.Rows, row)
		}
		render(w, tmpl, "reporting_rule.tmpl", view)
	}
}
