package assessor

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed container_registries.yaml
var containerRegistriesYAML []byte

// ContainerRegistry is one entry from container_registries.yaml.
type ContainerRegistry struct {
	Host   string `yaml:"host"`
	Vendor string `yaml:"vendor"`
}

type containerRegistriesFile struct {
	Registries []ContainerRegistry `yaml:"registries"`
}

var (
	containerRegistriesOnce sync.Once
	containerRegistries     []ContainerRegistry
	containerRegistriesErr  error
)

func loadContainerRegistries() ([]ContainerRegistry, error) {
	containerRegistriesOnce.Do(func() {
		var f containerRegistriesFile
		if err := yaml.UnmarshalStrict(containerRegistriesYAML, &f); err != nil {
			containerRegistriesErr = fmt.Errorf("assessor: parse container_registries.yaml: %w", err)
			return
		}
		containerRegistries = f.Registries
	})
	return containerRegistries, containerRegistriesErr
}

// RegistryMatch represents the result of classifying an image
// reference: the matched registry, the explicit registry host
// that was used (so the verdict can report "docker.io implicit"
// for bare names), and whether a match was found.
type RegistryMatch struct {
	Registry        ContainerRegistry
	ResolvedHost    string // the registry host the imageRef resolved to
	ImpliedDockerIO bool   // true when the imageRef carried no registry prefix
}

// ContainerRegistryMatch classifies a container image reference
// against the embedded US-registries list. Image refs without a
// registry prefix are folded to `docker.io` per the Docker
// Engine's default; the match's ImpliedDockerIO flag preserves
// that information for the verdict.
//
// The boolean return is true iff the resolved registry host is
// in the embedded list.
func ContainerRegistryMatch(imageRef string) (RegistryMatch, bool) {
	if _, err := loadContainerRegistries(); err != nil {
		return RegistryMatch{}, false
	}
	host, implicit := resolveImageRegistry(imageRef)
	hostLower := strings.ToLower(host)
	for _, r := range containerRegistries {
		if strings.EqualFold(r.Host, hostLower) {
			return RegistryMatch{
				Registry:        r,
				ResolvedHost:    host,
				ImpliedDockerIO: implicit,
			}, true
		}
	}
	return RegistryMatch{ResolvedHost: host, ImpliedDockerIO: implicit}, false
}

// resolveImageRegistry extracts the registry host from a Docker
// image reference. The Docker reference grammar:
//
//	[host[:port]/][namespace/]name[:tag][@digest]
//
// A host is present when the first segment (before the first `/`)
// contains a `.` or a `:` (i.e. looks like a hostname / a port).
// Otherwise the ref is `[namespace/]name`, which the Docker
// Engine folds to `docker.io/library/<name>` or
// `docker.io/<namespace>/<name>`.
func resolveImageRegistry(imageRef string) (host string, implicit bool) {
	// Strip tag / digest first.
	if i := strings.IndexAny(imageRef, "@"); i >= 0 {
		imageRef = imageRef[:i]
	}
	if i := strings.LastIndex(imageRef, ":"); i >= 0 {
		// Don't strip a port from a host portion — only strip the
		// tag, which always sits AFTER the last `/`.
		lastSlash := strings.LastIndex(imageRef, "/")
		if i > lastSlash {
			imageRef = imageRef[:i]
		}
	}

	firstSlash := strings.Index(imageRef, "/")
	if firstSlash < 0 {
		// Bare name: nginx, alpine, etc. → docker.io/library/<name>.
		return "docker.io", true
	}
	candidate := imageRef[:firstSlash]
	// docker.io implicit if the first segment isn't a hostname-
	// shaped thing.
	if !strings.ContainsAny(candidate, ".:") && candidate != "localhost" {
		return "docker.io", true
	}
	return candidate, false
}

// ContainerRegistryEntries returns the loaded list (for tests +
// rule-description text).
func ContainerRegistryEntries() []ContainerRegistry {
	list, _ := loadContainerRegistries()
	return list
}
