package serveconfig_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/serveconfig"
)

func TestParse_EmptyYAMLValidates(t *testing.T) {
	c, err := serveconfig.Parse([]byte("{}"))
	if err != nil {
		t.Fatalf("empty YAML must parse: %v", err)
	}
	if c.Listen != "" || c.DB != "" || c.UI.Enabled {
		t.Errorf("empty YAML should yield zero Config, got %+v", c)
	}
}

func TestParse_FullExample(t *testing.T) {
	yamlBody := `
listen: ":9090"
db: /var/lib/wanderer/wanderer.db

geoip:
  asn: /tmp/asn.mmdb
  country: /tmp/country.mmdb
  optional: false

ui:
  enabled: true
  htpasswd: /etc/wanderer/htpasswd

schedules: /etc/wanderer/schedules.yaml

scan:
  per_probe_timeout: 45s
  budget: 3m
  user_agent: "Wanderer/1.0"
  allow_private_targets: false
`
	c, err := serveconfig.Parse([]byte(yamlBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.Listen != ":9090" {
		t.Errorf("Listen = %q", c.Listen)
	}
	if c.GeoIP.ASN != "/tmp/asn.mmdb" {
		t.Errorf("GeoIP.ASN = %q", c.GeoIP.ASN)
	}
	if !c.UI.Enabled {
		t.Errorf("UI.Enabled = false, want true")
	}
	if c.UI.Htpasswd != "/etc/wanderer/htpasswd" {
		t.Errorf("UI.Htpasswd = %q", c.UI.Htpasswd)
	}
	if c.Scan.PerProbeTimeout != 45*time.Second {
		t.Errorf("Scan.PerProbeTimeout = %s", c.Scan.PerProbeTimeout)
	}
	if c.Scan.Budget != 3*time.Minute {
		t.Errorf("Scan.Budget = %s", c.Scan.Budget)
	}
}

func TestParse_StrictRejectsUnknownField(t *testing.T) {
	// "htpasswrd" is the boring typo — strict mode must catch it.
	yamlBody := `
ui:
  enabled: true
  htpasswrd: /etc/wanderer/htpasswd
`
	_, err := serveconfig.Parse([]byte(yamlBody))
	if err == nil {
		t.Fatal("strict parse must reject unknown field, got nil error")
	}
	if !strings.Contains(err.Error(), "htpasswrd") {
		t.Errorf("error should name the bad field, got: %v", err)
	}
}

func TestParse_StrictRejectsTopLevelTypo(t *testing.T) {
	// Same protection at top level — a misnested or misspelled
	// top-level key fails fast.
	yamlBody := `
listenz: ":9090"
`
	_, err := serveconfig.Parse([]byte(yamlBody))
	if err == nil {
		t.Fatal("strict parse must reject unknown top-level field")
	}
}

func TestLoad_MissingFileIsHardError(t *testing.T) {
	_, err := serveconfig.Load("/nonexistent/path/to/serve.yaml")
	if err == nil {
		t.Fatal("Load on missing path must error")
	}
	if !strings.Contains(err.Error(), "/nonexistent") {
		t.Errorf("error should name the missing path, got: %v", err)
	}
}

func TestParse_OIDCBlock(t *testing.T) {
	yamlBody := `
ui:
  enabled: true
oidc:
  provider_url: https://cloud.example.nl
  client_id: wanderer
  client_secret_file: /etc/wanderer/oidc-secret
  redirect_url: https://wanderer.example.nl/ui/oauth/callback
  scopes: [openid, profile, email, groups]
  session_ttl: 8h
  revalidate_interval: 1m
`
	c, err := serveconfig.Parse([]byte(yamlBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !c.OIDC.Enabled() {
		t.Fatal("OIDC.Enabled() = false, want true")
	}
	if c.OIDC.ClientID != "wanderer" {
		t.Errorf("ClientID = %q", c.OIDC.ClientID)
	}
	if c.OIDC.SessionTTL != 8*time.Hour {
		t.Errorf("SessionTTL = %s", c.OIDC.SessionTTL)
	}
	if c.OIDC.RevalidateInterval != time.Minute {
		t.Errorf("RevalidateInterval = %s", c.OIDC.RevalidateInterval)
	}
	if len(c.OIDC.Scopes) != 4 {
		t.Errorf("Scopes = %v", c.OIDC.Scopes)
	}
}

func TestParse_PartialOIDCBlockIsRejected(t *testing.T) {
	// provider_url set but the rest missing — a config mistake we
	// catch at startup rather than at first login.
	yamlBody := `
oidc:
  provider_url: https://cloud.example.nl
`
	_, err := serveconfig.Parse([]byte(yamlBody))
	if err == nil {
		t.Fatal("partial oidc block must fail validation")
	}
	if !strings.Contains(err.Error(), "client_id") {
		t.Errorf("error should name the missing field, got: %v", err)
	}
}

func TestParse_EmptyOIDCBlockIsDisabled(t *testing.T) {
	c, err := serveconfig.Parse([]byte("{}"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.OIDC.Enabled() {
		t.Error("empty config should leave OIDC disabled")
	}
}

func TestParse_NextcloudBlock(t *testing.T) {
	yamlBody := `
nextcloud:
  enabled: true
  url: https://cloud.example.nl
  username: wanderer-bot
  app_password_file: /etc/wanderer/nc.token
  target_dir: Wanderer
`
	c, err := serveconfig.Parse([]byte(yamlBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !c.Nextcloud.Enabled {
		t.Fatal("Nextcloud.Enabled = false, want true")
	}
	if c.Nextcloud.Username != "wanderer-bot" || c.Nextcloud.TargetDir != "Wanderer" {
		t.Errorf("unexpected nextcloud block: %+v", c.Nextcloud)
	}
}

func TestParse_PartialNextcloudBlockIsRejected(t *testing.T) {
	// enabled: true but url missing — a config mistake caught at startup.
	yamlBody := `
nextcloud:
  enabled: true
  username: wanderer-bot
`
	_, err := serveconfig.Parse([]byte(yamlBody))
	if err == nil {
		t.Fatal("partial nextcloud block must fail validation")
	}
	if !strings.Contains(err.Error(), "nextcloud.url") {
		t.Errorf("error should name the missing field, got: %v", err)
	}
}
