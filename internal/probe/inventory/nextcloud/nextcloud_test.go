package nextcloud_test

import (
	"context"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/probe/inventory/nextcloud"
)

// stubResolver returns deterministic ASN data per host so tests
// can pin the geoip annotation pipeline without a real GeoLite2
// DB.
type stubResolver struct {
	byHost map[string]struct {
		asn     uint
		org     string
		country string
	}
}

func (r stubResolver) Resolve(host string) (uint, string, string, bool) {
	d, ok := r.byHost[host]
	return d.asn, d.org, d.country, ok
}

func TestParse_AppList(t *testing.T) {
	got, err := nextcloud.Parse(`{"enabled":{"calendar":"4.5.0"},"disabled":{"talk":"19.0.0"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	for _, f := range got {
		if f.ProbeID != "inventory.nextcloud.app" {
			t.Errorf("ProbeID = %s", f.ProbeID)
		}
	}
}

func TestParseStatus_Happy(t *testing.T) {
	raw := `{"versionstring":"28.0.5","version":"28.0.5.1","edition":"","productname":"Nextcloud"}`
	got, err := nextcloud.ParseStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProbeID != "inventory.nextcloud.version" {
		t.Errorf("ProbeID = %s", got.ProbeID)
	}
	if got.Subject != "28.0.5" {
		t.Errorf("Subject = %s, want 28.0.5", got.Subject)
	}
	if got.Attributes["major"].(int) != 28 {
		t.Errorf("major = %v, want 28", got.Attributes["major"])
	}
	if got.Attributes["supported"].(bool) != true {
		t.Errorf("supported = false, want true (28 is in the contract)")
	}
}

func TestParseStatus_UnsupportedMajor(t *testing.T) {
	raw := `{"versionstring":"24.0.1","version":"24.0.1.0","productname":"Nextcloud"}`
	got, err := nextcloud.ParseStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Attributes["supported"].(bool) != false {
		t.Errorf("supported = true, want false (24 is below the contract)")
	}
}

func TestParseStatus_EmptyVersionString(t *testing.T) {
	_, err := nextcloud.ParseStatus(`{"versionstring":""}`)
	if err == nil {
		t.Error("expected error on empty versionstring")
	}
}

func TestParseSystemConfig_TrustedDomains(t *testing.T) {
	raw := `{"system":{"trusted_domains":["cloud.example.nl","cloud.example.com"]}}`
	trusted, _ := nextcloud.ParseSystemConfig(raw)
	if len(trusted) != 2 {
		t.Fatalf("trusted = %d, want 2", len(trusted))
	}
	for _, f := range trusted {
		if f.ProbeID != "inventory.nextcloud.trusted_domain" {
			t.Errorf("ProbeID = %s", f.ProbeID)
		}
	}
}

func TestParseSystemConfig_ObjectstoreWithEndpoint(t *testing.T) {
	raw := `{"system":{"objectstore":{"class":"OC\\Files\\ObjectStore\\S3","arguments":{"bucket":"nextcloud-data","region":"us-east-1","hostname":"s3.amazonaws.com"}}}}`
	_, stores := nextcloud.ParseSystemConfig(raw)
	if len(stores) != 1 {
		t.Fatalf("stores = %d, want 1", len(stores))
	}
	s := stores[0]
	if s.ProbeID != "inventory.nextcloud.objectstore" {
		t.Errorf("ProbeID = %s", s.ProbeID)
	}
	if s.Attributes["bucket"] != "nextcloud-data" {
		t.Errorf("bucket = %v", s.Attributes["bucket"])
	}
	if s.Attributes["endpoint_host"] != "s3.amazonaws.com" {
		t.Errorf("endpoint_host = %v", s.Attributes["endpoint_host"])
	}
}

func TestParseSystemConfig_ObjectstoreEndpointURL(t *testing.T) {
	// Some installs put the full URL in `endpoint` rather than
	// splitting into hostname.
	raw := `{"system":{"objectstore":{"class":"OC\\Files\\ObjectStore\\S3","arguments":{"bucket":"data","endpoint":"https://s3.eu-central-1.amazonaws.com"}}}}`
	_, stores := nextcloud.ParseSystemConfig(raw)
	if len(stores) != 1 {
		t.Fatalf("stores = %d", len(stores))
	}
	if stores[0].Attributes["endpoint_host"] != "s3.eu-central-1.amazonaws.com" {
		t.Errorf("endpoint_host = %v", stores[0].Attributes["endpoint_host"])
	}
}

func TestParseOIDCProviders(t *testing.T) {
	raw := `[{"identifier":"keycloak","clientId":"nc","discoveryEndpoint":"https://login.example.nl/realms/main/.well-known/openid-configuration"}]`
	got, err := nextcloud.ParseOIDCProviders(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("providers = %d, want 1", len(got))
	}
	f := got[0]
	if f.Subject != "keycloak" {
		t.Errorf("Subject = %s", f.Subject)
	}
	if f.Attributes["issuer_host"] != "login.example.nl" {
		t.Errorf("issuer_host = %v", f.Attributes["issuer_host"])
	}
	if f.Attributes["issuer_url"] != "https://login.example.nl/realms/main" {
		t.Errorf("issuer_url = %v", f.Attributes["issuer_url"])
	}
}

func TestInspect_MergesAllFamilies(t *testing.T) {
	n := nextcloud.Nextcloud{
		QueryFunc: func(context.Context) (string, error) {
			return `{"enabled":{"calendar":"4.5.0"},"disabled":{}}`, nil
		},
		StatusFunc: func(context.Context) (string, error) {
			return `{"versionstring":"28.0.5","version":"28.0.5.1","productname":"Nextcloud"}`, nil
		},
		ConfigSystemFunc: func(context.Context) (string, error) {
			return `{"system":{"trusted_domains":["cloud.example.nl"],"objectstore":{"class":"S3","arguments":{"bucket":"data","hostname":"s3.amazonaws.com"}}}}`, nil
		},
		OIDCProviderFunc: func(context.Context) (string, error) {
			return `[{"identifier":"keycloak","clientId":"nc","discoveryEndpoint":"https://login.example.nl/.well-known/openid-configuration"}]`, nil
		},
		Resolver: stubResolver{byHost: map[string]struct {
			asn     uint
			org     string
			country string
		}{
			"s3.amazonaws.com": {16509, "Amazon.com, Inc.", "US"},
		}},
	}

	findings, err := n.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	seenFamilies := map[string]bool{}
	for _, f := range findings {
		seenFamilies[f.ProbeID] = true
	}
	for _, fam := range []string{
		"inventory.nextcloud.app",
		"inventory.nextcloud.version",
		"inventory.nextcloud.trusted_domain",
		"inventory.nextcloud.objectstore",
		"inventory.nextcloud.oidc_provider",
	} {
		if !seenFamilies[fam] {
			t.Errorf("missing family: %s", fam)
		}
	}

	// Check the objectstore Finding picked up the geoip annotation.
	for _, f := range findings {
		if f.ProbeID == "inventory.nextcloud.objectstore" {
			if f.Attributes["country"] != "US" {
				t.Errorf("objectstore country = %v, want US (resolver wired)", f.Attributes["country"])
			}
		}
	}
}

func TestInspect_OIDCUnavailable_WithAlternativeAppHint(t *testing.T) {
	n := nextcloud.Nextcloud{
		QueryFunc: func(context.Context) (string, error) {
			// social_login is installed but disabled — still
			// reportable as the alternative app.
			return `{"enabled":{"social_login":"5.0.0"},"disabled":{}}`, nil
		},
		StatusFunc: func(context.Context) (string, error) {
			return `{"versionstring":"28.0.5","version":"28.0.5.1"}`, nil
		},
		ConfigSystemFunc: func(context.Context) (string, error) {
			return `{"system":{"trusted_domains":[]}}`, nil
		},
		OIDCProviderFunc: func(context.Context) (string, error) {
			return "", errOIDCMissing
		},
	}

	findings, err := n.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var sawUnavailable bool
	for _, f := range findings {
		if f.ProbeID == "inventory.nextcloud.oidc.unavailable" {
			sawUnavailable = true
			if f.Attributes["alternative_app"] != "social_login" {
				t.Errorf("alternative_app = %v, want social_login", f.Attributes["alternative_app"])
			}
			reason, _ := f.Attributes["reason"].(string)
			if !strings.Contains(reason, "missing") {
				t.Errorf("reason = %q, want substring 'missing'", reason)
			}
		}
	}
	if !sawUnavailable {
		t.Error("expected inventory.nextcloud.oidc.unavailable finding when user_oidc is absent")
	}
}

// errOIDCMissing simulates the shell-out failure the user_oidc
// app's absence causes.
type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errOIDCMissing = sentinelErr("occ command missing: user_oidc:provider")
