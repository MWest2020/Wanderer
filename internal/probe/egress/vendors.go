package egress

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"regexp"

	"go.yaml.in/yaml/v2"
)

//go:embed vendors.yaml
var defaultVendorsYAML []byte

// Vendors is the externalised classifier table. The embedded default
// (vendors.yaml) ships with the binary; an operator may replace it
// wholesale via `--vendors <path>` or `WANDERER_VENDORS=<path>` so
// new log shippers, webhook endpoints, or object-storage hosts can
// land without a rebuild.
type Vendors struct {
	LogShippers     []VendorEntry        `yaml:"log_shippers"`
	LogShipperKeyRE string               `yaml:"log_shipper_key_regex"`
	Webhooks        []VendorEntry        `yaml:"webhooks"`
	WebhookKeyRE    string               `yaml:"webhook_key_regex"`
	ObjectStorage   ObjectStorageVendors `yaml:"object_storage"`
}

// VendorEntry is one host-based classifier rule. Both fields are
// required: HostContains is the substring matched against the
// destination host; RuleID becomes Classification.Rule (and the
// Finding's `classifier_rule` attribute) so an auditor can trace
// which line of the YAML produced the verdict.
type VendorEntry struct {
	HostContains string `yaml:"host_contains"`
	RuleID       string `yaml:"rule_id"`
}

// ObjectStorageVendors captures the cloud-storage substrings the
// classifier consults. Each provider has a fixed RuleID
// (`aws_s3_region_host`, `gcs_storage_host`, `azure_blob_host`)
// so existing assessor cross-references stay stable when the
// underlying YAML changes.
type ObjectStorageVendors struct {
	AWSRegionalRegex  string `yaml:"aws_regional_regex"`
	GCSHostContains   string `yaml:"gcs_host_contains"`
	AzureHostContains string `yaml:"azure_host_contains"`
}

// LoadVendors loads the vendor table. Precedence: `path` argument,
// then `WANDERER_VENDORS` env var, then the embedded default. A
// non-empty override that fails to read or parse is fatal — the
// caller (CLI) should surface the error and exit non-zero so an
// operator does not silently fall back to the embedded list when
// they meant to ship a custom one.
func LoadVendors(path string) (*Vendors, error) {
	src, name := defaultVendorsYAML, "<embedded>"
	if path == "" {
		path = os.Getenv("WANDERER_VENDORS")
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("vendors: read %s: %w", path, err)
		}
		src, name = b, path
	}
	return parseVendors(src, name)
}

func parseVendors(src []byte, name string) (*Vendors, error) {
	v := &Vendors{}
	if err := yaml.UnmarshalStrict(src, v); err != nil {
		return nil, fmt.Errorf("vendors: parse %s: %w", name, err)
	}
	if err := v.validate(); err != nil {
		return nil, fmt.Errorf("vendors: %s: %w", name, err)
	}
	return v, nil
}

func (v *Vendors) validate() error {
	if v.ObjectStorage.AWSRegionalRegex == "" {
		return errors.New("object_storage.aws_regional_regex: required")
	}
	if _, err := regexp.Compile(v.ObjectStorage.AWSRegionalRegex); err != nil {
		return fmt.Errorf("object_storage.aws_regional_regex: %w", err)
	}
	if v.ObjectStorage.GCSHostContains == "" {
		return errors.New("object_storage.gcs_host_contains: required")
	}
	if v.ObjectStorage.AzureHostContains == "" {
		return errors.New("object_storage.azure_host_contains: required")
	}
	if v.LogShipperKeyRE != "" {
		if _, err := regexp.Compile(v.LogShipperKeyRE); err != nil {
			return fmt.Errorf("log_shipper_key_regex: %w", err)
		}
	}
	if v.WebhookKeyRE != "" {
		if _, err := regexp.Compile(v.WebhookKeyRE); err != nil {
			return fmt.Errorf("webhook_key_regex: %w", err)
		}
	}
	for i, e := range v.LogShippers {
		if e.HostContains == "" || e.RuleID == "" {
			return fmt.Errorf("log_shippers[%d]: host_contains and rule_id required", i)
		}
	}
	for i, e := range v.Webhooks {
		if e.HostContains == "" || e.RuleID == "" {
			return fmt.Errorf("webhooks[%d]: host_contains and rule_id required", i)
		}
	}
	return nil
}

// Configure swaps the active classifier table. Callers (the agent
// CLI) should call this once at startup after flag parsing. The
// classifier is not safe to reconfigure while probes are running.
func Configure(v *Vendors) {
	if v == nil {
		return
	}
	activeVendors = v
	activeAWSRE = regexp.MustCompile(v.ObjectStorage.AWSRegionalRegex)
	if v.LogShipperKeyRE != "" {
		activeLogShipperKeyRE = regexp.MustCompile(v.LogShipperKeyRE)
	} else {
		activeLogShipperKeyRE = nil
	}
	if v.WebhookKeyRE != "" {
		activeWebhookKeyRE = regexp.MustCompile(v.WebhookKeyRE)
	} else {
		activeWebhookKeyRE = nil
	}
	rules = buildRules()
}

var (
	activeVendors         *Vendors
	activeAWSRE           *regexp.Regexp
	activeLogShipperKeyRE *regexp.Regexp
	activeWebhookKeyRE    *regexp.Regexp
)

func init() {
	v, err := LoadVendors("")
	if err != nil {
		panic(fmt.Sprintf("egress: load embedded vendors: %v", err))
	}
	Configure(v)
}
