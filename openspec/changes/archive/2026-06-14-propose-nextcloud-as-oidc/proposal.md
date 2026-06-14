# Proposal: Nextcloud as OIDC — accept Nextcloud login for wanderer serve

> **Status:** Accepted + implemented (2026-06-14). Direction (3)
> of the four-direction Nextcloud integration proposal, picked by
> Mark over `as-output` and `marketplace-app`. All four open
> questions resolved as recommended; see ADR-0013 and tasks.md.

## Intent

Today `wanderer serve --ui` authenticates via HTTP Basic
against an htpasswd file. That's the
boring-MVP stop-gap (memory: "Authentication is HTTP Basic
against an htpasswd file"). For organisations running a
self-hosted Nextcloud as their identity hub, the htpasswd
flow is a duplicate trust store: every operator needs two
accounts, and offboarding is "delete from htpasswd" instead
of "deactivate in Nextcloud".

This change accepts Nextcloud as an OIDC provider for the
read-only UI — operators sign in once at their Nextcloud,
the wanderer UI honours their Nextcloud session, and
deactivation in Nextcloud cuts off Wanderer access
automatically.

## Scope (depends on Q1)

The Nextcloud `user_oidc` app exposes a standard OIDC
provider endpoint. Wanderer becomes the OIDC client.
Mechanically:

- New `internal/auth/oidc/` package implementing the
  authorization-code flow.
- New `[oidc]` block in `serve.yaml` with
  `provider_url`, `client_id`, `client_secret_file`,
  `scopes`, `redirect_url`.
- `/ui/login` redirects to the Nextcloud authorize endpoint;
  `/ui/oauth/callback` exchanges the code + sets a session
  cookie.
- htpasswd remains as a fallback for the no-Nextcloud case.

## Open questions

1. **Session model.** Cookie-backed session table in SQLite
   OR stateless JWT signed by Wanderer? Recommendation:
   **cookie + SQLite session table.** Stateless JWT means
   revocation needs a blocklist; SQLite session table makes
   "Nextcloud-side disable revokes Wanderer access on next
   request" a trivial JOIN. Boring + auditable.

2. **Authorisation scope.** OIDC authentication only ("you
   are someone Nextcloud knows"), OR group-based
   authorisation ("you must be in the `wanderer-operators`
   Nextcloud group")? Recommendation: **authentication only
   in the first wave.** Group-based authorisation lands when
   a customer asks; until then, the existing read-only UI is
   already access-controlled at the network layer.

3. **Multi-tenant Wanderer + multi-Nextcloud.** If a
   Wanderer instance serves three customer organisations,
   each with their own Nextcloud, can each organisation use
   *its* Nextcloud as the IdP? Recommendation: **single OIDC
   provider per Wanderer instance for now.** Multi-IdP is a
   second-wave concern; the organisation pivot already
   handles multi-tenancy at the data layer.

4. **OIDC client secret handling.** The secret has to live
   somewhere — env var, file path, sealed in serve.yaml? The
   existing pattern is `hmac_secret_file: /path/to/secret`
   for the agent's remote mode. Recommendation:
   **`client_secret_file: <path>` mirroring the existing
   convention.**

## Risks

- **Authentication outage = UI unusable.** If Nextcloud goes
  down, no operator can log in. Mitigation: htpasswd
  fallback stays available for break-glass access. Documented
  in operator.md.
- **Session-cookie hijack.** Standard mitigations apply:
  `Secure`, `HttpOnly`, `SameSite=Strict`, short TTL, IP
  pinning optional.
- **Nextcloud user_oidc app drift.** The Nextcloud OIDC app
  is community-maintained; its endpoints could shift. We
  treat it as a standard OIDC provider, so drift surfaces as
  "OIDC discovery fails" not "Wanderer doesn't know about a
  new field".

## Not in scope

- SAML / LDAP. OIDC only.
- Replacing htpasswd entirely — kept as the no-OIDC
  fallback.
- Authorisation rules beyond authentication (the read-only
  UI is uniform across users).

## Parallel-safe

Touches `internal/auth/` (new), `internal/serveconfig/`,
`internal/ui/` (the login + callback routes), docs. No
schema change beyond an optional sessions table that lives
only when the OIDC block is enabled.
