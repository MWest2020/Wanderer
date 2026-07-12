---
status: draft
last_reviewed: 2026-07-12
---

# 0014 — Nextcloud marketplace distribution via AppAPI ExApp

**Status:** Accepted, 2026-06-15. Resolves
`openspec/changes/propose-nextcloud-marketplace-app` (direction 4 of
the four-direction Nextcloud integration). Chosen over the proposal's
options A/B/C.

**Context.** Direction 4 asked whether Wanderer should be installable
from the Nextcloud app marketplace. The proposal framed three
architectures on the premise that *Nextcloud apps are PHP*: A (PHP
shim + Go sidecar), B (PHP reimplementation), C (Go→WASM embedded in
PHP). All three were unattractive — A leaks "install two services", B
rewrites everything in the wrong language, C depends on a non-standard
`ext-wasm` runtime — so the proposal recommended deferring.

That premise is obsolete. Nextcloud's **AppAPI** runs **ExApps
(External Apps)** as Docker containers, managed by a Deploy Daemon,
language-agnostic, and distributable through the App Store. AppAPI is
a default dependency since Nextcloud 30.0.1.

**Decision.** Distribute Wanderer as an **AppAPI ExApp** — call it
**architecture D**. Wanderer is already a single pure-Go binary, so it
containerises trivially. A thin Go shim implements the AppAPI
lifecycle the Deploy Daemon expects and fronts the existing Wanderer
HTTP API:

- `GET /heartbeat` (unauthenticated) — liveness; reflects whether the
  colocated Wanderer process is actually reachable.
- `POST /init`, `PUT /enabled?enabled=` (AppAPIAuth) — lifecycle hooks.
- everything else (AppAPIAuth) — reverse-proxied to `wanderer serve`
  on the container loopback.
- Incoming auth is the `AUTHORIZATION-APP-API` shared secret
  (base64 `userid:secret`) the Deploy Daemon injects as `APP_SECRET`.

**Two repos, one-way dependency.** The ExApp surface ships as a
**separate downstream repo, `MWest2020/wanderer-exapp`**, NOT in this
core repo. This honours the marketplace change's spec requirement:
the core Go module gains no PHP/Composer/Nextcloud-app dependency and
`go test ./...` stays toolchain-free. The downstream repo consumes the
core by `go install`-ing a pinned version of `cmd/wanderer` into its
image; a release-triggered GitHub Action (`repository_dispatch`) bumps
the pin so the ExApp tracks core releases. The dependency runs one way
— exactly like the Billbird ← billbird-client ← Gitsweeper split.

Why a separate repo rather than a subdirectory: the core's API lives
under `internal/` (module-private, not importable by another module),
so there is no clean in-repo import path anyway; and the ExApp has its
own release cadence and marketplace quality bar. A pinned-binary
dependency is the honest coupling.

**Consequences.**

- Marketplace distribution is feasible at far lower cost than the
  ~2-week PHP-shim estimate for option A — the shim is a few hundred
  lines of Go and the core needs zero changes.
- The "operators see Wanderer in Nextcloud" goal is now covered at
  three levels: as-target (scan a Nextcloud), as-output (publish scans
  into Nextcloud, ADR-adjacent / exporters spec), and — with this —
  installable *inside* Nextcloud.
- A core release must trigger the downstream rebuild for the ExApp to
  stay current; that wiring (a dispatch step in the core's release
  workflow) is a follow-up once the core tags releases. Until then the
  ExApp pins `WANDERER_VERSION=main`.
- **Validated end-to-end (2026-06-15).** The ExApp was smoke-tested
  against a live Nextcloud 30.0.17 + AppAPI 4.0.6: the image builds
  (buildx), the container runs the shim + the pinned core binary, and
  AppAPI registered it via a `manual-install` daemon — authenticating
  with the real `AUTHORIZATION-APP-API` secret, calling `/init`, and
  ending with `app_api:app:list` showing `wanderer … [enabled]`. The
  runtime contract (auth, lifecycle, proxy) is proven.
- The ExApp `info.xml` manifest is still a template — the live
  validation registered the app via `--json-info` (the runtime path),
  not `info.xml` (the App-Store packaging path). The AppAPI manifest
  schema drifted across versions (3.0+ deprecated `<scopes>`/`<system>`)
  and must be validated against the target AppAPI version before the
  app is published to the store.

**Alternatives considered.**

- *A — PHP shim + Go sidecar*: rejected. AppAPI's Deploy Daemon already
  manages the container, so the "two services" problem D was meant to
  avoid simply doesn't arise; a hand-written PHP shim adds a second
  language and release cadence for no gain over D.
- *B — PHP reimplementation*: rejected outright (rewrites every probe
  and rule in the wrong language; test coverage from zero).
- *C — Go→WASM in PHP*: rejected. `ext-wasm` is non-standard and the
  probe networking surface (raw sockets, TLS) isn't exposed by
  Nextcloud's PHP/WASI runtime.
- *Subdirectory in the core repo*: rejected — `internal/` blocks
  in-module import, and it would risk dragging container/manifest
  concerns into the core's CI. A separate repo keeps the boundary
  crisp.

**See also.**

- `openspec/changes/.../propose-nextcloud-marketplace-app` — the
  decision-space proposal this ADR resolves
- `MWest2020/wanderer-exapp` — the downstream ExApp repo (the spike)
- ADR-0013 (Nextcloud as OIDC) — sibling Nextcloud-integration decision
- Nextcloud AppAPI / ExApp docs:
  https://docs.nextcloud.com/server/latest/admin_manual/exapps_management/AppAPIAndExternalApps.html
