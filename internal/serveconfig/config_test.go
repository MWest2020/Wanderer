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
