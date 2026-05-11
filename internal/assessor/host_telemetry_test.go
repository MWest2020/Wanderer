package assessor

import (
	"strings"
	"testing"
)

func TestHostTelemetryEntries_NotEmpty(t *testing.T) {
	entries := HostTelemetryEntries()
	if len(entries) == 0 {
		t.Fatal("HostTelemetryEntries returned zero entries; host_telemetry.yaml did not load")
	}
	for i, e := range entries {
		if e.Prefix == "" {
			t.Errorf("entry %d: empty prefix (every host_telemetry.yaml entry must carry a prefix)", i)
		}
		if e.Vendor == "" {
			t.Errorf("entry %d (prefix %q): empty vendor — verdict text needs a vendor of record", i, e.Prefix)
		}
	}
}

func TestHostTelemetryMatch_Cases(t *testing.T) {
	cases := []struct {
		name      string
		subject   string
		wantMatch bool
		wantNeed  string
	}{
		{name: "exact datadog package", subject: "datadog-agent", wantMatch: true, wantNeed: "Datadog"},
		{name: "case insensitive", subject: "Datadog-Agent", wantMatch: true, wantNeed: "Datadog"},
		{name: "newrelic prefix variant", subject: "newrelic-infra", wantMatch: true, wantNeed: "New Relic"},
		{name: "splunk forwarder", subject: "splunkforwarder", wantMatch: true, wantNeed: "Splunk"},
		{name: "open-source agent stays clean", subject: "collectd", wantMatch: false},
		{name: "unrelated package", subject: "nginx", wantMatch: false},
		{name: "empty subject", subject: "", wantMatch: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := HostTelemetryMatch(tc.subject)
			if ok != tc.wantMatch {
				t.Fatalf("HostTelemetryMatch(%q) ok = %v, want %v", tc.subject, ok, tc.wantMatch)
			}
			if tc.wantMatch && !strings.Contains(got.Vendor, tc.wantNeed) {
				t.Errorf("HostTelemetryMatch(%q) vendor = %q, want substring %q", tc.subject, got.Vendor, tc.wantNeed)
			}
		})
	}
}
