package eucsf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// containerSupplyChain is the SEAL analogue of the two wand
// docker rules. SEAL rolls supply-chain / vendor exposure into
// one observation, so this single rule walks both
// `inventory.docker.image` and `inventory.docker.container`
// findings and emits one combined verdict.
func containerSupplyChain() assessor.Rule {
	return assessor.Rule{
		ID:          "eucsf.sov6.container_supply_chain",
		Dimension:   models.DimensionTechnologie,
		Description: "Container supply chain (images + running containers) draws from no US-headquartered registries.",
		Rationale: "EUCSF sov-6 covers vendor dependency. For a host that " +
			"runs containers the supply chain is two-layered: the " +
			"set of images present on disk + the set of containers " +
			"actively running. The rule combines both into one SEAL " +
			"observation so the dashboard surfaces one supply-chain " +
			"verdict alongside the wand-side split.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var inspectedImg, inspectedCon []models.Finding
			var hits []containerHit
			for _, f := range findings {
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				switch f.ProbeID {
				case "inventory.docker.image":
					inspectedImg = append(inspectedImg, f)
					for _, ref := range repoTags(f) {
						if h, ok := classifyHit(ref, "image", f); ok {
							hits = append(hits, h)
						}
					}
				case "inventory.docker.container":
					inspectedCon = append(inspectedCon, f)
					if image, ok := f.Attributes["image"].(string); ok && image != "" {
						if h, ok := classifyHit(image, "container", f); ok {
							hits = append(hits, h)
						}
					}
				}
			}
			if len(inspectedImg)+len(inspectedCon) == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no inventory.docker.* findings — Docker inspector did not run or no containers present",
				}
			}
			if len(hits) == 0 {
				return assessor.RuleResult{
					Score: models.ScoreSoeverein,
					Verdict: fmt.Sprintf(
						"inspected %d images + %d containers — no image sourced from a US-headquartered registry [SEAL 4]",
						len(inspectedImg), len(inspectedCon),
					),
					Evidence: containerSampleEvidence(inspectedImg, inspectedCon),
				}
			}
			sort.Slice(hits, func(i, j int) bool {
				if hits[i].kind != hits[j].kind {
					return hits[i].kind < hits[j].kind
				}
				return hits[i].imgRef < hits[j].imgRef
			})
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  containerHitsVerdict(hits),
				Evidence: containerHitsEvidence(hits),
			}
		},
	}
}

type containerHit struct {
	kind    string
	imgRef  string
	vendor  string
	host    string
	finding models.Finding
}

func classifyHit(ref, kind string, f models.Finding) (containerHit, bool) {
	if ref == "" || ref == "<none>:<none>" {
		return containerHit{}, false
	}
	match, ok := assessor.ContainerRegistryMatch(ref)
	if !ok {
		return containerHit{}, false
	}
	host := match.Registry.Host
	if match.ImpliedDockerIO {
		host += " (implicit)"
	}
	return containerHit{
		kind:    kind,
		imgRef:  ref,
		vendor:  match.Registry.Vendor,
		host:    host,
		finding: f,
	}, true
}

func repoTags(f models.Finding) []string {
	if tags, ok := f.Attributes["repo_tags"].([]string); ok {
		return tags
	}
	if tags, ok := f.Attributes["repo_tags"].([]any); ok {
		out := make([]string, 0, len(tags))
		for _, t := range tags {
			if s, ok := t.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if f.Subject != "" {
		return []string{f.Subject}
	}
	return nil
}

func containerHitsVerdict(hits []containerHit) string {
	if len(hits) == 1 {
		h := hits[0]
		return fmt.Sprintf("%s %s pulls from %s (%s) [SEAL 1]", h.kind, h.imgRef, h.host, h.vendor)
	}
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		parts = append(parts, fmt.Sprintf("%s %s", h.kind, h.imgRef))
	}
	return fmt.Sprintf("%d container supply-chain hits: %s [SEAL 1]", len(hits), strings.Join(parts, ", "))
}

func containerHitsEvidence(hits []containerHit) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.finding.ID == "" || seen[h.finding.ID] {
			continue
		}
		seen[h.finding.ID] = true
		out = append(out, h.finding.ID)
	}
	return out
}

const containerSampleCap = 10

func containerSampleEvidence(images, containers []models.Finding) []string {
	merged := append([]models.Finding{}, images...)
	merged = append(merged, containers...)
	limit := len(merged)
	if limit > containerSampleCap {
		limit = containerSampleCap
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if merged[i].ID != "" {
			out = append(out, merged[i].ID)
		}
	}
	return out
}
