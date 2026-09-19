package wand

import "testing"

func TestPrivacyProxyEntries_NotEmpty(t *testing.T) {
	entries := PrivacyProxyEntries()
	if len(entries) == 0 {
		t.Fatal("PrivacyProxyEntries returned zero entries")
	}
	for i, e := range entries {
		if e.Name == "" {
			t.Errorf("entry %d: empty name", i)
		}
	}
}

func TestMatchPrivacyProxy(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"exact seeded name", "Domains By Proxy", "Domains By Proxy", true},
		{"substring within a longer string", "Domains By Proxy, LLC", "Domains By Proxy", true},
		{"case insensitive", "DOMAINS BY PROXY, LLC", "Domains By Proxy", true},
		{"unrelated name", "Gemeente Voorbeeld B.V.", "", false},
		{"generic redaction placeholder is not a commercial proxy", "REDACTED FOR PRIVACY", "", false},
		{"empty input", "", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MatchPrivacyProxy(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (matched %q)", ok, tc.ok, got)
			}
			if tc.ok && got != tc.want {
				t.Errorf("matched = %q, want %q", got, tc.want)
			}
		})
	}
}
