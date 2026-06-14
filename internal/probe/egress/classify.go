package egress

import (
	"net/url"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Confidence labels how strongly classify believes its category
// match. The threshold for emitting an evidence-bearing Finding (vs
// dropping to egress.unknown) is medium and above.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
	ConfidenceNone   Confidence = "none"
)

// Classification is the verdict for one Candidate.
type Classification struct {
	Category   string // object_storage, smtp, oidc, database, log_shipper, webhook, unknown
	Provider   string // aws, gcs, azure, minio, generic, ""
	Region     string // best-effort, e.g. "eu-west-1"
	Host       string // resolved host portion (without scheme)
	Port       string // best-effort, "" when none
	Confidence Confidence
	Rule       string // identifier of the rule that matched (e.g. "aws_s3_region_host")
	Dimension  models.DimensionHint
}

// Classify maps a single (key, value) pair (typically env-var or
// config-key) into one of the category bins. The function is total —
// values that cannot be classified come back with Category=="unknown"
// and ConfidenceNone, never as an error.
func Classify(key, value string) Classification {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Classification{Category: "unknown", Confidence: ConfidenceNone, Rule: "empty_value"}
	}
	for _, r := range rules {
		if r.match(key, trimmed) {
			return r.classify(trimmed)
		}
	}
	return Classification{Category: "unknown", Confidence: ConfidenceNone, Rule: "no_match", Host: extractHost(trimmed)}
}

// classifierRule captures a single rule in the table. The match
// closure decides whether the rule fires; classify returns the
// Classification when it does.
type classifierRule struct {
	id       string
	match    func(key, value string) bool
	classify func(value string) Classification
}

// rules is the active rule table populated by Configure on every load
// of the vendor list. Order matters: the first match wins.
var rules []classifierRule

// buildRules assembles the rule table from the active Vendors. Called
// from Configure (vendors.go) so a CLI flag override rebuilds the
// table without process restart.
func buildRules() []classifierRule {
	return []classifierRule{
		awsS3RegionRule(),
		gcsRule(),
		azureBlobRule(),
		minioGenericS3Rule(),
		oidcIssuerRule(),
		databaseRule(),
		smtpRule(),
		logShipperRule(),
		webhookRule(),
	}
}

// ----- AWS S3 -----

func awsS3RegionRule() classifierRule {
	return classifierRule{
		id: "aws_s3_region_host",
		match: func(_, v string) bool {
			h := extractHost(v)
			if strings.HasPrefix(strings.ToLower(v), "s3://") {
				return true
			}
			return activeAWSRE.MatchString(strings.ToLower(h))
		},
		classify: func(v string) Classification {
			h := extractHost(v)
			region := ""
			if m := activeAWSRE.FindStringSubmatch(strings.ToLower(h)); len(m) > 1 {
				region = m[1]
			}
			return Classification{
				Category:   "object_storage",
				Provider:   "aws",
				Region:     region,
				Host:       h,
				Confidence: ConfidenceHigh,
				Rule:       "aws_s3_region_host",
				Dimension:  models.DimensionJuridisch,
			}
		},
	}
}

// ----- GCS -----

func gcsRule() classifierRule {
	return classifierRule{
		id: "gcs_storage_host",
		match: func(_, v string) bool {
			return strings.Contains(strings.ToLower(extractHost(v)), activeVendors.ObjectStorage.GCSHostContains) ||
				strings.HasPrefix(strings.ToLower(v), "gs://")
		},
		classify: func(v string) Classification {
			return Classification{
				Category:   "object_storage",
				Provider:   "gcs",
				Host:       extractHost(v),
				Confidence: ConfidenceHigh,
				Rule:       "gcs_storage_host",
				Dimension:  models.DimensionJuridisch,
			}
		},
	}
}

// ----- Azure Blob -----

func azureBlobRule() classifierRule {
	return classifierRule{
		id: "azure_blob_host",
		match: func(_, v string) bool {
			return strings.Contains(strings.ToLower(extractHost(v)), activeVendors.ObjectStorage.AzureHostContains)
		},
		classify: func(v string) Classification {
			return Classification{
				Category:   "object_storage",
				Provider:   "azure",
				Host:       extractHost(v),
				Confidence: ConfidenceHigh,
				Rule:       "azure_blob_host",
				Dimension:  models.DimensionJuridisch,
			}
		},
	}
}

// ----- MinIO / generic S3-style -----

func minioGenericS3Rule() classifierRule {
	return classifierRule{
		id: "s3_endpoint_keyname",
		match: func(k, v string) bool {
			lk := strings.ToLower(k)
			return (strings.HasPrefix(lk, "s3_") || strings.Contains(lk, "_s3_") || lk == "s3_endpoint") &&
				extractHost(v) != ""
		},
		classify: func(v string) Classification {
			return Classification{
				Category:   "object_storage",
				Provider:   "generic",
				Host:       extractHost(v),
				Confidence: ConfidenceMedium,
				Rule:       "s3_endpoint_keyname",
				Dimension:  models.DimensionJuridisch,
			}
		},
	}
}

// ----- OIDC -----

func oidcIssuerRule() classifierRule {
	return classifierRule{
		id: "oidc_issuer",
		match: func(k, v string) bool {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "oidc") || strings.Contains(lk, "issuer") {
				return urlSchemeRE.MatchString(v)
			}
			return strings.Contains(strings.ToLower(v), "/.well-known/openid-configuration")
		},
		classify: func(v string) Classification {
			return Classification{
				Category:   "oidc",
				Host:       extractHost(v),
				Confidence: ConfidenceHigh,
				Rule:       "oidc_issuer",
				Dimension:  models.DimensionDataAI,
			}
		},
	}
}

// ----- database -----

var dbSchemes = []string{"postgres", "postgresql", "mysql", "mariadb", "mongodb", "redis", "rediss", "mssql", "sqlserver", "clickhouse"}

func databaseRule() classifierRule {
	return classifierRule{
		id: "database_url_scheme",
		match: func(_, v string) bool {
			lower := strings.ToLower(v)
			for _, s := range dbSchemes {
				if strings.HasPrefix(lower, s+"://") {
					return true
				}
			}
			return false
		},
		classify: func(v string) Classification {
			engine := strings.SplitN(strings.ToLower(v), "://", 2)[0]
			port := portFromURL(v)
			return Classification{
				Category:   "database",
				Provider:   engine,
				Host:       extractHost(v),
				Port:       port,
				Confidence: ConfidenceHigh,
				Rule:       "database_url_scheme",
				Dimension:  models.DimensionJuridisch,
			}
		},
	}
}

// ----- SMTP -----

func smtpRule() classifierRule {
	return classifierRule{
		id: "smtp_keyname_or_scheme",
		match: func(k, v string) bool {
			lk := strings.ToLower(k)
			if strings.HasPrefix(strings.ToLower(v), "smtp://") {
				return true
			}
			return (strings.Contains(lk, "smtp") || lk == "mail_host" || lk == "mailer_host") &&
				extractHost(v) != ""
		},
		classify: func(v string) Classification {
			return Classification{
				Category:   "smtp",
				Host:       extractHost(v),
				Port:       portFromURL(v),
				Confidence: ConfidenceHigh,
				Rule:       "smtp_keyname_or_scheme",
				Dimension:  models.DimensionDataAI,
			}
		},
	}
}

// ----- log shipper -----

func logShipperRule() classifierRule {
	return classifierRule{
		id: "log_shipper",
		match: func(k, v string) bool {
			h := strings.ToLower(extractHost(v))
			for _, e := range activeVendors.LogShippers {
				if strings.Contains(h, e.HostContains) {
					return true
				}
			}
			return activeLogShipperKeyRE != nil && activeLogShipperKeyRE.MatchString(k)
		},
		classify: func(v string) Classification {
			h := strings.ToLower(extractHost(v))
			rule := "log_shipper"
			for _, e := range activeVendors.LogShippers {
				if strings.Contains(h, e.HostContains) {
					rule = e.RuleID
					break
				}
			}
			return Classification{
				Category:   "log_shipper",
				Host:       extractHost(v),
				Confidence: ConfidenceMedium,
				Rule:       rule,
				Dimension:  models.DimensionOperationeel,
			}
		},
	}
}

// ----- webhook -----

func webhookRule() classifierRule {
	return classifierRule{
		id: "webhook",
		match: func(k, v string) bool {
			h := strings.ToLower(extractHost(v))
			for _, e := range activeVendors.Webhooks {
				if strings.Contains(h, e.HostContains) {
					return true
				}
			}
			return activeWebhookKeyRE != nil && activeWebhookKeyRE.MatchString(k) && extractHost(v) != ""
		},
		classify: func(v string) Classification {
			h := strings.ToLower(extractHost(v))
			rule := "webhook"
			for _, e := range activeVendors.Webhooks {
				if strings.Contains(h, e.HostContains) {
					rule = e.RuleID
					break
				}
			}
			return Classification{
				Category:   "webhook",
				Host:       extractHost(v),
				Confidence: ConfidenceMedium,
				Rule:       rule,
				Dimension:  models.DimensionTechnologie,
			}
		},
	}
}

// ----- helpers -----

// extractHost returns the host portion of a URL-or-host string, or
// the empty string if no host is recognisable. It tolerates inputs
// like "host:port", "scheme://host:port/path", and bare hostnames.
func extractHost(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if urlSchemeRE.MatchString(v) {
		u, err := url.Parse(v)
		if err == nil && u.Host != "" {
			h := u.Hostname()
			if h != "" {
				return h
			}
		}
	}
	// Bare host or host:port.
	host := v
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.Index(host, "?"); i >= 0 {
		host = host[:i]
	}
	if strings.Contains(host, ":") {
		parts := strings.SplitN(host, ":", 2)
		host = parts[0]
	}
	return host
}

func portFromURL(v string) string {
	if !urlSchemeRE.MatchString(v) {
		return ""
	}
	u, err := url.Parse(v)
	if err != nil || u == nil {
		return ""
	}
	return u.Port()
}
