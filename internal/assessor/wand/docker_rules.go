package wand

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// dockerImagesUSRegistry reads `inventory.docker.image` findings
// and classifies their image references against the embedded
// US-registry list. Soeverein on a clean host with negative
// evidence sample; afhankelijk on any US-registry hit;
// onbekend without findings.
func dockerImagesUSRegistry() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.docker.images_us_registry",
		Dimension:   models.DimensionTechnologie,
		Description: "Host has no container images sourced from a US-headquartered registry.",
		Rationale: "Container images are runtime supply-chain dependencies. " +
			"An image pulled from a US-headquartered registry routes " +
			"the layer download — and any future updates — through a " +
			"control plane in a foreign jurisdiction. The rule names " +
			"the registry without forbidding the image: operators " +
			"choose registries, the assessor surfaces the choice.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			return dockerRegistryMatch(findings, "inventory.docker.image", "image", imageRefsFromImageFinding)
		},
	}
}

// dockerContainersUSRegistry mirrors the image rule but reads
// `inventory.docker.container` findings — the actually-running
// surface, which is a stronger signal than "image is on disk".
func dockerContainersUSRegistry() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.docker.containers_us_registry",
		Dimension:   models.DimensionTechnologie,
		Description: "Host runs no containers from a US-headquartered registry.",
		Rationale: "What is actively running is a stronger signal than what " +
			"is merely available on disk: a US-registry image that " +
			"sits unused does not carry sovereignty risk today, but " +
			"a running container does. This rule scopes the verdict " +
			"to the live surface so the operator sees where data is " +
			"actually flowing right now.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			return dockerRegistryMatch(findings, "inventory.docker.container", "container", imageRefsFromContainerFinding)
		},
	}
}

// imageRefsFromImageFinding returns the image refs to classify
// from an `inventory.docker.image` Finding. Docker reports
// repo_tags as a `[]string` Attribute; tagged + untagged
// (`<none>:<none>`) images both appear here.
func imageRefsFromImageFinding(f models.Finding) []string {
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
	// Fallback: the subject itself when no repo_tags
	// (Docker emits `<none>:<none>` shortened digests as Subject).
	if f.Subject != "" {
		return []string{f.Subject}
	}
	return nil
}

// imageRefsFromContainerFinding returns the image ref a running
// container was started from. The Docker inspector puts it on
// the `image` attribute.
func imageRefsFromContainerFinding(f models.Finding) []string {
	if s, ok := f.Attributes["image"].(string); ok && s != "" {
		return []string{s}
	}
	return nil
}

type dockerHit struct {
	subject string
	imgRef  string
	vendor  string
	host    string
	finding models.Finding
}

func dockerRegistryMatch(findings []models.Finding, probeID, kind string, refs func(models.Finding) []string) assessor.RuleResult {
	var inspected []models.Finding
	var hits []dockerHit
	for _, f := range findings {
		if f.ProbeID != probeID {
			continue
		}
		if !assessor.IsEvidenceLike(f) {
			continue
		}
		inspected = append(inspected, f)
		for _, ref := range refs(f) {
			if ref == "" || ref == "<none>:<none>" {
				continue
			}
			match, ok := assessor.ContainerRegistryMatch(ref)
			if !ok {
				continue
			}
			hits = append(hits, dockerHit{
				subject: f.Subject,
				imgRef:  ref,
				vendor:  match.Registry.Vendor,
				host:    dockerHostLabel(match),
				finding: f,
			})
		}
	}
	if len(inspected) == 0 {
		return assessor.RuleResult{
			Score:   models.ScoreOnbekend,
			Verdict: fmt.Sprintf("no %s findings — Docker inspector did not run or no %ss are present", probeID, kind),
		}
	}
	if len(hits) == 0 {
		return assessor.RuleResult{
			Score:    models.ScoreSoeverein,
			Verdict:  fmt.Sprintf("inspected %d %ss — no image sourced from a US-headquartered registry", len(inspected), kind),
			Evidence: dockerSampleEvidence(inspected),
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].host != hits[j].host {
			return hits[i].host < hits[j].host
		}
		return hits[i].imgRef < hits[j].imgRef
	})
	return assessor.RuleResult{
		Score:    models.ScoreAfhankelijk,
		Verdict:  dockerHitsVerdict(kind, hits),
		Evidence: dockerHitsEvidence(hits),
	}
}

func dockerHostLabel(m assessor.RegistryMatch) string {
	if m.ImpliedDockerIO {
		return m.Registry.Host + " (implicit)"
	}
	return m.Registry.Host
}

func dockerHitsVerdict(kind string, hits []dockerHit) string {
	if len(hits) == 1 {
		h := hits[0]
		return fmt.Sprintf("%s %s pulls from %s (%s)", kind, h.imgRef, h.host, h.vendor)
	}
	refs := make([]string, 0, len(hits))
	for _, h := range hits {
		refs = append(refs, h.imgRef)
	}
	return fmt.Sprintf("%d %ss pull from US-headquartered registries: %s", len(hits), kind, strings.Join(refs, ", "))
}

func dockerHitsEvidence(hits []dockerHit) []string {
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

const dockerSampleEvidenceCap = 10

func dockerSampleEvidence(inspected []models.Finding) []string {
	limit := len(inspected)
	if limit > dockerSampleEvidenceCap {
		limit = dockerSampleEvidenceCap
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if inspected[i].ID != "" {
			out = append(out, inspected[i].ID)
		}
	}
	return out
}
