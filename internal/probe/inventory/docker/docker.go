// Package docker is the Docker inventory inspector. It reads the
// running containers and pulled images from a Docker daemon over a
// unix domain socket. The inspector is read-only: it issues only GET
// requests against the Engine API. Mutating endpoints
// (`/exec`, `/start`, `/stop`, ...) are off-limits per the
// "Agent runs read-only" requirement.
package docker

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Docker is the inspector.
type Docker struct {
	// Socket is the path to the Docker daemon socket, typically
	// /var/run/docker.sock. When empty the inspector is unavailable.
	Socket string
}

// ID returns the inspector identifier used in ProbeID prefixes.
func (Docker) ID() string { return "docker" }

// Available reports whether the configured socket exists and is
// readable. We only stat the file: the actual connect happens during
// Inspect, where its error becomes an inventory.docker.error
// Finding via the inventory orchestrator.
func (d Docker) Available() (bool, string) {
	if d.Socket == "" {
		return false, "docker socket not configured"
	}
	if _, err := os.Stat(d.Socket); err != nil {
		if os.IsPermission(err) {
			return false, "permission denied on docker socket: " + err.Error()
		}
		if os.IsNotExist(err) {
			return false, "docker socket not present at " + d.Socket
		}
		return false, "docker socket: " + err.Error()
	}
	return true, ""
}

// Inspect lists containers and images. A daemon error bubbles up to
// the inventory orchestrator, which produces a single
// inventory.docker.error Finding while letting the other inspectors
// run.
func (d Docker) Inspect(ctx context.Context) ([]models.Finding, error) {
	if d.Socket == "" {
		return nil, errors.New("docker: socket not configured")
	}
	c := newClient(d.Socket)

	containers, err := c.listContainers(ctx)
	if err != nil {
		return nil, err
	}
	images, err := c.listImages(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]models.Finding, 0, len(containers)+len(images))
	for _, ct := range containers {
		out = append(out, containerFinding(ct))
	}
	for _, img := range images {
		out = append(out, imageFinding(img))
	}
	return out, nil
}

func containerFinding(c container) models.Finding {
	name := strings.TrimPrefix(firstName(c.Names), "/")
	if name == "" {
		name = shortID(c.ID)
	}
	attrs := map[string]any{
		"image":        c.Image,
		"image_digest": c.ImageID,
		"state":        c.State,
		"status":       c.Status,
		"created_at":   time.Unix(c.Created, 0).UTC().Format(time.RFC3339),
	}
	if len(c.Labels) > 0 {
		attrs["labels"] = c.Labels
	}
	return models.Finding{
		ProbeID:       "inventory.docker.container",
		DimensionHint: models.DimensionOperationeel,
		Subject:       name,
		Severity:      models.SeverityInfo,
		Attributes:    attrs,
	}
}

func imageFinding(i image) models.Finding {
	subject := firstString(i.RepoTags)
	if subject == "" || subject == "<none>:<none>" {
		subject = shortID(i.ID)
	}
	attrs := map[string]any{
		"digest":     i.ID,
		"size_bytes": i.Size,
		"created_at": time.Unix(i.Created, 0).UTC().Format(time.RFC3339),
	}
	if len(i.RepoTags) > 0 {
		// Sort for determinism — Docker returns RepoTags in
		// arbitrary order; a stable list keeps Finding diffs auditable.
		tags := append([]string(nil), i.RepoTags...)
		sort.Strings(tags)
		attrs["repo_tags"] = tags
	}
	return models.Finding{
		ProbeID:       "inventory.docker.image",
		DimensionHint: models.DimensionTechnologie,
		Subject:       subject,
		Severity:      models.SeverityInfo,
		Attributes:    attrs,
	}
}

func firstName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func firstString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// shortID returns the first 12 chars of a docker ID, mimicking the
// CLI's truncated form so log lines and Finding subjects stay short.
func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// IsAPIError reports whether err is a non-2xx response from the
// Docker API.
func IsAPIError(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr)
}

// StatusCode extracts the HTTP status from an API error, or 0 if err
// is not an api error. Useful for tests.
func StatusCode(err error) int {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}
