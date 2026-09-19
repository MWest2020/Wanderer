// Package variants is the path-convergence probe. It attempts the 8
// paths apex/www x IPv4/IPv6 x http/https and reports, per path,
// whether they converge on one canonical HTTPS origin. Every hop
// passes the existing SSRF guard (internal/probe.IsPrivateOrMetadata);
// the probe never writes a second guard. A hard budget of 24
// connections per target bounds total network use across all 8 paths,
// sequential, no retries, inside the scan's per-probe timeout.
package variants

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	wprobe "github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

const (
	// maxRedirectsPerPath is the redirect depth cap per path (not
	// counting the initial fetch): at most 5 hops are followed before
	// a path that keeps redirecting is recorded unreachable.
	maxRedirectsPerPath = 5

	// connectionBudget is the hard cap on TCP connections opened
	// across all 8 paths for one target. DNS lookups and SSRF-guard
	// refusals don't consume it — only attempted dials do.
	connectionBudget = 24

	// StatusReachable etc. are the values recorded per path.
	StatusReachable         = "reachable"
	StatusUnreachable       = "unreachable"
	StatusRefused           = "refused"
	StatusNotFollowedBudget = "not_followed_budget"
	StatusNotTested         = "not_tested"

	// reasonScannerNoIPv6 mirrors internal/assessor.ReasonScannerNoIPv6.
	// Kept as a literal string here — probes must not import the
	// assessor package — the two must be kept in sync by hand.
	reasonScannerNoIPv6 = "scanner_no_ipv6"
)

// Resolver is the minimal DNS surface the probe needs. *net.Resolver
// satisfies it; tests inject a stub that returns canned, non-private
// addresses so the SSRF guard clears without touching real DNS.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Probe is the variants probe.
type Probe struct {
	// Resolver overrides DNS resolution. Nil means net.DefaultResolver.
	Resolver Resolver

	// Dial overrides the low-level connection used for each hop, after
	// the SSRF guard has cleared the resolved address. addr is always
	// a literal "ip:port" (already resolved and guard-checked). Nil
	// means a plain net.Dialer. Tests point every hop at one local
	// httptest listener regardless of which host/family it logically
	// belongs to.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)

	// HasIPv6 reports whether the scanner host has a local IPv6 route.
	// Nil means a real interface-based detector (see detectIPv6).
	// Evaluated once per Run, not once per path, so a test can drive
	// both branches deterministically.
	HasIPv6 func() bool

	// Timeout bounds each individual hop's HTTP round trip. Zero means
	// 10s.
	Timeout time.Duration

	// TLSClientConfig overrides the TLS configuration used for https
	// hops. Nil means Go's secure default (real certificate
	// verification). Tests inject a config trusting their httptest
	// server's certificate.
	TLSClientConfig *tls.Config
}

// New returns a variants probe with default settings.
func New() *Probe { return &Probe{} }

// ID implements probe.Probe.
func (*Probe) ID() string { return "variants" }

// pathSpec names one of the 8 paths. Order is fixed and deterministic
// so findings and connection-budget spend are reproducible.
type pathSpec struct {
	hostPart string // "apex" | "www"
	family   string // "v4" | "v6"
	scheme   string // "http" | "https"
}

var paths = []pathSpec{
	{"apex", "v4", "http"},
	{"apex", "v4", "https"},
	{"apex", "v6", "http"},
	{"apex", "v6", "https"},
	{"www", "v4", "http"},
	{"www", "v4", "https"},
	{"www", "v6", "http"},
	{"www", "v6", "https"},
}

// PathResult is the per-path evidence recorded in the http.variants
// finding.
type PathResult struct {
	HostPart    string   `json:"host_part"`
	Family      string   `json:"family"`
	Scheme      string   `json:"scheme"`
	Status      string   `json:"status"`
	Chain       []string `json:"chain,omitempty"`
	FinalOrigin string   `json:"final_origin,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, cfg wprobe.Config) ([]models.Finding, error) {
	domain := target.Domain
	if err := ctx.Err(); err != nil {
		// The scan's budget was already exhausted before this probe
		// got its turn: nothing below can succeed, so report total
		// failure rather than 8 misleading "unreachable" paths.
		return []models.Finding{unavailable(domain, err.Error())}, nil
	}

	hasIPv6 := p.HasIPv6
	if hasIPv6 == nil {
		hasIPv6 = detectIPv6
	}
	scannerHasIPv6 := hasIPv6()

	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = "Wanderer/0.x"
	}

	w := &walker{
		resolver:     resolver,
		dial:         p.Dial,
		allowPrivate: cfg.AllowPrivateTargets,
		ua:           ua,
		timeout:      timeout,
		tlsConfig:    p.TLSClientConfig,
		budget:       connectionBudget,
		verified:     map[string]map[string]string{},
	}

	results := make([]PathResult, 0, len(paths))
	for _, ps := range paths {
		if ps.family == "v6" && !scannerHasIPv6 {
			results = append(results, PathResult{
				HostPart: ps.hostPart,
				Family:   ps.family,
				Scheme:   ps.scheme,
				Status:   StatusNotTested,
				Reason:   reasonScannerNoIPv6,
			})
			continue
		}
		host := domain
		if ps.hostPart == "www" {
			host = "www." + domain
		}
		start := ps.scheme + "://" + host + "/"
		results = append(results, w.follow(ctx, ps, start))
	}

	return []models.Finding{{
		ProbeID:       "http.variants",
		DimensionHint: models.DimensionOperationeel,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"paths":             results,
			"connections_used":  connectionBudget - w.budget,
			"connection_budget": connectionBudget,
		},
	}}, nil
}

// walker carries the state shared across all 8 paths of one Run: the
// remaining connection budget and the set of origins already verified
// reachable, so a later path's chain can stop as soon as it lands on
// one of them. verified is keyed per address family first: a path
// dialed over v6 must never shortcut on an origin that was only ever
// reached over v4, and vice versa — the two families are verified
// independently even when they resolve to the same origin string.
type walker struct {
	resolver     Resolver
	dial         func(ctx context.Context, network, addr string) (net.Conn, error)
	allowPrivate bool
	ua           string
	timeout      time.Duration
	tlsConfig    *tls.Config
	budget       int
	verified     map[string]map[string]string // family -> origin -> origin
}

func origin(u *url.URL) string {
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// follow walks one path from start, hopping through redirects until it
// reaches a terminal response, an already-verified origin, a refused
// hop, a connection failure, the redirect-depth cap, or the exhausted
// connection budget.
func (w *walker) follow(ctx context.Context, ps pathSpec, start string) PathResult {
	res := PathResult{HostPart: ps.hostPart, Family: ps.family, Scheme: ps.scheme}
	cur, err := url.Parse(start)
	if err != nil {
		res.Status = StatusUnreachable
		return res
	}

	famVerified := w.verified[ps.family]
	if famVerified == nil {
		famVerified = map[string]string{}
		w.verified[ps.family] = famVerified
	}

	redirects := 0
	for {
		if o, ok := famVerified[origin(cur)]; ok {
			res.Chain = append(res.Chain, cur.String())
			res.FinalOrigin = o
			res.Status = StatusReachable
			return res
		}
		if w.budget <= 0 {
			res.Status = StatusNotFollowedBudget
			return res
		}

		ip, port, refused, rerr := w.resolveAndGuard(ctx, cur, ps.family)
		if rerr != nil {
			res.Status = StatusUnreachable
			return res
		}
		if refused {
			res.Chain = append(res.Chain, cur.String())
			res.Status = StatusRefused
			return res
		}

		w.budget--
		status, location, herr := w.doHop(ctx, cur, ip, port)
		res.Chain = append(res.Chain, cur.String())
		if herr != nil {
			res.Status = StatusUnreachable
			return res
		}

		if status >= 300 && status < 400 && location != "" {
			if redirects >= maxRedirectsPerPath {
				res.Status = StatusUnreachable
				return res
			}
			loc, perr := url.Parse(location)
			if perr != nil {
				res.Status = StatusUnreachable
				return res
			}
			redirects++
			cur = cur.ResolveReference(loc)
			continue
		}

		res.FinalOrigin = origin(cur)
		res.Status = StatusReachable
		famVerified[res.FinalOrigin] = res.FinalOrigin
		return res
	}
}

// resolveAndGuard resolves u's host to the address family ps requires
// and applies the existing SSRF guard (probe.IsPrivateOrMetadata) to
// every candidate. It never dials.
func (w *walker) resolveAndGuard(ctx context.Context, u *url.URL, family string) (ip, port string, refused bool, err error) {
	host := u.Hostname()
	port = u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	var candidates []net.IP
	if literal := net.ParseIP(host); literal != nil {
		candidates = []net.IP{literal}
	} else {
		addrs, lerr := w.resolver.LookupIPAddr(ctx, host)
		if lerr != nil {
			return "", "", false, lerr
		}
		for _, a := range addrs {
			candidates = append(candidates, a.IP)
		}
	}

	var familyMatched []net.IP
	for _, c := range candidates {
		isV4 := c.To4() != nil
		if (family == "v4") == isV4 {
			familyMatched = append(familyMatched, c)
		}
	}
	if len(familyMatched) == 0 {
		return "", "", false, fmt.Errorf("variants: no %s address for %s", family, host)
	}

	var allowed []net.IP
	for _, c := range familyMatched {
		if w.allowPrivate || !wprobe.IsPrivateOrMetadata(c) {
			allowed = append(allowed, c)
		}
	}
	if len(allowed) == 0 {
		return "", "", true, nil
	}
	return allowed[0].String(), port, false, nil
}

// doHop issues exactly one GET at u, dialing ip:port directly (the
// address resolveAndGuard already cleared) and reports the response's
// status and Location header without following the redirect itself —
// the caller owns hop accounting and the budget.
func (w *walker) doHop(ctx context.Context, u *url.URL, ip, port string) (status int, location string, err error) {
	dial := w.dial
	if dial == nil {
		d := &net.Dialer{Timeout: w.timeout}
		dial = d.DialContext
	}
	addr := net.JoinHostPort(ip, port)
	client := &http.Client{
		Timeout: w.timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return dial(ctx, network, addr)
			},
			TLSClientConfig: w.tlsConfig,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if rerr != nil {
		return 0, "", rerr
	}
	req.Header.Set("User-Agent", w.ua)
	resp, derr := client.Do(req)
	if derr != nil {
		return 0, "", derr
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location"), nil
}

func unavailable(domain, reason string) models.Finding {
	return models.Finding{
		ProbeID:  "http.variants.unavailable",
		Subject:  domain,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"reason": reason,
		},
	}
}
