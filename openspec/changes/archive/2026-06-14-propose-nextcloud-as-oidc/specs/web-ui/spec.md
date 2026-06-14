# Delta for web-ui

> Accepted 2026-06-14 — implemented and ready to archive into the
> canonical web-ui spec.

## ADDED Requirements

### Requirement: Wanderer can accept Nextcloud as the OIDC provider for `wanderer serve --ui`

Wanderer SHALL, when the `serve.yaml` `oidc:` block is
configured, redirect unauthenticated `/ui/*` requests to the
configured Nextcloud's authorize endpoint, exchange the
returned code on the callback URL, and set a session cookie
keyed against a server-side SQLite session table. A
Nextcloud-side disable of the user SHALL cut off Wanderer UI
access on the next request.

#### Scenario: Authenticated browse honours the Nextcloud session

- **GIVEN** OIDC is configured against `cloud.example.nl` and
  an operator has authenticated successfully
- **WHEN** they request `/ui/orgs/conduction`
- **THEN** the page renders without an additional login
  prompt

#### Scenario: Nextcloud-side disable cuts Wanderer access

- **GIVEN** an authenticated operator's Nextcloud account is
  disabled while their Wanderer session cookie is still
  valid
- **WHEN** they request any `/ui/*` page
- **THEN** the request redirects to `/ui/login` and the
  Nextcloud authorize endpoint refuses re-authentication

#### Scenario: OIDC outage leaves htpasswd fallback usable

- **GIVEN** OIDC is configured AND htpasswd is also
  configured AND the OIDC provider is unreachable
- **WHEN** an operator requests `/ui/` with
  HTTP Basic credentials matching htpasswd
- **THEN** the request renders normally
