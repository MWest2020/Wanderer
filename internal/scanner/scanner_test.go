package scanner_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

type stubProbe struct {
	id       string
	findings []models.Finding
	err      error
	panic    bool
	sleep    time.Duration

	gotTarget models.Target // last target seen on Run
}

func (s *stubProbe) ID() string { return s.id }
func (s *stubProbe) Run(ctx context.Context, t models.Target, _ probe.Config) ([]models.Finding, error) {
	s.gotTarget = t
	if s.panic {
		panic("stub panic")
	}
	if s.sleep > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.sleep):
		}
	}
	return s.findings, s.err
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func mkFinding(probeID string) models.Finding {
	return models.Finding{
		ProbeID:    probeID,
		Subject:    "example.nl",
		Severity:   models.SeverityInfo,
		Attributes: map[string]any{},
	}
}

func TestScanAllProbesSucceed(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}},
		&stubProbe{id: "b", findings: []models.Finding{mkFinding("b.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusComplete {
		t.Errorf("status = %q, want complete", res.Status)
	}
	// 2 probe findings + the config.expected_registrant finding the
	// scanner always records for the scan's organisation.
	if len(res.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(res.Findings))
	}
}

func TestScanOneProbeFailsScanContinues(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}},
		&stubProbe{id: "b", err: errors.New("boom")},
		&stubProbe{id: "c", findings: []models.Finding{mkFinding("c.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusPartial {
		t.Errorf("status = %q, want partial", res.Status)
	}
	var sawA, sawC, sawBErr bool
	for _, f := range res.Findings {
		switch f.ProbeID {
		case "a.ok":
			sawA = true
		case "c.ok":
			sawC = true
		case "b.error":
			sawBErr = true
		}
	}
	if !sawA || !sawC {
		t.Errorf("other probes did not run: a=%v c=%v", sawA, sawC)
	}
	if !sawBErr {
		t.Error("failing probe did not produce a .error finding")
	}
}

func TestScanPanicContained(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "a", panic: true},
		&stubProbe{id: "b", findings: []models.Finding{mkFinding("b.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status != models.ScanStatusPartial {
		t.Errorf("status = %q, want partial", res.Status)
	}
	var sawPanic, sawB bool
	for _, f := range res.Findings {
		if f.ProbeID == "a.panic" {
			sawPanic = true
		}
		if f.ProbeID == "b.ok" {
			sawB = true
		}
	}
	if !sawPanic {
		t.Error("panic did not produce a .panic finding")
	}
	if !sawB {
		t.Error("probe after panic did not run")
	}
}

func TestScanTimeoutIsNotFatal(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{
		&stubProbe{id: "slow", sleep: 100 * time.Millisecond},
		&stubProbe{id: "fast", findings: []models.Finding{mkFinding("fast.ok")}},
	}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: 10 * time.Millisecond})
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	// Timeout alone is not a hard failure — scan should be complete.
	if res.Status != models.ScanStatusComplete {
		t.Errorf("status = %q, want complete (timeout is not a hard fail)", res.Status)
	}
	var sawTimeout bool
	for _, f := range res.Findings {
		if f.ProbeID == "slow.timeout" {
			sawTimeout = true
		}
	}
	if !sawTimeout {
		t.Error("timeout did not produce a .timeout finding")
	}
}

// TestIPProbeReceivesDiscoveredHosts pins the contract that drives the
// juridisch correlation rules: hosts discovered by DNS (MX) and HTTP
// (third parties) MUST be appended to target.Related before the IP
// probe runs. Without this, mx_vendor_jurisdiction / third_parties_eea
// silently return Onbekend on real scans even though the unit tests
// pass.
func TestIPProbeReceivesDiscoveredHosts(t *testing.T) {
	s := newStore(t)
	dns := &stubProbe{
		id: "dns",
		findings: []models.Finding{
			{
				ProbeID:    "dns.mx",
				Subject:    "example.nl",
				Severity:   models.SeverityObservation,
				Attributes: map[string]any{"host": "mail.fastmail.com."},
			},
			{
				ProbeID:    "dns.ns",
				Subject:    "example.nl",
				Severity:   models.SeverityObservation,
				Attributes: map[string]any{"host": "ns1.transip.net."},
			},
		},
	}
	http := &stubProbe{
		id: "http",
		findings: []models.Finding{
			{
				ProbeID:  "http.third_party",
				Subject:  "tracker.example.com",
				Severity: models.SeverityObservation,
			},
			{
				ProbeID:  "http.third_party",
				Subject:  "MAIL.FASTMAIL.COM", // duplicate of MX host with different casing
				Severity: models.SeverityObservation,
			},
		},
	}
	ip := &stubProbe{id: "ip"}
	other := &stubProbe{id: "tls"}

	sc := scanner.New(s, []probe.Probe{dns, other, http, ip}, probe.Config{PerProbeTimeout: time.Second})
	if _, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"}); err != nil {
		t.Fatalf("scan: %v", err)
	}

	want := []string{"mail.fastmail.com", "tracker.example.com", "ns1.transip.net"}
	if diff := relatedDiff(ip.gotTarget.Related, want); diff != "" {
		t.Errorf("IP probe Related: %s", diff)
	}
	// Probes other than IP must NOT see the enriched Related list — we
	// only enrich for the probe that needs it. (Otherwise DNS would
	// resolve every third party as if it were the target, which is not
	// what those probes are for.)
	if len(other.gotTarget.Related) != 0 {
		t.Errorf("non-IP probe should see original Related (got %v)", other.gotTarget.Related)
	}
	if len(dns.gotTarget.Related) != 0 {
		t.Errorf("DNS probe should see original Related (got %v)", dns.gotTarget.Related)
	}
}

// relatedDiff returns a non-empty string describing how got differs
// from want (order-independent), or "" if they match as multisets.
func relatedDiff(got, want []string) string {
	gotSet := map[string]bool{}
	for _, h := range got {
		gotSet[h] = true
	}
	wantSet := map[string]bool{}
	for _, h := range want {
		wantSet[h] = true
	}
	var missing, extra []string
	for h := range wantSet {
		if !gotSet[h] {
			missing = append(missing, h)
		}
	}
	for h := range gotSet {
		if !wantSet[h] {
			extra = append(extra, h)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	return "missing=" + fmt.Sprint(missing) + " extra=" + fmt.Sprint(extra)
}

func nsHostFinding(domain, host string) models.Finding {
	return models.Finding{
		ProbeID:    "dns.ns",
		Subject:    domain,
		Severity:   models.SeverityObservation,
		Attributes: map[string]any{"host": host},
	}
}

// TestScan_NSHolderTwoProvidersOneLookupEach pins the scanner spec
// scenario "Two providers, three nameservers": ns1/ns2.provider-a.nl
// and ns1.provider-b.eu share the RDAP endpoint used in this test, so
// the scanner MUST dedupe to their two distinct registrable domains
// and issue exactly two RDAP lookups (N nameservers under one
// provider cost one lookup), producing exactly two whois.ns_holder
// findings.
func TestScan_NSHolderTwoProvidersOneLookupEach(t *testing.T) {
	s := newStore(t)
	requests := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write([]byte(`{"entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Some Registrant"]]]}]}`))
	}))
	defer srv.Close()

	dns := &stubProbe{
		id: "dns",
		findings: []models.Finding{
			nsHostFinding("example.nl", "ns1.provider-a.nl"),
			nsHostFinding("example.nl", "ns2.provider-a.nl"),
			nsHostFinding("example.nl", "ns1.provider-b.eu"),
		},
	}
	sc := scanner.New(s, []probe.Probe{dns}, probe.Config{PerProbeTimeout: time.Second})
	sc.RDAPBaseURL = srv.URL + "/domain/"

	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(requests) != 2 {
		t.Errorf("RDAP lookups = %d (%v), want exactly 2", len(requests), requests)
	}
	var holders int
	for _, f := range res.Findings {
		if f.ProbeID == "whois.ns_holder" {
			holders++
		}
	}
	if holders != 2 {
		t.Errorf("whois.ns_holder findings = %d, want 2", holders)
	}
}

// TestScan_NSHolderUnavailableOnNoRDAPTLD pins the scenario "Registry
// without RDAP": a 404 from the bootstrap (no RDAP service for the
// nameserver's TLD) emits whois.ns_holder.unavailable for that domain
// and the scan continues (status is not failed).
func TestScan_NSHolderUnavailableOnNoRDAPTLD(t *testing.T) {
	s := newStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dns := &stubProbe{
		id: "dns",
		findings: []models.Finding{
			nsHostFinding("example.nl", "ns1.no-rdap-tld.zz"),
		},
	}
	sc := scanner.New(s, []probe.Probe{dns}, probe.Config{PerProbeTimeout: time.Second})
	sc.RDAPBaseURL = srv.URL + "/domain/"

	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Status == models.ScanStatusFailed {
		t.Errorf("status = %q, want scan to continue past the ns_holder failure", res.Status)
	}
	var sawUnavailable bool
	for _, f := range res.Findings {
		if f.ProbeID == "whois.ns_holder.unavailable" && f.Subject == "no-rdap-tld.zz" {
			sawUnavailable = true
		}
	}
	if !sawUnavailable {
		t.Errorf("expected whois.ns_holder.unavailable for no-rdap-tld.zz; got %+v", res.Findings)
	}
}

func TestScanInvalidDomain(t *testing.T) {
	s := newStore(t)
	sc := scanner.New(s, []probe.Probe{&stubProbe{id: "a"}}, probe.Config{})
	_, err := sc.Scan(context.Background(), models.Target{Domain: ""})
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}

// findExpectedRegistrant returns the names attribute of the
// config.expected_registrant finding, or nil + false if the scan
// carries none. Accepts both the []string the scanner constructs
// in-memory and the []interface{} a Finding decodes to once it has
// round-tripped through the store's JSON-encoded attributes column.
func findExpectedRegistrant(findings []models.Finding) ([]string, bool) {
	for _, f := range findings {
		if f.ProbeID != "config.expected_registrant" {
			continue
		}
		switch v := f.Attributes["expected_registrant"].(type) {
		case []string:
			return v, true
		case []interface{}:
			out := make([]string, len(v))
			for i, e := range v {
				out[i], _ = e.(string)
			}
			return out, true
		default:
			return nil, true
		}
	}
	return nil, false
}

func TestScan_RecordsExpectedRegistrantForScanOrganisation(t *testing.T) {
	s := newStore(t)
	org := &models.Organisation{Slug: "voorbeeld", Name: "Gemeente Voorbeeld", ExpectedRegistrant: []string{"Gemeente Voorbeeld"}}
	if err := s.UpsertOrganisation(context.Background(), org); err != nil {
		t.Fatalf("upsert organisation: %v", err)
	}
	probes := []probe.Probe{&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}}}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})

	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl", OrganisationID: org.ID})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	names, ok := findExpectedRegistrant(res.Findings)
	if !ok {
		t.Fatal("no config.expected_registrant finding recorded")
	}
	if len(names) != 1 || names[0] != "Gemeente Voorbeeld" {
		t.Errorf("expected_registrant = %v, want [Gemeente Voorbeeld]", names)
	}
}

func TestScan_RecordsExpectedRegistrantAsEmptyListWhenOrganisationDeclaresNone(t *testing.T) {
	s := newStore(t)
	probes := []probe.Probe{&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}}}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})

	// No organisation set on the target: the target falls back to the
	// default organisation, which declares no expected registrant.
	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	names, ok := findExpectedRegistrant(res.Findings)
	if !ok {
		t.Fatal("no config.expected_registrant finding recorded")
	}
	if names == nil || len(names) != 0 {
		t.Errorf("expected_registrant = %v, want an empty (non-nil) list", names)
	}
}

// TestScan_ChangingOrganisationListDoesNotRewriteHistory pins the
// scanner spec scenario "Changing the list does not rewrite history":
// a scan's config.expected_registrant finding is fixed at scan time,
// so a later change to the organisation's declared names must not
// alter what a stored scan (and any re-assessment of it) sees.
func TestScan_ChangingOrganisationListDoesNotRewriteHistory(t *testing.T) {
	s := newStore(t)
	org := &models.Organisation{Slug: "voorbeeld", Name: "Gemeente Voorbeeld", ExpectedRegistrant: []string{"Gemeente Voorbeeld"}}
	if err := s.UpsertOrganisation(context.Background(), org); err != nil {
		t.Fatalf("upsert organisation: %v", err)
	}
	probes := []probe.Probe{&stubProbe{id: "a", findings: []models.Finding{mkFinding("a.ok")}}}
	sc := scanner.New(s, probes, probe.Config{PerProbeTimeout: time.Second})

	res, err := sc.Scan(context.Background(), models.Target{Domain: "example.nl", OrganisationID: org.ID})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	// The organisation's declared names change after the scan ran.
	updated := &models.Organisation{Slug: "voorbeeld", Name: "Gemeente Voorbeeld", ExpectedRegistrant: []string{"Gemeente Nieuw"}}
	if err := s.UpsertOrganisation(context.Background(), updated); err != nil {
		t.Fatalf("update organisation: %v", err)
	}

	stored, err := s.GetScan(context.Background(), res.ID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	names, ok := findExpectedRegistrant(stored.Findings)
	if !ok {
		t.Fatal("no config.expected_registrant finding on stored scan")
	}
	if len(names) != 1 || names[0] != "Gemeente Voorbeeld" {
		t.Errorf("stored scan's expected_registrant = %v, want [Gemeente Voorbeeld] (recorded at scan time)", names)
	}
}
