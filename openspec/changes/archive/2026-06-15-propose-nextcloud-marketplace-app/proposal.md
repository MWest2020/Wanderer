# Proposal: Wanderer as a Nextcloud marketplace app

> **Status:** Decided + spiked (2026-06-15). **Go — architecture D
> (AppAPI / ExApp), which supersedes A/B/C.** The A/B/C options
> assumed "Nextcloud apps are PHP"; Nextcloud's AppAPI lets an ExApp
> ship as a language-agnostic Docker container managed by the deploy
> daemon and distributed via the App Store — no PHP, core untouched.
> Implemented as a separate downstream repo `MWest2020/wanderer-exapp`
> (skeleton spike: AppAPI shim + packaging). See ADR-0014.

## Intent

Nextcloud's app marketplace is the discovery path for
Nextcloud admins looking to add capability. A "Wanderer:
sovereignty observation for your Nextcloud" app would let an
admin install + configure Wanderer through their existing
admin UI, see the verdicts directly in the Nextcloud
sidebar, and never run a separate Go binary.

The mechanical question is bigger than the other three
directions combined: Nextcloud apps are PHP. Wanderer is
pure Go. There are three plausible architectures, each with
different trade-offs.

## Architecture options

### A. PHP shim app + Go sidecar

The marketplace app is a thin PHP wrapper. It manages
configuration (which targets to scan, when, by which
organisation) via the standard Nextcloud admin UI; it
delegates actual scanning to a Wanderer binary that runs as
a sidecar (separate systemd unit, separate port, separate
process). The PHP app talks to the sidecar over the existing
HTTP API + MCP transport.

Pros: keeps the Go codebase intact, adds only a marketplace-
discoverable surface. Cons: customer needs to install
*two* services (Nextcloud app + Wanderer sidecar) — the
"marketplace install = ready to go" promise leaks.

### B. PHP-only reimplementation

Rewrite the probe set in PHP so the entire thing runs as a
pure Nextcloud app, no sidecar. The dependency on a Go
build chain goes away.

Pros: pure marketplace install. Cons: catastrophic — every
sovereignty rule, every probe, every parser needs to be
ported. Test coverage starts at zero. The Go codebase
diverges from the PHP one. Reject this option.

### C. Compile Wanderer to WebAssembly, embed in PHP

`tinygo` can produce a WASM binary that PHP loads via
`ext-wasm` or a sidecar runtime. The Go code stays
authoritative; the PHP app embeds the binary.

Pros: zero-config marketplace install. Cons: `ext-wasm` is
not standard Nextcloud; customer needs to add the PHP
extension. Performance impact is real (WASM probes are
slower than native). The networking surface (egress probes,
TLS probes) requires the host filesystem + raw sockets,
which WASI partially provides but Nextcloud's PHP runtime
doesn't expose by default.

### D. AppAPI ExApp — Go container (CHOSEN, 2026-06-15)

The premise behind A/B/C — "Nextcloud apps are PHP" — is
obsolete. Nextcloud's **AppAPI** runs **ExApps (External
Apps)** as Docker containers managed by a Deploy Daemon:
language-agnostic, App-Store-distributable, AppAPI a default
dependency since NC 30.0.1. Wanderer (one Go binary) ships as
an ExApp image; a thin Go shim implements the AppAPI lifecycle
(`/heartbeat`, `/init`, `/enabled`) and reverse-proxies authed
Nextcloud traffic to the colocated Wanderer process.

Pros: no PHP rewrite (vs B); the Deploy Daemon manages the
container so there is no "install two services" leak (vs A);
no `ext-wasm` runtime gap (vs C); the Go core stays
authoritative and untouched. Cons: the customer's Nextcloud
must have AppAPI + a configured deploy daemon (standard on
modern Nextcloud).

Shipped as a **separate downstream repo**
`MWest2020/wanderer-exapp` — honouring the spec requirement
that the marketplace surface not pollute the core Go module
(no PHP/Composer; `go test ./...` stays toolchain-free). The
repo consumes the core via a pinned `go install` version that
a release-triggered Action bumps. Verified:
`docs/decisions/0013`-style ADR at ADR-0014; AppAPI contract
grounded against the Nextcloud AppAPI docs.

Source: https://docs.nextcloud.com/server/latest/admin_manual/exapps_management/AppAPIAndExternalApps.html

## Recommendation

**Superseded — see architecture D above (chosen).** The
original recommendation, retained for the record, was to defer
the go/no-go. The three PHP-framed options tell us:

- A is feasible today (sidecar, ~2 weeks of PHP shim work)
- B is a non-starter (rewrites everything in the wrong
  language)
- C is interesting but the runtime situation isn't there
  yet

If marketplace distribution becomes a business priority,
ship A. If not, the existing CLI + systemd story plus the
direction-(2) Nextcloud-as-output integration covers the
"operators see Wanderer output in Nextcloud" goal at lower
cost.

## Open questions

1. **Is marketplace distribution a business priority?**
   This is the gate. Mark's call.

2. **If yes, who maintains the PHP shim?** A Nextcloud app
   has its own release cadence, its own PHP version matrix
   (Nextcloud requires PHP 8.1+), its own quality bar in
   the marketplace review.

3. **What does the sidecar lifecycle look like?** systemd
   unit shipped alongside the app? Container that the
   admin starts manually? Nextcloud-managed process?

## Not in scope

- The actual implementation. This proposal documents the
  decision space; the implementation proposal lives in
  `add-nextcloud-marketplace-app` IF Mark picks Q1=yes.

## Parallel-safe

This proposal is paper-only. Implementation would touch new
top-level directories (`nextcloud-app/` or similar) — design
to be worked out per the picked architecture.
