# Proposal: Wanderer as a Nextcloud marketplace app

> **Status:** Design pass — awaiting Mark's scope call.
> Direction (4) of the four-direction Nextcloud integration
> proposal. This is the **heaviest** direction by an order
> of magnitude. Mark may legitimately decide this is a
> separate product, not a Wanderer feature.

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

## Recommendation

**Defer the decision.** Marketplace distribution is a
go-to-market choice, not a technical one. The three options
above tell us:

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
