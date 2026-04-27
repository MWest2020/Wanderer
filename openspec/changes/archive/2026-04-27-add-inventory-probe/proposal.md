# Proposal: Inventory Probe — What Actually Runs Inside

## Intent

External observation tells us what the world sees. It cannot tell us
what is running on our own infrastructure. A sovereignty picture that
ignores "which containers, packages, and services do we operate" is
half-blind — organisations routinely discover that a pilot project
quietly installed a SaaS connector, that a legacy server runs an
unmaintained PHP version, or that a helm chart pulled in a US-based
observability agent.

This change introduces **`wanderer agent`**: a second operational mode
that runs *on* a host inside the organisation, observes what is
installed and running, and writes Findings to the Wanderer store.
Same `Finding` shape, same DICTU dimension hints — the assessor and
the CLI do not care which modus produced a Finding.

## Scope

**In scope:**

- New binary modus `wanderer agent` — same executable as `wanderer
  scan|serve|mcp`, selected by subcommand.
- New probe package `internal/probe/inventory/` with inspectors for:
  - **systemd** services (name, unit file path, `LoadedActiveSubState`)
  - **Docker** containers (via `/var/run/docker.sock`)
  - **dpkg/rpm** installed packages
  - **Nextcloud apps** via `occ app:list` (opt-in, for Conduction-stack
    hosts)
- Agent writes to the Wanderer store directly (same SQLite file) or,
  when the core runs elsewhere, posts Findings to a configured core
  via `POST /scans/{id}/findings` (new endpoint, landed in this change).
- Config file `wanderer-agent.yaml`: list of inspectors to enable,
  their paths, and target Wanderer core (local DB or remote URL).
- Read-only. The agent never modifies system state.

**Out of scope:**

- **Kubernetes inspection.** Deferred to a follow-up change; requires
  a separate trust model (service account, RBAC). Design supports
  adding a `k8s` inspector later without restructuring.
- **File-content SBOM.** We record *installed packages*, not every
  file hash. SBOM-in-SPDX-JSON is a plausible future export on top
  of inventory Findings.
- **Remote execution / orchestration.** Each agent runs on its host,
  inspects that host. No central scheduler invoking agents — that is
  `add-scheduling` combined with per-host cron.
- **Windows hosts.** Linux-only for the MVP of this change; add-on
  later if demand exists.

## DICTU dimensions informed

- **Operationeel** (primary): what runs, which versions, which are
  EOL, who patches.
- **Technologie**: which vendors' components are actually deployed,
  complementing the external `http.third_party` picture.
- **Data & AI** (partial): observed database engines, object-storage
  clients — although *where* data goes is `add-egress-probe`, not
  this change. Inventory answers "what is installed"; egress answers
  "where does it point".

## Passive/active boundary

The agent runs locally and reads local state: Unix sockets, config
files, system utilities (`dpkg -l`, `systemctl list-units`, `docker
ps`). It makes **no** outbound network calls except to the configured
Wanderer core endpoint when agent-remote mode is used.

`docker ps` via the Docker socket requires group membership (`docker`)
or root. `systemctl` requires nothing. `occ` requires read access to
the Nextcloud config. These are all credential-adjacent —
documentation must be explicit about the principle of least privilege:
run the agent as a dedicated `wanderer-agent` user with the minimum
group memberships for the enabled inspectors.

## Parallel-safe

Mostly independent. The cross-cutting touch points:

- `pkg/models.Finding` — no change (inventory findings reuse the
  existing shape). Documentation update only.
- `cmd/wanderer/main.go` — one new `agent` case entry.
- `internal/store` — possibly one new column to tag Findings by their
  **source modus** (`perimeter | inventory | egress`) so the assessor
  can report coverage. This is the one merge-conflict risk with the
  assessor change; we resolve it by adding the column in this change
  and having the assessor treat its absence as `perimeter`.
- New HTTP endpoint `POST /scans/{id}/findings` for remote agents —
  lives in `internal/api`.

## Trust model

Two deployment shapes:

1. **Co-located**: agent runs on the same host as `wanderer serve`,
   writes to the same SQLite file. Simplest; suitable for a single
   Nextcloud-stack server where Wanderer observes itself.
2. **Split**: agent on each host in a fleet, core on one control
   machine. Agents post Findings over HTTPS with a shared secret
   (HMAC-signed request body, not bearer tokens — no token-reuse risk
   across hosts). A future change replaces HMAC with workload-identity
   if the fleet outgrows it.

ADR `0006-agent-trust-model.md` captures the HMAC-over-TLS choice and
why we rejected mTLS for the MVP (operational cost of cert rotation
without gain for a ≤10-host fleet).
