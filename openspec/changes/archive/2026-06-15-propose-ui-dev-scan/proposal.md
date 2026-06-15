# Proposal: dev-mode scan form in the UI

> **Status:** Accepted + implemented (2026-06-15). Opt-in; read-only
> stays the prod default. See ADR-0016.

## Why

The read-only UI couldn't enter or scan a target — a no-go for the
local day-to-day loop (enter a target / check your own host, then see
the sovereignty overview). Locally the real distinction is dev vs prod.

## What Changes

- `wanderer serve --ui-allow-scan` (default off) mounts `POST /ui/scan`
  and a "Scan a target" form on the dashboard. The route scans +
  assesses (wand) + redirects to the assessment page (overview +
  diagram). Read-only stays the default; the static read-only test is
  tightened to allow only this one POST.

## Not in scope / safeguards

- SSRF guard unchanged (no private targets without --allow-private-targets).
- No forced auth (local dev); serve warns when on without auth — gate
  behind OIDC/htpasswd when exposed.

## Parallel-safe

`internal/ui` (Options.Scanner + the route/form) + a serve flag + the
read-only test tightened + a Playwright assertion. No schema/probe/rule change.
