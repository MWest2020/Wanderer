package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func pkgWithVendor(id, name, vendor string) models.Finding {
	return models.Finding{
		ID:       id,
		ProbeID:  "inventory.packages.rpm",
		Subject:  name,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"vendor": vendor,
		},
	}
}

func pkgWithMaintainer(id, name, maintainer string) models.Finding {
	return models.Finding{
		ID:       id,
		ProbeID:  "inventory.packages.dpkg",
		Subject:  name,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"maintainer": maintainer,
		},
	}
}

func TestEUPackageOrigin_AllFedoraIsAfhankelijk(t *testing.T) {
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		pkgWithVendor("p1", "bash", "Fedora Project"),
		pkgWithVendor("p2", "vim", "Fedora Project"),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("fedora host: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "Red Hat") {
		t.Errorf("verdict = %q must name the parent_org (Red Hat)", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "inspected 2 packages") {
		t.Errorf("verdict = %q must include inspected count", got.Verdict)
	}
	if len(got.Evidence) == 0 {
		t.Error("US-hit branch must cite Finding IDs")
	}
}

func TestEUPackageOrigin_AllOpenSUSEIsSoeverein(t *testing.T) {
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		pkgWithVendor("p1", "bash", "openSUSE"),
		pkgWithVendor("p2", "vim", "openSUSE"),
		pkgWithVendor("p3", "curl", "openSUSE"),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("openSUSE host: score = %s, want soeverein", got.Score)
	}
	if !strings.Contains(got.Verdict, "EU-tied") {
		t.Errorf("verdict = %q must call out EU-tied attribution", got.Verdict)
	}
	if len(got.Evidence) == 0 {
		t.Error("soeverein branch must cite negative evidence")
	}
}

func TestEUPackageOrigin_MixedScoresVoldoende(t *testing.T) {
	// Inspected packages with unclassified vendors only.
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		pkgWithVendor("p1", "local-build", "Acme Internal Builder"),
		pkgWithVendor("p2", "another", "Custom Org"),
	})
	if got.Score != models.ScoreVoldoende {
		t.Errorf("mixed/unclassified: score = %s, want voldoende", got.Score)
	}
	if !strings.Contains(got.Verdict, "0 classified") && !strings.Contains(got.Verdict, "jurisdiction not fully attributable") {
		t.Errorf("verdict = %q must explain why no soeverein call was made", got.Verdict)
	}
}

func TestEUPackageOrigin_OneUSHitDominates(t *testing.T) {
	// Even one US package among many EU ones flips the verdict.
	// That's intentional: the rule's job is to make any
	// US-vendored dependency visible.
	r := ruleByID(t, "wand.host.eu_package_origin")
	findings := []models.Finding{
		pkgWithVendor("p1", "bash", "openSUSE"),
		pkgWithVendor("p2", "vim", "openSUSE"),
		pkgWithVendor("p3", "azure-cli", "Microsoft Corporation"),
	}
	got := r.Match(findings)
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("mixed-with-US: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "Microsoft") {
		t.Errorf("verdict = %q must name Microsoft as parent", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "azure-cli") {
		t.Errorf("verdict = %q must list the offending package", got.Verdict)
	}
}

func TestEUPackageOrigin_DpkgMaintainer(t *testing.T) {
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		pkgWithMaintainer("p1", "bash", "Bash Maintainers <bash@packages.debian.org>"),
		pkgWithMaintainer("p2", "vim", "Vim Team <vim@tracker.debian.org>"),
	})
	// All packages classified as `other` (Debian governance) →
	// no US hits, no EU hits → voldoende.
	if got.Score != models.ScoreVoldoende {
		t.Errorf("all-debian: score = %s, want voldoende (community governance)", got.Score)
	}
}

func TestEUPackageOrigin_UbuntuIsUSBecauseCanonicalUK(t *testing.T) {
	// Canonical is UK, which is non-EEA — the rule's binary
	// jurisdiction call buckets that under US-tied. The verdict
	// names UK explicitly so an operator can see the call.
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		pkgWithMaintainer("p1", "systemd", "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>"),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("ubuntu: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "UK") {
		t.Errorf("verdict = %q must call out UK (non-EEA)", got.Verdict)
	}
}

func TestEUPackageOrigin_NoFindingsIsOnbekend(t *testing.T) {
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}

func TestEUPackageOrigin_PerimeterFindingsIgnored(t *testing.T) {
	r := ruleByID(t, "wand.host.eu_package_origin")
	got := r.Match([]models.Finding{
		{
			ID:       "t1",
			ProbeID:  "tls.issuer",
			Subject:  "example.nl",
			Severity: models.SeverityFinding,
			Attributes: map[string]any{
				"issuer_country": []string{"NL"},
			},
		},
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("perimeter only: score = %s, want onbekend (this rule reads inventory.packages.*)", got.Score)
	}
}
