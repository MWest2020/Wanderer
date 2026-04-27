package egress

import "testing"

func TestClassify_AWSS3(t *testing.T) {
	c := Classify("S3_ENDPOINT", "https://s3.eu-west-1.amazonaws.com")
	if c.Category != "object_storage" || c.Provider != "aws" {
		t.Errorf("got %+v", c)
	}
	if c.Region != "eu-west-1" {
		t.Errorf("region = %s", c.Region)
	}
	if c.Rule != "aws_s3_region_host" {
		t.Errorf("rule = %s", c.Rule)
	}
}

func TestClassify_GCS(t *testing.T) {
	c := Classify("BACKUP_TARGET", "gs://my-bucket/backups")
	if c.Category != "object_storage" || c.Provider != "gcs" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_AzureBlob(t *testing.T) {
	c := Classify("STORAGE_ACCOUNT", "https://acct.blob.core.windows.net/container")
	if c.Category != "object_storage" || c.Provider != "azure" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_DatabaseURL(t *testing.T) {
	c := Classify("DATABASE_URL", "postgres://app:secret@db.example.nl:5432/app")
	if c.Category != "database" || c.Provider != "postgres" {
		t.Errorf("got %+v", c)
	}
	if c.Host != "db.example.nl" {
		t.Errorf("host = %s", c.Host)
	}
	if c.Port != "5432" {
		t.Errorf("port = %s", c.Port)
	}
}

func TestClassify_SMTP(t *testing.T) {
	c := Classify("SMTP_HOST", "smtp.example.nl")
	if c.Category != "smtp" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_OIDC(t *testing.T) {
	c := Classify("OIDC_ISSUER", "https://login.example.nl/realms/wanderer")
	if c.Category != "oidc" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_LogShipper(t *testing.T) {
	c := Classify("LOG_TARGET", "https://api.datadoghq.com/v1/logs")
	if c.Category != "log_shipper" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_Webhook(t *testing.T) {
	c := Classify("ALERT_WEBHOOK_URL", "https://hooks.slack.com/services/T0/B0/abc")
	if c.Category != "webhook" {
		t.Errorf("got %+v", c)
	}
}

func TestClassify_Unknown(t *testing.T) {
	c := Classify("MISC", "https://exports.random-company.example/api")
	if c.Category != "unknown" {
		t.Errorf("got %+v", c)
	}
	if c.Rule != "no_match" {
		t.Errorf("rule = %s", c.Rule)
	}
	if c.Host != "exports.random-company.example" {
		t.Errorf("host = %s", c.Host)
	}
}

func TestClassify_EmptyValue(t *testing.T) {
	c := Classify("ANY", "")
	if c.Category != "unknown" || c.Rule != "empty_value" {
		t.Errorf("got %+v", c)
	}
}

func TestExtractHost(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.nl/path", "example.nl"},
		{"smtp://mail.example.nl:587", "mail.example.nl"},
		{"db.example.nl:5432", "db.example.nl"},
		{"db.example.nl", "db.example.nl"},
		{"", ""},
	}
	for _, c := range cases {
		if got := extractHost(c.in); got != c.want {
			t.Errorf("extractHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
