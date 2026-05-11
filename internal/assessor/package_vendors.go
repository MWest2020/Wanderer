package assessor

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed package_vendors.yaml
var packageVendorsYAML []byte

// PackageVendor is one classified entry from
// package_vendors.yaml.
type PackageVendor struct {
	RpmVendor    string `yaml:"rpm_vendor,omitempty"`
	DpkgDomain   string `yaml:"dpkg_domain,omitempty"`
	Jurisdiction string `yaml:"jurisdiction"`
	ParentOrg    string `yaml:"parent_org"`
}

type packageVendorsFile struct {
	Vendors []PackageVendor `yaml:"vendors"`
}

var (
	packageVendorsOnce sync.Once
	packageVendorsList []PackageVendor
	packageVendorsErr  error
)

func loadPackageVendors() ([]PackageVendor, error) {
	packageVendorsOnce.Do(func() {
		var f packageVendorsFile
		if err := yaml.UnmarshalStrict(packageVendorsYAML, &f); err != nil {
			packageVendorsErr = fmt.Errorf("assessor: parse package_vendors.yaml: %w", err)
			return
		}
		packageVendorsList = f.Vendors
	})
	return packageVendorsList, packageVendorsErr
}

// PackageVendorEntries returns the loaded list (for tests +
// rule-description text).
func PackageVendorEntries() []PackageVendor {
	list, _ := loadPackageVendors()
	return list
}

// ClassifyPackageVendor classifies a Finding's vendor /
// maintainer pair against the embedded list. At most one
// non-empty input should be provided; the function takes both
// for caller convenience so the rule can pass straight from
// the Attributes map without branching.
//
// Returns the matched entry + true when classified. When
// nothing matches, the second return value is false and the
// entry is empty — callers SHOULD treat that as "unknown
// jurisdiction" rather than a default "us" or "eu".
func ClassifyPackageVendor(rpmVendor, dpkgMaintainer string) (PackageVendor, bool) {
	list, err := loadPackageVendors()
	if err != nil {
		return PackageVendor{}, false
	}

	rpmLower := strings.ToLower(strings.TrimSpace(rpmVendor))
	domain := emailDomainFromMaintainer(dpkgMaintainer)

	for _, v := range list {
		if rpmLower != "" && v.RpmVendor != "" {
			if strings.Contains(rpmLower, strings.ToLower(v.RpmVendor)) {
				return v, true
			}
		}
		if domain != "" && v.DpkgDomain != "" {
			if strings.EqualFold(domain, v.DpkgDomain) {
				return v, true
			}
		}
	}
	return PackageVendor{}, false
}

// emailDomainFromMaintainer extracts the email domain from a
// dpkg Maintainer field. Maintainer values look like:
//
//	"Bash Maintainers <bash@packages.debian.org>"
//	"Some Person <x@y.example>"
//
// Returns "packages.debian.org" / "y.example" respectively, or
// "" when no email is parseable.
func emailDomainFromMaintainer(s string) string {
	if s == "" {
		return ""
	}
	lt := strings.LastIndex(s, "<")
	gt := strings.LastIndex(s, ">")
	if lt < 0 || gt < 0 || gt < lt {
		// Bare email like `team@host.example`?
		if at := strings.LastIndex(s, "@"); at >= 0 {
			return strings.TrimSpace(s[at+1:])
		}
		return ""
	}
	email := s[lt+1 : gt]
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return ""
	}
	return strings.TrimSpace(email[at+1:])
}
