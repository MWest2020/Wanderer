package whois

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

const sampleRDAP = `{
  "objectClassName": "domain",
  "ldhName": "example.nl",
  "entities": [
    {
      "roles": ["registrant"],
      "vcardArray": ["vcard", [
        ["version", {}, "text", "4.0"],
        ["adr", {"cc": "NL"}, "text", ["", "", "Strawinskylaan 1", "Amsterdam", "", "1077XX", "Netherlands"]]
      ]]
    },
    {
      "roles": ["registrar"],
      "vcardArray": ["vcard", [
        ["fn", {}, "text", "TransIP B.V."]
      ]]
    }
  ]
}`

const redactedRDAP = `{
  "objectClassName": "domain",
  "ldhName": "example.nl",
  "entities": [
    {
      "roles": ["registrant"],
      "vcardArray": ["vcard", [
        ["fn", {}, "text", "REDACTED FOR PRIVACY"]
      ]]
    }
  ]
}`

// runFixture starts an httptest server returning body for any request
// and runs the probe against it, returning the resulting Findings.
func runFixture(t *testing.T, body []byte) []models.Finding {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return got
}

func findFinding(t *testing.T, findings []models.Finding, probeID string) models.Finding {
	t.Helper()
	for _, f := range findings {
		if f.ProbeID == probeID {
			return f
		}
	}
	t.Fatalf("missing finding %s (got %+v)", probeID, findings)
	return models.Finding{}
}

func hasFinding(findings []models.Finding, probeID string) bool {
	for _, f := range findings {
		if f.ProbeID == probeID {
			return true
		}
	}
	return false
}

func TestRun_HappyPath(t *testing.T) {
	got := runFixture(t, []byte(sampleRDAP))

	registrant := findFinding(t, got, "whois.registrant")
	if registrant.Attributes["country"] != "NL" {
		t.Errorf("country = %v", registrant.Attributes["country"])
	}
	registrar := findFinding(t, got, "whois.registrar")
	if registrar.Attributes["name"] != "TransIP B.V." {
		t.Errorf("name = %v", registrar.Attributes["name"])
	}

	// No fn on the registrant vcard: registrant_identity reports the
	// name as absent, not silently omitted.
	identity := findFinding(t, got, "whois.registrant_identity")
	if identity.Attributes["name"] != "absent" {
		t.Errorf("registrant_identity.name = %v, want absent", identity.Attributes["name"])
	}
	if identity.DimensionHint != models.DimensionAccountability {
		t.Errorf("registrant_identity.DimensionHint = %v", identity.DimensionHint)
	}

	reseller := findFinding(t, got, "whois.reseller")
	if reseller.Attributes["present"] != false || reseller.Attributes["name"] != "absent" {
		t.Errorf("reseller = %+v, want present=false name=absent", reseller.Attributes)
	}

	status := findFinding(t, got, "whois.status")
	codes, _ := status.Attributes["codes"].([]string)
	if len(codes) != 0 {
		t.Errorf("status.codes = %v, want empty", codes)
	}

	expiry := findFinding(t, got, "whois.expiry")
	if expiry.Attributes["present"] != false || expiry.Attributes["date"] != "absent" {
		t.Errorf("expiry = %+v, want present=false date=absent", expiry.Attributes)
	}
}

func TestRun_RedactedVcard(t *testing.T) {
	got := runFixture(t, []byte(redactedRDAP))

	identity := findFinding(t, got, "whois.registrant_identity")
	if identity.Attributes["name"] != "REDACTED FOR PRIVACY" {
		t.Errorf("name = %v", identity.Attributes["name"])
	}
	if identity.Attributes["privacy_proxy"] != true {
		t.Errorf("privacy_proxy = %v, want true", identity.Attributes["privacy_proxy"])
	}
}

// TestRun_ResellerNestedUnderRegistrar pins the scanner spec scenario
// "Reseller nested under the registrar": the reseller entity only
// appears inside the registrar entity's own `entities` array, so
// finding it requires recursive entity parsing. It also exercises the
// "Full RDAP response" scenario (registrant vcard, reseller, status
// codes, expiration event all present).
func TestRun_ResellerNestedUnderRegistrar(t *testing.T) {
	body, err := os.ReadFile("testdata/rdap-reseller-nested.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got := runFixture(t, body)

	reseller := findFinding(t, got, "whois.reseller")
	if reseller.Attributes["present"] != true {
		t.Fatalf("reseller.present = %v, want true", reseller.Attributes["present"])
	}
	if reseller.Attributes["name"] != "Local Hosting Reseller" {
		t.Errorf("reseller.name = %v", reseller.Attributes["name"])
	}

	identity := findFinding(t, got, "whois.registrant_identity")
	if identity.Attributes["name"] != "Gemeente Voorbeeld" {
		t.Errorf("identity.name = %v", identity.Attributes["name"])
	}
	if identity.Attributes["kind"] != "org" {
		t.Errorf("identity.kind = %v", identity.Attributes["kind"])
	}
	if identity.Attributes["privacy_proxy"] != false {
		t.Errorf("identity.privacy_proxy = %v, want false", identity.Attributes["privacy_proxy"])
	}

	status := findFinding(t, got, "whois.status")
	codes, _ := status.Attributes["codes"].([]string)
	if len(codes) != 2 {
		t.Errorf("status.codes = %v, want 2 entries", codes)
	}

	expiry := findFinding(t, got, "whois.expiry")
	if expiry.Attributes["present"] != true {
		t.Fatalf("expiry.present = %v, want true", expiry.Attributes["present"])
	}
	if expiry.Attributes["date"] != "2027-01-15T00:00:00Z" {
		t.Errorf("expiry.date = %v", expiry.Attributes["date"])
	}
}

// TestRun_RijksoverheidFixture pins the scanner spec scenario
// "Organisation is its own registrar (live fixture)": the live RDAP
// capture for rijksoverheid.nl (2026-09-19) — registrant redacted,
// registrar "Rijksoverheid", no reseller at any nesting level, and
// only registration / last-changed events (no expiration), so
// whois.expiry records absent.
func TestRun_RijksoverheidFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/rdap-rijksoverheid.nl-20260919.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got := runFixture(t, body)

	registrar := findFinding(t, got, "whois.registrar")
	if registrar.Attributes["name"] != "Rijksoverheid" {
		t.Errorf("registrar.name = %v", registrar.Attributes["name"])
	}

	identity := findFinding(t, got, "whois.registrant_identity")
	if identity.Attributes["name"] != "REDACTED FOR PRIVACY" {
		t.Errorf("identity.name = %v", identity.Attributes["name"])
	}
	if identity.Attributes["privacy_proxy"] != true {
		t.Errorf("identity.privacy_proxy = %v, want true", identity.Attributes["privacy_proxy"])
	}
	// RFC 9537 `redacted` array is recorded as evidence only — it does
	// not decide anything by itself (design.md).
	if identity.Attributes["rfc9537_redacted"] != true {
		t.Errorf("identity.rfc9537_redacted = %v, want true", identity.Attributes["rfc9537_redacted"])
	}

	reseller := findFinding(t, got, "whois.reseller")
	if reseller.Attributes["present"] != false {
		t.Errorf("reseller.present = %v, want false (no reseller at any nesting level)", reseller.Attributes["present"])
	}

	expiry := findFinding(t, got, "whois.expiry")
	if expiry.Attributes["present"] != false || expiry.Attributes["date"] != "absent" {
		t.Errorf("expiry = %+v, want present=false date=absent (registry publishes no expiration event)", expiry.Attributes)
	}

	status := findFinding(t, got, "whois.status")
	codes, _ := status.Attributes["codes"].([]string)
	if len(codes) != 1 || codes[0] != "active" {
		t.Errorf("status.codes = %v, want [active]", codes)
	}
}

func TestRun_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, err := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(got) != 1 || got[0].ProbeID != "whois.unavailable" {
		t.Errorf("expected unavailable; got %+v", got)
	}
}

func TestRun_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	p := &Probe{BaseURL: srv.URL + "/domain/"}
	got, _ := p.Run(context.Background(), models.Target{Domain: "example.nl"}, probe.Config{})
	if len(got) != 1 || got[0].ProbeID != "whois.unavailable" {
		t.Errorf("expected unavailable on bad JSON; got %+v", got)
	}
}

func TestRun_EmptyDomain(t *testing.T) {
	p := &Probe{}
	_, err := p.Run(context.Background(), models.Target{}, probe.Config{})
	if err == nil {
		t.Errorf("expected error on empty domain")
	}
}

// TestRun_NoEntities pins that a syntactically valid but data-sparse
// RDAP document (no entities, no status, no events) is not a failure:
// the probe still emits the four new Findings, each carrying its
// explicit absent/empty value, rather than falling back to
// whois.unavailable. Only the legacy whois.registrant / whois.registrar
// findings are absent, since there is nothing to report for them.
func TestRun_NoEntities(t *testing.T) {
	got := runFixture(t, []byte(`{"ldhName":"example.nl"}`))

	if hasFinding(got, "whois.unavailable") {
		t.Errorf("a data-sparse but valid RDAP doc must not be reported unavailable; got %+v", got)
	}
	if hasFinding(got, "whois.registrant") || hasFinding(got, "whois.registrar") {
		t.Errorf("no legacy registrant/registrar finding expected; got %+v", got)
	}

	identity := findFinding(t, got, "whois.registrant_identity")
	if identity.Attributes["name"] != "absent" || identity.Attributes["kind"] != "absent" {
		t.Errorf("identity = %+v, want name=absent kind=absent", identity.Attributes)
	}

	reseller := findFinding(t, got, "whois.reseller")
	if reseller.Attributes["present"] != false || reseller.Attributes["name"] != "absent" {
		t.Errorf("reseller = %+v, want present=false name=absent", reseller.Attributes)
	}

	status := findFinding(t, got, "whois.status")
	codes, _ := status.Attributes["codes"].([]string)
	if len(codes) != 0 {
		t.Errorf("status.codes = %v, want empty", codes)
	}

	expiry := findFinding(t, got, "whois.expiry")
	if expiry.Attributes["present"] != false || expiry.Attributes["date"] != "absent" {
		t.Errorf("expiry = %+v, want present=false date=absent", expiry.Attributes)
	}
}

func TestLookupRegistrantStatus(t *testing.T) {
	resellerFixture, err := os.ReadFile("testdata/rdap-reseller-nested.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	tests := []struct {
		name string
		body string
		want string
	}{
		{"present", string(resellerFixture), "present"}, // registrant fn "Gemeente Voorbeeld"
		{"no_fn_is_proxied", sampleRDAP, "proxied"},      // registrant entity present, no fn at all
		{"proxied", redactedRDAP, "proxied"},
		{"absent", `{"ldhName":"example.nl"}`, "absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			got, err := LookupRegistrantStatus(context.Background(), "example.nl", srv.URL+"/domain/", srv.Client(), "")
			if err != nil {
				t.Fatalf("lookup: %v", err)
			}
			if got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLookupRegistrantStatus_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := LookupRegistrantStatus(context.Background(), "example.zz", srv.URL+"/domain/", srv.Client(), "")
	if err == nil {
		t.Fatal("expected error on 404")
	}
}
