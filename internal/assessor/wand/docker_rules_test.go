package wand

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func dockerImage(id, subject string, tags []string) models.Finding {
	return models.Finding{
		ID:       id,
		ProbeID:  "inventory.docker.image",
		Subject:  subject,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"repo_tags": tags,
		},
	}
}

func dockerContainer(id, name, image string) models.Finding {
	return models.Finding{
		ID:       id,
		ProbeID:  "inventory.docker.container",
		Subject:  name,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"image": image,
		},
	}
}

func TestDockerImagesUSRegistry_Soeverein(t *testing.T) {
	r := ruleByID(t, "wand.docker.images_us_registry")
	got := r.Match([]models.Finding{
		dockerImage("i1", "harbor.example.de/team/app:v3", []string{"harbor.example.de/team/app:v3"}),
		dockerImage("i2", "harbor.example.de/team/other:v1", []string{"harbor.example.de/team/other:v1"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("EU registry: score = %s, want soeverein", got.Score)
	}
	if len(got.Evidence) == 0 {
		t.Error("soeverein must cite negative evidence")
	}
	if !strings.Contains(got.Verdict, "inspected 2 images") {
		t.Errorf("verdict = %q must include inspected count", got.Verdict)
	}
}

func TestDockerImagesUSRegistry_GcrIo(t *testing.T) {
	r := ruleByID(t, "wand.docker.images_us_registry")
	got := r.Match([]models.Finding{
		dockerImage("i1", "harbor.example.de/team/app:v3", []string{"harbor.example.de/team/app:v3"}),
		dockerImage("i2", "gcr.io/foo/bar:v1", []string{"gcr.io/foo/bar:v1"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("gcr.io: score = %s, want afhankelijk", got.Score)
	}
	if !strings.Contains(got.Verdict, "gcr.io/foo/bar:v1") {
		t.Errorf("verdict = %q must name the offending image", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "Google") {
		t.Errorf("verdict = %q must name the vendor", got.Verdict)
	}
	if len(got.Evidence) != 1 || got.Evidence[0] != "i2" {
		t.Errorf("evidence = %v, want [i2]", got.Evidence)
	}
}

func TestDockerImagesUSRegistry_BareNameIsDockerIoImplicit(t *testing.T) {
	r := ruleByID(t, "wand.docker.images_us_registry")
	got := r.Match([]models.Finding{
		dockerImage("i1", "nginx:1.27", []string{"nginx:1.27"}),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("bare name: score = %s, want afhankelijk (docker.io implicit)", got.Score)
	}
	if !strings.Contains(got.Verdict, "docker.io") {
		t.Errorf("verdict = %q must name docker.io", got.Verdict)
	}
	if !strings.Contains(got.Verdict, "implicit") {
		t.Errorf("verdict = %q must flag the implicit resolution", got.Verdict)
	}
}

func TestDockerImagesUSRegistry_NoneIsSkipped(t *testing.T) {
	// Docker emits <none>:<none> for built-but-not-tagged images.
	// Those shouldn't count as US-registry hits.
	r := ruleByID(t, "wand.docker.images_us_registry")
	got := r.Match([]models.Finding{
		dockerImage("i1", "harbor.example.de/team/app:v3", []string{"harbor.example.de/team/app:v3"}),
		dockerImage("i2", "<none>:<none>", []string{"<none>:<none>"}),
	})
	if got.Score != models.ScoreSoeverein {
		t.Errorf("<none>:<none>: score = %s, want soeverein", got.Score)
	}
}

func TestDockerImagesUSRegistry_NoFindingsIsOnbekend(t *testing.T) {
	r := ruleByID(t, "wand.docker.images_us_registry")
	got := r.Match(nil)
	if got.Score != models.ScoreOnbekend {
		t.Errorf("no findings: score = %s, want onbekend", got.Score)
	}
}

func TestDockerContainersUSRegistry_RunningGcr(t *testing.T) {
	r := ruleByID(t, "wand.docker.containers_us_registry")
	got := r.Match([]models.Finding{
		dockerContainer("c1", "web", "gcr.io/foo/bar:v1"),
	})
	if got.Score != models.ScoreAfhankelijk {
		t.Errorf("running gcr: score = %s, want afhankelijk", got.Score)
	}
}

func TestDockerContainersUSRegistry_ImageFindingsIgnored(t *testing.T) {
	r := ruleByID(t, "wand.docker.containers_us_registry")
	got := r.Match([]models.Finding{
		dockerImage("i1", "gcr.io/foo/bar:v1", []string{"gcr.io/foo/bar:v1"}),
	})
	if got.Score != models.ScoreOnbekend {
		t.Errorf("only image findings: score = %s, want onbekend (containers rule ignores images)", got.Score)
	}
}

func TestDefaultRules_RegistersDockerRules(t *testing.T) {
	want := map[string]bool{
		"wand.docker.images_us_registry":     false,
		"wand.docker.containers_us_registry": false,
	}
	for _, r := range DefaultRules() {
		if _, ok := want[r.ID]; ok {
			want[r.ID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("docker rule %q not registered", id)
		}
	}
}
