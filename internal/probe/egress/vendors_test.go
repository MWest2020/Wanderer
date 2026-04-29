package egress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadVendors_Embedded(t *testing.T) {
	v, err := LoadVendors("")
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	if v.ObjectStorage.AWSRegionalRegex == "" {
		t.Fatal("aws regex empty")
	}
	if len(v.LogShippers) == 0 {
		t.Fatal("log_shippers empty")
	}
	// Embedded default keeps the historical datadog entry.
	found := false
	for _, e := range v.LogShippers {
		if e.HostContains == "datadoghq.com" && e.RuleID == "datadog" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("datadoghq.com entry missing from embedded defaults")
	}
}

func TestLoadVendors_OverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vendors.yaml")
	body := `log_shippers:
  - host_contains: vendor.example.nl
    rule_id: example_logger
log_shipper_key_regex: "(?i)example_logger"
webhooks:
  - host_contains: hooks.example.nl
    rule_id: example_webhook
webhook_key_regex: "(?i)example_webhook"
object_storage:
  aws_regional_regex: '^s3[.\-]([a-z]{2}-[a-z]+-\d)\.amazonaws\.com$'
  gcs_host_contains: storage.googleapis.com
  azure_host_contains: blob.core.windows.net
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := LoadVendors(path)
	if err != nil {
		t.Fatalf("load override: %v", err)
	}
	if got := v.LogShippers[0].HostContains; got != "vendor.example.nl" {
		t.Errorf("LogShippers[0] = %s", got)
	}

	// Apply the override and check the classifier picks it up.
	defer restoreEmbeddedVendors(t)
	Configure(v)
	c := Classify("LOG_TARGET", "https://vendor.example.nl/ingest")
	if c.Category != "log_shipper" || c.Rule != "example_logger" {
		t.Errorf("override classification = %+v", c)
	}
}

func TestLoadVendors_MalformedYAMLFatal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("log_shippers: [\n  this is not yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadVendors(path)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should name offending file, got %v", err)
	}
}

func TestLoadVendors_ValidationCatchesMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "incomplete.yaml")
	body := `log_shippers:
  - host_contains: ""
    rule_id: x
object_storage:
  aws_regional_regex: '.*'
  gcs_host_contains: gcs
  azure_host_contains: azure
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadVendors(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "log_shippers[0]") {
		t.Errorf("expected log_shippers[0] error, got %v", err)
	}
}

func TestLoadVendors_EnvFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.yaml")
	body := `webhooks:
  - host_contains: hooks.env.example
    rule_id: env_hook
object_storage:
  aws_regional_regex: '.*'
  gcs_host_contains: gcs
  azure_host_contains: azure
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WANDERER_VENDORS", path)
	v, err := LoadVendors("")
	if err != nil {
		t.Fatalf("env fallback: %v", err)
	}
	if v.Webhooks[0].HostContains != "hooks.env.example" {
		t.Errorf("env-loaded webhook = %+v", v.Webhooks[0])
	}
}

// restoreEmbeddedVendors resets the active classifier table back to
// the embedded default so a test that called Configure() does not
// leak state into other tests in this package.
func restoreEmbeddedVendors(t *testing.T) {
	t.Helper()
	v, err := LoadVendors("")
	if err != nil {
		t.Fatalf("restore embedded: %v", err)
	}
	Configure(v)
}
