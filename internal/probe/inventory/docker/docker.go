// Package docker is the placeholder Docker inspector. The MVP of
// add-inventory-probe ships it as Available()=false unless an
// override is supplied — full Docker socket integration is deferred
// to a follow-up change. The package exists so the inspector
// registers itself as `inventory.docker.unavailable` on hosts
// without Docker, satisfying the "graceful unavailability" spec.
package docker

import (
	"context"
	"errors"
	"os"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Docker is the inspector.
type Docker struct {
	// Socket is the path to the Docker daemon socket. When empty
	// (default) the inspector is unavailable.
	Socket string
}

func (Docker) ID() string { return "docker" }

// Available reports whether the configured socket exists. We
// deliberately do not attempt a connection here — the
// "permission denied" branch surfaces during Inspect for now.
func (d Docker) Available() (bool, string) {
	if d.Socket == "" {
		return false, "docker socket not configured (MVP placeholder)"
	}
	if _, err := os.Stat(d.Socket); err != nil {
		if os.IsPermission(err) {
			return false, "permission denied on docker socket: " + err.Error()
		}
		return false, "docker socket not present: " + err.Error()
	}
	return false, "docker inspector not yet implemented in MVP"
}

// Inspect always returns an error in the MVP — Available() guards
// the call site so this should not be reached in normal operation.
func (d Docker) Inspect(_ context.Context) ([]models.Finding, error) {
	return nil, errors.New("docker: full inspector deferred to follow-up change")
}
