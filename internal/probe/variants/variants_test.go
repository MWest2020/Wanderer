package variants_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	wprobe "github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/probe/variants"
	"github.com/MWest2020/wanderer/pkg/models"
)

// stubResolver hands back one public-looking v4 and one public-looking
// v6 address for any host, so the SSRF guard clears without touching
// real DNS. 203.0.113.0/24 and 2001:db8::/32 are documentation ranges
// deliberately absent from probe.IsPrivateOrMetadata's block list.
type stubResolver struct{}

func (stubResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	return []net.IPAddr{
		{IP: net.ParseIP("203.0.113.10")},
		{IP: net.ParseIP("2001:db8::1")},
	}, nil
}

// routeByPort ignores the resolved address entirely and routes every
// hop to one of two local httptest listeners, chosen by the requested
// port (80 -> plain, 443 -> TLS) — the same signal doHop uses when it
// builds the default port for a scheme. No real network is touched.
func routeByPort(plainAddr, tlsAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		target := plainAddr
		if port == "443" {
			target = tlsAddr
		}
		var d net.Dialer
		return d.DialContext(ctx, network, target)
	}
}

func alwaysErrorDial(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("variants test: dial disabled")
}

func findingByProbeID(findings []models.Finding, id string) *models.Finding {
	for i := range findings {
		if findings[i].ProbeID == id {
			return &findings[i]
		}
	}
	return nil
}

func pathsOf(t *testing.T, f *models.Finding) []variants.PathResult {
	t.Helper()
	paths, ok := f.Attributes["paths"].([]variants.PathResult)
	if !ok {
		t.Fatalf("attributes[paths] has type %T, want []variants.PathResult", f.Attributes["paths"])
	}
	return paths
}

func findPath(paths []variants.PathResult, hostPart, family, scheme string) *variants.PathResult {
	for i := range paths {
		p := &paths[i]
		if p.HostPart == hostPart && p.Family == family && p.Scheme == scheme {
			return p
		}
	}
	return nil
}

// TestRun_ConvergingPaths covers 6.1: all eight paths eventually land
// on one canonical HTTPS origin, and the "already verified" shortcut
// keeps the connection count far under the 24-connection budget.
func TestRun_ConvergingPaths(t *testing.T) {
	const canonical = "www.voorbeeld.nl"

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+canonical+"/", http.StatusMovedPermanently)
	}))
	defer plain.Close()

	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == canonical {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "https://"+canonical+"/", http.StatusMovedPermanently)
	}))
	defer tlsSrv.Close()

	p := &variants.Probe{
		Resolver:        stubResolver{},
		Dial:            routeByPort(plain.Listener.Addr().String(), tlsSrv.Listener.Addr().String()),
		HasIPv6:         func() bool { return true },
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test-only, self-signed httptest cert
	}

	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.variants")
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}
	if f.DimensionHint != models.DimensionOperationeel {
		t.Errorf("dimension = %q, want operationeel", f.DimensionHint)
	}

	paths := pathsOf(t, f)
	if len(paths) != 8 {
		t.Fatalf("got %d paths, want 8", len(paths))
	}
	for _, pr := range paths {
		if pr.Status != variants.StatusReachable {
			t.Errorf("path %+v: status = %q, want reachable", pr, pr.Status)
			continue
		}
		if pr.FinalOrigin != "https://"+canonical {
			t.Errorf("path %+v: final_origin = %q, want https://%s", pr, pr.FinalOrigin, canonical)
		}
	}

	used, _ := f.Attributes["connections_used"].(int)
	if used == 0 || used >= 24 {
		t.Errorf("connections_used = %d, want in (0, 24) — the verified-origin shortcut should keep this well under budget", used)
	}
}

// TestRun_BudgetExhausted covers 6.1: paths that would each burn a
// full 5-redirect chain must not push the total past 24 connections;
// paths that can't even start are recorded not_followed_budget.
func TestRun_BudgetExhausted(t *testing.T) {
	var dialCount int
	loop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirects forever within the same server, never converging,
		// so every path is forced through the full redirect-depth cap
		// (5 redirects, 6 connections) unless the budget runs out
		// first.
		http.Redirect(w, r, "http://voorbeeld.nl/next", http.StatusFound)
	}))
	defer loop.Close()

	dial := routeByPort(loop.Listener.Addr().String(), loop.Listener.Addr().String())
	countingDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialCount++
		return dial(ctx, network, addr)
	}

	p := &variants.Probe{
		Resolver: stubResolver{},
		Dial:     countingDial,
		HasIPv6:  func() bool { return true },
	}

	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.variants")
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}

	if dialCount > 24 {
		t.Fatalf("dialCount = %d, want <= 24 (the hard connection budget)", dialCount)
	}
	used, _ := f.Attributes["connections_used"].(int)
	if used != dialCount {
		t.Errorf("connections_used = %d, want %d (== actual dials)", used, dialCount)
	}
	if used > 24 {
		t.Errorf("connections_used = %d, want <= 24", used)
	}

	paths := pathsOf(t, f)
	var budgetSkipped int
	for _, pr := range paths {
		if pr.Status == variants.StatusNotFollowedBudget {
			budgetSkipped++
		}
	}
	if budgetSkipped == 0 {
		t.Error("expected at least one path recorded as not_followed_budget once the budget ran out")
	}
}

// TestRun_PrivateRedirectRefused covers 6.2: a redirect chain pointing
// at 10.0.0.5 must be refused by the existing SSRF guard and never
// followed.
func TestRun_PrivateRedirectRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://10.0.0.5/", http.StatusFound)
	}))
	defer srv.Close()

	p := &variants.Probe{
		Resolver: stubResolver{},
		Dial:     routeByPort(srv.Listener.Addr().String(), srv.Listener.Addr().String()),
		HasIPv6:  func() bool { return true },
	}

	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.variants")
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}

	paths := pathsOf(t, f)
	pr := findPath(paths, "apex", "v4", "http")
	if pr == nil {
		t.Fatalf("no apex/v4/http path in %+v", paths)
	}
	if pr.Status != variants.StatusRefused {
		t.Errorf("status = %q, want refused", pr.Status)
	}
	if len(pr.Chain) == 0 || pr.Chain[len(pr.Chain)-1] != "http://10.0.0.5/" {
		t.Errorf("chain = %v, want it to end at the refused hop http://10.0.0.5/", pr.Chain)
	}
	if pr.FinalOrigin != "" {
		t.Errorf("final_origin = %q, want empty for a refused path", pr.FinalOrigin)
	}
}

// TestRun_NoLocalIPv6 covers 6.3: when the scanner has no IPv6 route,
// the four v6 paths are not_tested with reason scanner_no_ipv6, and
// the capability is checked once, not per path.
func TestRun_NoLocalIPv6(t *testing.T) {
	var calls int
	hasIPv6 := func() bool {
		calls++
		return false
	}

	p := &variants.Probe{
		Resolver: stubResolver{},
		Dial:     alwaysErrorDial,
		HasIPv6:  hasIPv6,
	}

	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 1 {
		t.Errorf("HasIPv6 called %d times, want exactly 1 (once per scan, not per path)", calls)
	}

	f := findingByProbeID(findings, "http.variants")
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}
	paths := pathsOf(t, f)
	var v6Count int
	for _, pr := range paths {
		if pr.Family != "v6" {
			continue
		}
		v6Count++
		if pr.Status != variants.StatusNotTested {
			t.Errorf("path %+v: status = %q, want not_tested", pr, pr.Status)
		}
		if pr.Reason != "scanner_no_ipv6" {
			t.Errorf("path %+v: reason = %q, want scanner_no_ipv6", pr, pr.Reason)
		}
	}
	if v6Count != 4 {
		t.Fatalf("got %d v6 paths, want 4", v6Count)
	}
}

// TestRun_ShortcutDoesNotCrossFamily covers 6.5: the "already verified
// origin" shortcut must not let a v6 path report reachable on the
// strength of a v4 dial. With a stub where every v4 dial reaches the
// server and every v6 dial fails outright, the four v4 paths must
// converge and report reachable while the four v6 paths — which share
// the same host/origin strings as their v4 counterparts — must never
// borrow that v4 verification.
func TestRun_ShortcutDoesNotCrossFamily(t *testing.T) {
	const canonical = "www.voorbeeld.nl"

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+canonical+"/", http.StatusMovedPermanently)
	}))
	defer plain.Close()

	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == canonical {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "https://"+canonical+"/", http.StatusMovedPermanently)
	}))
	defer tlsSrv.Close()

	// dialByFamily routes by the resolved IP's family (not by port,
	// unlike routeByPort above): v4-resolved addresses reach the local
	// httptest servers, v6-resolved addresses always fail, simulating a
	// scanner that only has a working v4 route to this target.
	dialByFamily := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.To4() == nil {
			return nil, errors.New("variants test: v6 unreachable")
		}
		target := plain.Listener.Addr().String()
		if port == "443" {
			target = tlsSrv.Listener.Addr().String()
		}
		var d net.Dialer
		return d.DialContext(ctx, network, target)
	}

	p := &variants.Probe{
		Resolver:        stubResolver{},
		Dial:            dialByFamily,
		HasIPv6:         func() bool { return true },
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test-only, self-signed httptest cert
	}

	findings, err := p.Run(context.Background(), models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	f := findingByProbeID(findings, "http.variants")
	if f == nil {
		t.Fatalf("no http.variants finding, got %+v", findings)
	}

	paths := pathsOf(t, f)
	var v4Reachable, v6NotReachable int
	for _, pr := range paths {
		switch pr.Family {
		case "v4":
			if pr.Status == variants.StatusReachable {
				v4Reachable++
			} else {
				t.Errorf("v4 path %+v: status = %q, want reachable", pr, pr.Status)
			}
		case "v6":
			if pr.Status == variants.StatusReachable {
				t.Errorf("v6 path %+v: status = reachable, but no v6 dial ever succeeded — shortcut crossed families", pr)
			} else {
				v6NotReachable++
			}
		}
	}
	if v4Reachable != 4 {
		t.Errorf("got %d reachable v4 paths, want 4", v4Reachable)
	}
	if v6NotReachable != 4 {
		t.Errorf("got %d non-reachable v6 paths, want 4", v6NotReachable)
	}
}

// TestRun_TotalFailure covers the http.variants.unavailable case: the
// scan's own budget was already exhausted before this probe's turn.
func TestRun_TotalFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := variants.New()
	findings, err := p.Run(ctx, models.Target{Domain: "voorbeeld.nl"}, wprobe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(findings) != 1 || findings[0].ProbeID != "http.variants.unavailable" {
		t.Fatalf("expected single http.variants.unavailable finding, got %+v", findings)
	}
}
