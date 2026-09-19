package wand

import "testing"

func TestRegistryRedactionEntries_NotEmpty(t *testing.T) {
	entries := RegistryRedactionEntries()
	if len(entries) == 0 {
		t.Fatal("RegistryRedactionEntries returned zero entries")
	}
	for i, e := range entries {
		if e.TLD == "" {
			t.Errorf("entry %d: empty tld", i)
		}
		if e.Registry == "" {
			t.Errorf("entry %d: empty registry — verdict text needs it", i)
		}
	}
}

func TestRegistryRedactionFor(t *testing.T) {
	cases := []struct {
		name string
		tld  string
		want string
		ok   bool
	}{
		{"nl seeded", "nl", "SIDN", true},
		{"case insensitive", "NL", "SIDN", true},
		{"leading dot tolerated", ".nl", "SIDN", true},
		{"unlisted TLD", "com", "", false},
		{"empty input", "", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RegistryRedactionFor(tc.tld)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (got %q)", ok, tc.ok, got)
			}
			if tc.ok && got != tc.want {
				t.Errorf("registry = %q, want %q", got, tc.want)
			}
		})
	}
}
