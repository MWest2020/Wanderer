# Proposal: container image sovereignty

> **Status:** Implementation. Decisions locked per auto-mode +
> "boring + auditable + verkoopbaar" framing. Status block at
> the top of this file records what was picked and why; the
> open-question record is preserved below.

## Resolved decisions

1. **Implicit `docker.io` is treated as a docker.io hit.** An
   image reference `nginx` or `library/nginx` (no registry
   prefix) is the Docker Engine's shorthand for
   `docker.io/library/nginx`. The rule's job is to make hidden
   dependencies visible; ignoring the bare-name case would hide
   the largest single dependency in the ecosystem.
2. **Match list as YAML, embedded via `go:embed`** — mirrors
   the egress probe's `vendors.yaml` and the host telemetry
   `host_telemetry.yaml`. Operator-visible, reviewable in one
   place. New file:
   `internal/assessor/container_registries.yaml`.
3. **Soeverein with negative evidence sample** — same pattern
   the host + Nextcloud rules already use: cite up to 10
   inspected `inventory.docker.image` Finding IDs and include
   the inspected count in the Verdict text.
4. **Two wand rules, one eucsf rule.** The wand split keeps
   container scoring next to host scoring conceptually (a
   container image is a vendor dependency much like a
   telemetry agent is). EUCSF rolls the container surface into
   the existing sov6 supply-chain dimension.

## Intent

The Docker inspector already emits two finding families:

- `inventory.docker.container` — running containers, with
  `image` attribute carrying the ref the container was started
  from
- `inventory.docker.image` — every image present on the host,
  with `repo_tags` carrying the human-readable refs

Neither scores today. A Wanderer agent running on a host that
pulls 30 images from `gcr.io` and `mcr.microsoft.com` produces
two finding families' worth of evidence and zero verdict — the
same gap host-side scoring closed for packages + systemd units.

This change adds:

- `wand.docker.images_us_registry` — afhankelijk when any
  observed image's registry resolves to a US-headquartered
  registry from the YAML list. Soeverein on a clean host;
  onbekend without inventory data.
- `wand.docker.containers_us_registry` — same shape, but
  reads `inventory.docker.container` findings so a freshly-
  pulled-but-not-yet-started image does not score harder than
  the actually-running ones.
- `eucsf.sov6.container_supply_chain` — SEAL analogue rolling
  both shapes into one observation, sibling to
  `eucsf.sov6.nextcloud_supply_chain`.

## Scope

**In scope:**

- New `internal/assessor/container_registries.yaml` with the
  known US registries (`gcr.io`, `docker.io`,
  `index.docker.io`, `ghcr.io`, `mcr.microsoft.com`,
  `public.ecr.aws`, `quay.io`, `registry.suse.com`).
- New `internal/assessor/container_registries.go` loader +
  `MatchRegistry(imageRef)` helper that returns the matched
  registry entry. Handles bare names (`nginx` → docker.io
  implicit) + library shorthand (`library/nginx` → docker.io).
- Two wand rules (`images_us_registry`,
  `containers_us_registry`) + one eucsf rule
  (`container_supply_chain`).
- Tests for the loader, the helper, and the rules.
- Agent-host Playwright fixture gains one
  `inventory.docker.image` finding citing `gcr.io/foo/bar` so
  the rule rows show afhankelijk.
- New Playwright spec.
- Operator-doc paragraph.

**Out of scope:**

- Container *origin* attribution (image signature / SBOM /
  vulnerability scan). Different problem class; outside
  Wanderer's "where does data flow" charter.
- Image-layer scanning. Wanderer reads list-of-images
  metadata only; cracking layer manifests requires
  registry-side credentials and a different threat model.
- Multi-arch awareness. The rule reads tags as
  registry/image:tag strings; arch suffixes (`...:1.27-arm64`)
  are folded into the same registry classification.

## Risks

- **Registry-list drift.** Vendors get acquired
  (Quay → Red Hat → IBM = US still; GitHub → Microsoft = US
  still). The YAML is small enough to keep current by hand;
  any operator can override at runtime via an env var if Mark
  decides to add that hook later. For now embedded-only.
- **False negatives on EU self-hosted registries.** A
  `harbor.example.de` image scores soeverein because it does
  not match the US list. That's correct.
- **False positives on bare names.** Bare `nginx` =
  docker.io implicit. An operator who runs an EU-mirrored
  registry but pulls via the bare name without changing
  containerd config will see afhankelijk. Verdict text names
  the registry explicitly so the operator can correct.

## Parallel-safe

Touches `internal/assessor/`,
`internal/assessor/wand/`, `internal/assessor/eucsf/`,
`internal/fixtures/`, `tests/playwright/specs/`, docs. No
agent change, no schema change, no UI change required.
