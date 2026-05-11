package eucsf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// hostNoUSTelemetry is the SEAL analogue of the wand
// host-telemetry pair. SEAL rolls supply-chain / vendor
// exposure into one observation, so this single rule walks
// both `inventory.packages.*` and `inventory.systemd.service`
// findings against the same `host_telemetry.yaml` list.
func hostNoUSTelemetry() assessor.Rule {
	return assessor.Rule{
		ID:          "eucsf.sov5.host_no_us_telemetry",
		Dimension:   models.DimensionTechnologie,
		Description: "No US-headquartered telemetry / observability agent installed or running on the host.",
		Rationale: "Supply-chain exposure on a host that an organisation " +
			"operates inside the EU: even when the perimeter looks " +
			"sovereign, a US-vendored agent running on the host pipes " +
			"runtime telemetry back to a foreign control plane. SEAL " +
			"sov-5 covers this dimension of vendor dependency.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var hits []hostHit
			var inspected []models.Finding
			var pkgCount, svcCount int
			for _, f := range findings {
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				switch {
				case strings.HasPrefix(f.ProbeID, "inventory.packages."):
					pkgCount++
				case f.ProbeID == "inventory.systemd.service":
					svcCount++
				default:
					continue
				}
				inspected = append(inspected, f)
				if match, ok := assessor.HostTelemetryMatch(f.Subject); ok {
					hits = append(hits, hostHit{
						kind:    inputKind(f.ProbeID),
						subject: f.Subject,
						vendor:  match.Vendor,
						id:      f.ID,
					})
				}
			}
			if pkgCount == 0 && svcCount == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no inventory.packages.* / inventory.systemd.service findings — agent did not run or inspectors disabled",
				}
			}
			if len(hits) == 0 {
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("inspected %d packages + %d systemd units — no US-headquartered telemetry agent installed or running [SEAL 4]", pkgCount, svcCount),
					Evidence: sampleInspectedEvidence(inspected),
				}
			}
			sort.Slice(hits, func(i, j int) bool {
				if hits[i].kind != hits[j].kind {
					return hits[i].kind < hits[j].kind
				}
				return hits[i].subject < hits[j].subject
			})
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  hostHitsVerdict(hits),
				Evidence: hostHitsEvidence(hits),
			}
		},
	}
}

type hostHit struct {
	kind    string
	subject string
	vendor  string
	id      string
}

func inputKind(probeID string) string {
	if strings.HasPrefix(probeID, "inventory.packages.") {
		return "package"
	}
	return "service"
}

func hostHitsVerdict(hits []hostHit) string {
	if len(hits) == 1 {
		return fmt.Sprintf("%s %s matches %s [SEAL 1]", hits[0].kind, hits[0].subject, hits[0].vendor)
	}
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		parts = append(parts, h.kind+" "+h.subject)
	}
	return fmt.Sprintf("%d host telemetry hits: %s [SEAL 1]", len(hits), strings.Join(parts, ", "))
}

func hostHitsEvidence(hits []hostHit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.id != "" {
			out = append(out, h.id)
		}
	}
	return out
}

// hostSampleEvidenceCap mirrors the wand-side cap. Soeverein on a
// real host means thousands of inspected findings; we only want a
// handful in Rationale so the UI deep-link list stays browsable.
const hostSampleEvidenceCap = 10

func sampleInspectedEvidence(inspected []models.Finding) []string {
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
