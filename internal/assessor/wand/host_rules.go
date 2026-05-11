package wand

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// hostEUPackageOrigin scores the jurisdiction of every package
// vendor / maintainer observed on the host. Reads
// `inventory.packages.*` Findings and classifies each finding's
// `vendor` (rpm) or `maintainer` (dpkg) attribute against
// `internal/assessor/package_vendors.yaml`.
//
// Score buckets:
//
//   - afhankelijk — any single package classified to a US vendor.
//     The verdict names the package + parent_org.
//   - soeverein   — every classified package resolves to an EU
//     vendor (with negative-evidence sample).
//   - voldoende   — no US hits but not every package is
//     EU-classified (mixed / unclassified). The rule can't make
//     a confident sovereign call, but no red flag either.
//   - onbekend    — no inventory.packages.* findings.
func hostEUPackageOrigin() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.host.eu_package_origin",
		Dimension:   models.DimensionTechnologie,
		Description: "Installed packages source from EU-vendored distributions.",
		Rationale: "The package vendor is the upstream the host trusts for " +
			"binary updates, security patches, and the bill of " +
			"materials in the registry. A Fedora host trusts Red Hat " +
			"to ship `bash` and decide its security backports; an " +
			"openSUSE host trusts SUSE; a Debian host trusts the " +
			"Debian project. The rule names that trust without " +
			"forbidding any choice — operators pick distributions, " +
			"the assessor surfaces the jurisdiction the choice lands in.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var inspected, classified, usHits, euHits []models.Finding
			usByVendor := map[string][]string{}
			for _, f := range findings {
				if !strings.HasPrefix(f.ProbeID, "inventory.packages.") {
					continue
				}
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				inspected = append(inspected, f)
				rpmVendor, _ := f.Attributes["vendor"].(string)
				dpkgMaint, _ := f.Attributes["maintainer"].(string)
				v, ok := assessor.ClassifyPackageVendor(rpmVendor, dpkgMaint)
				if !ok {
					continue
				}
				classified = append(classified, f)
				switch v.Jurisdiction {
				case "us":
					usHits = append(usHits, f)
					usByVendor[v.ParentOrg] = append(usByVendor[v.ParentOrg], f.Subject)
				case "eu":
					euHits = append(euHits, f)
				}
			}
			if len(inspected) == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no inventory.packages.* findings — agent did not run or packages inspector disabled",
				}
			}
			if len(usHits) > 0 {
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  packageOriginUSVerdict(usByVendor, len(inspected)),
					Evidence: dedupeFindingIDs(usHits),
				}
			}
			if len(classified) > 0 && len(euHits) == len(classified) {
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d packages, %d classified — every classified package sources from an EU-tied vendor", len(inspected), len(classified)),
					Evidence: sampleEvidence(euHits),
				}
			}
			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("inspected %d packages, %d classified — no US-tied vendor observed, jurisdiction not fully attributable", len(inspected), len(classified)),
				Evidence: sampleEvidence(inspected),
			}
		},
	}
}

// packageOriginUSVerdict turns the per-parent_org subject map
// into a stable, alphabetically-sorted verdict string.
func packageOriginUSVerdict(byParent map[string][]string, totalInspected int) string {
	parents := make([]string, 0, len(byParent))
	for p := range byParent {
		parents = append(parents, p)
	}
	sort.Strings(parents)
	parts := make([]string, 0, len(parents))
	totalHits := 0
	for _, p := range parents {
		subjects := byParent[p]
		sort.Strings(subjects)
		totalHits += len(subjects)
		parts = append(parts, fmt.Sprintf("%s (%d packages, e.g. %s)", p, len(subjects), firstNJoined(subjects, 3)))
	}
	return fmt.Sprintf("inspected %d packages — %d sourced from US-tied vendors: %s", totalInspected, totalHits, strings.Join(parts, "; "))
}

func firstNJoined(items []string, n int) string {
	if len(items) > n {
		items = items[:n]
	}
	return strings.Join(items, ", ")
}

func dedupeFindingIDs(in []models.Finding) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, f := range in {
		if f.ID == "" || seen[f.ID] {
			continue
		}
		seen[f.ID] = true
		out = append(out, f.ID)
	}
	// Cap to the standard sample size so the verdict's evidence
	// list stays manageable on a host with 1000+ packages.
	if len(out) > hostSampleEvidenceCap {
		out = out[:hostSampleEvidenceCap]
	}
	return out
}

// hostNoUSTelemetryPackages flags installed packages whose name
// prefix-matches a known US-hosted telemetry / observability
// agent. Reads `inventory.packages.rpm` and
// `inventory.packages.dpkg` Findings.
//
// Soeverein when zero matches; afhankelijk on any match. The
// rule lists every matched package in the verdict so an
// operator can act on the specific finding.
func hostNoUSTelemetryPackages() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.host.no_us_telemetry_packages",
		Dimension:   models.DimensionTechnologie,
		Description: "Host carries no installed US-headquartered telemetry / observability agent.",
		Rationale: "Telemetry and observability agents (Datadog, New Relic, " +
			"AWS CloudWatch, Splunk, Dynatrace, Google Cloud Ops, Azure " +
			"Monitor, etc.) phone home to their vendor's control plane on " +
			"behalf of the host. When that control plane sits in a foreign " +
			"jurisdiction, the host's runtime telemetry — process names, " +
			"environment metadata, sometimes file contents — lands in a " +
			"system the operator does not control. The rule's job is not " +
			"to forbid these agents, but to make the dependency visible.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			hits := matchTelemetryHits(findings, "inventory.packages.")
			inspected := collectFindings(findings, func(f models.Finding) bool {
				return strings.HasPrefix(f.ProbeID, "inventory.packages.") && assessor.IsEvidenceLike(f)
			})
			if len(hits) == 0 {
				if len(inspected) == 0 {
					return assessor.RuleResult{
						Score:   models.ScoreOnbekend,
						Verdict: "no inventory.packages.* findings — agent did not run or packages inspector disabled",
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d packages — no US-headquartered telemetry agent installed", len(inspected)),
					Evidence: sampleEvidence(inspected),
				}
			}
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  hitsVerdict("installed package", hits),
				Evidence: hitsEvidence(hits),
			}
		},
	}
}

// hostNoUSTelemetryServices flags running systemd units whose
// name prefix-matches the same list. Catches binaries that were
// installed outside the package manager (tarball drop-in,
// container) but still expose themselves as systemd services.
func hostNoUSTelemetryServices() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.host.no_us_telemetry_services",
		Dimension:   models.DimensionTechnologie,
		Description: "Host runs no systemd unit matching a known US-headquartered telemetry agent.",
		Rationale: "Same dependency surface as the package-side rule, but " +
			"catches telemetry agents installed outside the package manager " +
			"— a tarball drop-in or a container's host-side companion that " +
			"still exposes itself via systemd. The two rules together cover " +
			"the realistic ways a US-vendored agent reaches a Linux host.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			hits := matchTelemetryHits(findings, "inventory.systemd.service")
			inspected := collectFindings(findings, func(f models.Finding) bool {
				return f.ProbeID == "inventory.systemd.service" && assessor.IsEvidenceLike(f)
			})
			if len(hits) == 0 {
				if len(inspected) == 0 {
					return assessor.RuleResult{
						Score:   models.ScoreOnbekend,
						Verdict: "no inventory.systemd.service findings — agent did not run or systemd inspector disabled",
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d systemd units — no US-headquartered telemetry agent running", len(inspected)),
					Evidence: sampleEvidence(inspected),
				}
			}
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  hitsVerdict("systemd unit", hits),
				Evidence: hitsEvidence(hits),
			}
		},
	}
}

type telemetryHit struct {
	subject string
	vendor  string
	finding models.Finding
}

// matchTelemetryHits walks findings whose ProbeID starts with
// `probePrefix` and returns the entries whose Subject matches
// any host_telemetry.yaml prefix. Stable order: alphabetical by
// subject so the verdict text and evidence list are
// deterministic across runs.
func matchTelemetryHits(findings []models.Finding, probePrefix string) []telemetryHit {
	var out []telemetryHit
	for _, f := range findings {
		if !strings.HasPrefix(f.ProbeID, probePrefix) {
			continue
		}
		if !assessor.IsEvidenceLike(f) {
			// Skip meta-rows (`error`, `unavailable`, etc.) so the
			// rule doesn't flag the absence of a real signal.
			continue
		}
		match, ok := assessor.HostTelemetryMatch(f.Subject)
		if !ok {
			continue
		}
		out = append(out, telemetryHit{
			subject: f.Subject,
			vendor:  match.Vendor,
			finding: f,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].subject < out[j].subject
	})
	return out
}

// collectFindings returns every finding for which pred is true.
// Used to gather the "we looked at this" evidence sample on the
// soeverein branch.
func collectFindings(findings []models.Finding, pred func(models.Finding) bool) []models.Finding {
	var out []models.Finding
	for _, f := range findings {
		if pred(f) {
			out = append(out, f)
		}
	}
	return out
}

// sampleEvidence returns up to hostSampleEvidenceCap finding IDs
// from the inspected slice. The host rules can produce thousands of
// inspected findings on a real host; carrying every ID inflates the
// Rationale row without helping the operator. A capped sample is
// enough to deep-link into a representative few.
const hostSampleEvidenceCap = 10

func sampleEvidence(inspected []models.Finding) []string {
	limit := len(inspected)
	if limit > hostSampleEvidenceCap {
		limit = hostSampleEvidenceCap
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if inspected[i].ID != "" {
			out = append(out, inspected[i].ID)
		}
	}
	return out
}

func hitsVerdict(kind string, hits []telemetryHit) string {
	if len(hits) == 1 {
		return fmt.Sprintf("%s %s matches %s", kind, hits[0].subject, hits[0].vendor)
	}
	subjects := make([]string, 0, len(hits))
	for _, h := range hits {
		subjects = append(subjects, h.subject)
	}
	return fmt.Sprintf("%d %ss match known US telemetry agents: %s", len(hits), kind, strings.Join(subjects, ", "))
}

func hitsEvidence(hits []telemetryHit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.finding.ID != "" {
			out = append(out, h.finding.ID)
		}
	}
	return out
}
