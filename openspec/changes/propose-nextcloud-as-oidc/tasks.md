# Tasks: Nextcloud as OIDC (design pending)

## 1. Direction decisions

- [ ] 1.1 Q1 — session model. Recommendation: cookie +
  SQLite session table.
- [ ] 1.2 Q2 — authorisation scope. Recommendation:
  authentication-only first wave.
- [ ] 1.3 Q3 — multi-tenant Wanderer + multi-Nextcloud.
  Recommendation: single OIDC provider per Wanderer
  instance for the first wave.
- [ ] 1.4 Q4 — client secret handling. Recommendation:
  `client_secret_file: <path>`.

## 2. Implementation skeleton

- [ ] 2.1 New `internal/auth/oidc/` package with the
  authorization-code flow.
- [ ] 2.2 New session table (migration 006) + middleware
  enforcing session presence on `/ui/*`.
- [ ] 2.3 `/ui/login` redirect + `/ui/oauth/callback`
  handler.
- [ ] 2.4 `serve.yaml` `[oidc]` block + validation.
- [ ] 2.5 htpasswd fallback documented.
- [ ] 2.6 Playwright spec asserting unauthenticated request
  to `/ui/` redirects to `/ui/login`.

## 3. Wrap-up

- [ ] 3.1 Commit + push.
- [ ] 3.2 Archive.
