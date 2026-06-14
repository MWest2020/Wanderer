# Tasks: Nextcloud as OIDC

## 1. Direction decisions

- [x] 1.1 Q1 — session model. **Decided: cookie + SQLite session
  table** (`ui_sessions`, migration 006). See ADR-0013.
- [x] 1.2 Q2 — authorisation scope. **Decided: authentication-only
  first wave.**
- [x] 1.3 Q3 — multi-tenant Wanderer + multi-Nextcloud. **Decided:
  single OIDC provider per Wanderer instance.**
- [x] 1.4 Q4 — client secret handling. **Decided:
  `client_secret_file: <path>`** (mirrors `hmac_secret_file`).

## 2. Implementation skeleton

- [x] 2.1 New `internal/auth/oidc/` package with the
  authorization-code flow (go-oidc/v3 + x/oauth2; lazy discovery).
- [x] 2.2 New session table (migration 006) + gate enforcing
  session presence on `/ui/*`, with per-request userinfo
  revalidation for revocation.
- [x] 2.3 `/ui/login` redirect + `/ui/oauth/callback` handler
  (+ `/ui/logout`), state cookie + nonce.
- [x] 2.4 `serve.yaml` `oidc:` block + validation (partial block
  rejected at startup).
- [x] 2.5 htpasswd fallback documented (operator.md → "Nextcloud
  login (OIDC)" + ADR-0013).
- [x] 2.6 Playwright spec asserting unauthenticated request to
  `/ui/` redirects to `/ui/login` (`oidc-login.spec.ts`); full
  exchange + revocation covered by Go tests
  (`internal/auth/oidc/oidc_test.go`).

## 3. Wrap-up

- [x] 3.1 Commit.
- [x] 3.2 Archive.
