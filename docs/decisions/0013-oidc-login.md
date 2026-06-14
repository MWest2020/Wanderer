# 0013 — Nextcloud as OIDC login for `wanderer serve --ui`

**Status:** Accepted, 2026-06-14. Implements
`openspec/changes/propose-nextcloud-as-oidc` (direction 3 of the
four-direction Nextcloud integration). Mark picked this direction
over `as-output` and the `marketplace-app`.

**Context.** `/ui/` authenticates with HTTP Basic against an
htpasswd file. For an organisation running a self-hosted Nextcloud
as its identity hub, htpasswd is a duplicate trust store: every
operator needs a second account, and offboarding is "delete from
htpasswd" instead of "deactivate in Nextcloud". The Nextcloud
`user_oidc` app exposes a standard OIDC provider endpoint, so
Wanderer can become an OIDC client and honour the Nextcloud
session.

**Decision.** The four open questions from the proposal are
resolved as recommended, and three of them are load-bearing enough
to record here.

1. **Session model: server-side SQLite, not stateless JWT.** A
   successful login sets an opaque `HttpOnly` cookie keyed against
   a row in the new `ui_sessions` table (migration 006). A
   stateless JWT in the cookie would need a revocation blocklist;
   a server-side row is deleted directly. Boring and auditable —
   the live sessions are a `SELECT` away on disk.

2. **Revocation by per-request userinfo revalidation.** The spec
   requires that a Nextcloud-side disable cut off access "on the
   next request". We honour that by re-validating the session
   against the provider's `userinfo` endpoint (refreshing the
   access token first; a revoked refresh token already fails
   here). The `revalidate_interval` defaults to `0s` — every
   request — so the disable propagates immediately. Operators who
   find the per-request userinfo call too chatty can raise it; the
   trade-off (immediacy vs IdP load) is theirs, documented in
   operator.md. We deliberately did **not** invent a cheaper
   "trivial JOIN" revocation: that only works if the disable
   writes into Wanderer's own DB, which it does not.

3. **Lazy discovery, htpasswd as break-glass.** `wanderer serve`
   must not hard-fail when Nextcloud is unreachable, or an IdP
   outage takes the whole UI down with it. So OIDC discovery is
   deferred to the first `/ui/login` — the server boots regardless
   — and when both `oidc:` and `htpasswd:` are configured, valid
   Basic credentials still pass the gate. htpasswd stays as the
   no-OIDC fallback and the outage escape hatch; it is not removed.

4. **Authentication-only, single provider (first wave).** Any user
   the IdP knows can browse the read-only UI; group-based
   authorisation (`wanderer-operators`) lands when a customer asks.
   One OIDC provider per Wanderer instance — multi-IdP for
   multi-tenant deployments is a second-wave concern that the
   organisation pivot's data-layer multi-tenancy does not require
   us to solve now.

5. **The client secret lives in a file, not YAML.**
   `client_secret_file: <path>` mirrors the agent's existing
   `hmac_secret_file` convention. The secret is never serialised
   into `serve.yaml`.

**Library choice.** The OIDC mechanics compose
`github.com/coreos/go-oidc/v3` with `golang.org/x/oauth2` — the
de-facto standard, audited Go libraries — rather than a hand-rolled
authorization-code flow. `internal/auth/oidc` is a thin wrapper
(discovery, exchange, ID-token verification, userinfo revalidation)
that knows nothing about chi, cookies, or sessions; the HTTP wiring
and the session→htpasswd→redirect policy live in `internal/ui`.

**Cookies are `SameSite=Lax`, not `Strict`.** The proposal floated
`Strict`, but the OIDC callback is a top-level cross-site
navigation back from Nextcloud, and `Strict` would withhold the
state cookie on that request, breaking the flow. `Lax` is the
correct setting for a redirect-based login. State (CSRF) is still
covered: a random state is pinned to the browser in a cookie and
recorded server-side, double-checked on callback, and a per-login
nonce binds the ID token.

## UI surface

- An unauthenticated request to any `/ui/*` page, when `oidc:` is
  configured, redirects (`302`) to `/ui/login`.
- `/ui/login`, `/ui/oauth/callback`, `/ui/logout`, and `/ui/static/*`
  bypass the gate (otherwise `/ui/login` would redirect to itself).
- An established session renders pages without a fresh login
  prompt; logout deletes the session row and clears the cookie.

The Playwright spec `tests/playwright/specs/oidc-login.spec.ts`
runs against a fixture serve instance configured with an
unreachable provider (discovery is lazy, so the redirect path never
contacts it) and asserts the `302 → /ui/login` redirect plus that
the login route itself is not gated. The full code-exchange and
revocation paths are covered by Go tests in `internal/auth/oidc`
against a mock provider (`oidc_test.go`).

**Consequences.**

- New dependency surface: `go-oidc/v3`, `x/oauth2`, and the
  transitive `go-jose/v4`. All are widely used and maintained.
- One new migration (006, `ui_sessions`). The table is created
  unconditionally so the schema is uniform across builds, but it
  stays empty unless an operator configures `oidc:`.
- The per-request userinfo call at the default `revalidate_interval`
  of `0s` adds one provider round-trip per page load. For a
  low-traffic operator UI this is acceptable; it is tunable.
- **Revalidation is fail-closed and does not distinguish "account
  disabled" from "provider unreachable":** any error from the
  userinfo/refresh round-trip deletes the session. This is the
  secure default for a sovereignty tool — an IdP outage must not
  leave a possibly-revoked session alive — but at `0s` it means a
  transient IdP blip logs every operator out (htpasswd break-glass
  covers the gap). **Backlog:** if outage-tolerance is wanted later,
  distinguish a `401`/`invalid_grant` (delete) from a transport
  error (keep the session, deny this request, retry next time).

**Alternatives considered.**

- *Stateless JWT session cookie*: rejected — revocation needs a
  blocklist, which is more moving parts than a deletable row.
- *Hard discovery at startup*: rejected — couples server boot to
  IdP availability and defeats the break-glass story.
- *Replacing htpasswd entirely*: rejected — it is the no-OIDC
  fallback and the outage escape hatch.
- *PKCE on top of the confidential-client secret*: deferred — a
  confidential client with state + nonce is the boring-standard
  Nextcloud setup; PKCE adds code without a threat it closes here.

**See also.**

- `openspec/changes/propose-nextcloud-as-oidc/` — the proposal +
  spec delta this ADR implements
- ADR-0007 (Agent trust model) — the `hmac_secret_file` convention
  the client-secret handling mirrors
- `docs/operator.md` → "Nextcloud login (OIDC)" — operator setup,
  revocation behaviour, and the htpasswd break-glass note
