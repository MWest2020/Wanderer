# Proposal: Full Docker inspector

## Intent

The MVP of `add-inventory-probe` shipped a Docker inspector that
returns `Available()=false` so it surfaces as
`inventory.docker.unavailable` on hosts with Docker. That fulfilled
the "graceful unavailability" spec scenario but produces no actual
container observations. This change wires the real Docker socket
read path: containers, images, and their pull origins so the
assessor and operator dashboards can answer "what containers are
running here, where do their images come from".

## Scope

**In scope:**

- Read-only HTTP-over-unix-socket calls against the Docker Engine
  API (`/v1.41/containers/json`, `/v1.41/images/json`).
- `inventory.docker.container` Findings with `image`,
  `image_digest`, `state`, `created_at`, `labels`.
- `inventory.docker.image` Findings with `digest`, `repo_tags`,
  `size_bytes`, `created_at`.
- Permission-denied and socket-missing branches surface as
  `inventory.docker.unavailable` (existing behaviour, unchanged).
- Stdlib-only HTTP transport via `net.Dial("unix", socketPath)` —
  no Docker SDK dependency.

**Out of scope:**

- Docker Compose project introspection. Compose files live
  outside the daemon; reading them is the egress probe's
  config-file scanner job.
- Image-layer inspection (running scanners against image layers
  for CVE detection). That is a separate change.
- Swarm / Kubernetes orchestration. Docker socket only here.
- Mutating calls (`docker run|pull|rm` equivalents). The
  inventory probe is read-only — see the existing
  `Agent runs read-only` requirement.

## DICTU dimensions informed

- **Operationeel** (primary): which containers run, lifecycle state.
- **Technologie**: image origins, vendor concentration in
  registries.

## Passive/active boundary

Local-only. The inspector talks to a unix domain socket; it makes
no outbound network calls.

## Parallel-safe

Touches only `internal/probe/inventory/docker/`. No schema
changes.
