package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWarnIfGeoIPMissing_DefaultEmits(t *testing.T) {
	t.Setenv("WANDERER_GEOIP_OPTIONAL", "")
	var buf bytes.Buffer
	warnIfGeoIPMissing(&buf, "", false)
	got := buf.String()
	if !strings.HasPrefix(got, "warning:") {
		t.Errorf("expected warning prefix, got %q", got)
	}
	if !strings.Contains(got, "--geoip") {
		t.Errorf("warning should reference --geoip flag, got %q", got)
	}
	if !strings.Contains(got, "docs/operator.md") {
		t.Errorf("warning should reference operator docs, got %q", got)
	}
}

func TestWarnIfGeoIPMissing_NoGeoIPSilences(t *testing.T) {
	t.Setenv("WANDERER_GEOIP_OPTIONAL", "")
	var buf bytes.Buffer
	warnIfGeoIPMissing(&buf, "", true)
	if buf.Len() != 0 {
		t.Errorf("--no-geoip should silence the warning, got %q", buf.String())
	}
}

func TestWarnIfGeoIPMissing_EnvOptionalSilences(t *testing.T) {
	t.Setenv("WANDERER_GEOIP_OPTIONAL", "1")
	var buf bytes.Buffer
	warnIfGeoIPMissing(&buf, "", false)
	if buf.Len() != 0 {
		t.Errorf("WANDERER_GEOIP_OPTIONAL=1 should silence the warning, got %q", buf.String())
	}
}

func TestWarnIfGeoIPMissing_ConfiguredPathSilences(t *testing.T) {
	t.Setenv("WANDERER_GEOIP_OPTIONAL", "")
	var buf bytes.Buffer
	warnIfGeoIPMissing(&buf, "/var/lib/wanderer/GeoLite2-ASN.mmdb", false)
	if buf.Len() != 0 {
		t.Errorf("a configured ASN path should silence the warning, got %q", buf.String())
	}
}
