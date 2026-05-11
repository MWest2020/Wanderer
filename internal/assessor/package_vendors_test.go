package assessor

import (
	"testing"
)

func TestPackageVendorEntries_NotEmpty(t *testing.T) {
	entries := PackageVendorEntries()
	if len(entries) == 0 {
		t.Fatal("PackageVendorEntries returned zero entries")
	}
	for i, e := range entries {
		if e.Jurisdiction == "" {
			t.Errorf("entry %d: empty jurisdiction", i)
		}
		switch e.Jurisdiction {
		case "eu", "us", "other":
		default:
			t.Errorf("entry %d: unexpected jurisdiction %q (must be eu/us/other)", i, e.Jurisdiction)
		}
		if e.RpmVendor == "" && e.DpkgDomain == "" {
			t.Errorf("entry %d: neither rpm_vendor nor dpkg_domain set", i)
		}
		if e.ParentOrg == "" {
			t.Errorf("entry %d: empty parent_org — verdict text needs it", i)
		}
	}
}

func TestClassifyPackageVendor_RpmHits(t *testing.T) {
	cases := []struct {
		name       string
		rpmVendor  string
		wantOK     bool
		wantJurisd string
	}{
		{"Fedora Project → US (Red Hat)", "Fedora Project", true, "us"},
		{"case insensitive", "FEDORA project", true, "us"},
		{"Red Hat, Inc. → US", "Red Hat, Inc.", true, "us"},
		{"openSUSE → EU", "openSUSE", true, "eu"},
		{"Debian → other", "Debian", true, "other"},
		{"Microsoft → US", "Microsoft Corporation", true, "us"},
		{"unknown vendor", "Acme Cloud Ltd", false, ""},
		{"empty input", "", false, ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ClassifyPackageVendor(tc.rpmVendor, "")
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (jurisdiction was %q)", ok, tc.wantOK, got.Jurisdiction)
			}
			if tc.wantOK && got.Jurisdiction != tc.wantJurisd {
				t.Errorf("jurisdiction = %q, want %q", got.Jurisdiction, tc.wantJurisd)
			}
		})
	}
}

func TestClassifyPackageVendor_DpkgMaintainer(t *testing.T) {
	cases := []struct {
		name       string
		maintainer string
		wantOK     bool
		wantJurisd string
	}{
		{"debian.org → other", "Foo <bash@packages.debian.org>", true, "other"},
		{"ubuntu.com → US (Canonical UK = non-EEA)", "Ubuntu Devs <x@lists.ubuntu.com>", true, "us"},
		{"suse.de → EU", "SUSE PostgreSQL Team <pg@suse.de>", true, "eu"},
		{"redhat.com → US", "RH Maintainers <rh@redhat.com>", true, "us"},
		{"unknown domain", "John <j@nowhere.example>", false, ""},
		{"bareword maintainer", "JustAName", false, ""},
		{"empty input", "", false, ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ClassifyPackageVendor("", tc.maintainer)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (jurisdiction was %q)", ok, tc.wantOK, got.Jurisdiction)
			}
			if tc.wantOK && got.Jurisdiction != tc.wantJurisd {
				t.Errorf("jurisdiction = %q, want %q", got.Jurisdiction, tc.wantJurisd)
			}
		})
	}
}

func TestClassifyPackageVendor_RpmTakesPrecedence(t *testing.T) {
	// When both channels are provided, the RPM channel matches
	// first because vendor lookups are cheap and unambiguous.
	got, ok := ClassifyPackageVendor("Fedora Project", "Foo <x@suse.de>")
	if !ok {
		t.Fatal("expected a classification")
	}
	if got.Jurisdiction != "us" {
		t.Errorf("jurisdiction = %q, want us (RPM vendor takes precedence)", got.Jurisdiction)
	}
}
