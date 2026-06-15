# Tasks: Wanderer as a Nextcloud marketplace app

## 1. Direction decisions

- [x] 1.1 Q1 — is marketplace distribution worth pursuing? **Yes**,
  via architecture **D (AppAPI / ExApp)** — the A/B/C PHP framing was
  obsolete. Go container, no PHP, core untouched.
- [x] 1.2 Q2 — who maintains the surface? A **separate downstream
  repo** `MWest2020/wanderer-exapp`, kept current via a
  release-triggered sync Action; its own release cadence.
- [x] 1.3 Q3 — sidecar/lifecycle: the AppAPI **Deploy Daemon** runs
  the ExApp container; the shim spawns the colocated Wanderer binary
  and proxies to it. No separate manual service for the admin.

## 2. Implementation

- [x] 2.1 Architecture chosen (D) and recorded — ADR-0014.
- [x] 2.2 Spike built in `MWest2020/wanderer-exapp`: AppAPI shim
  (heartbeat/init/enabled + AppAPIAuth), Dockerfile (pinned
  `go install` of core), ExApp manifest template, dev compose, CI +
  sync workflow, unit tests. Go shim builds/vets/tests green.
- [ ] 2.3 Live AppAPI deploy against a real Nextcloud — NOT yet
  validated (no Docker in the authoring env); the downstream repo's
  `deploy/docker-compose.dev.yml` is the harness. Tracked downstream.

## 3. Wrap-up

- [x] 3.1 Decision recorded in the proposal status block + ADR-0014.
- [x] 3.2 Escalated to a separate repo (not archived as no-go);
  this design change is archived here as decided, implementation
  tracked downstream.
