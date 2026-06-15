# 0016 — Dev-mode scan form in the UI (read-only stays the prod default)

**Status:** Accepted, 2026-06-15. Implements `propose-ui-dev-scan`.

**Context.** The `/ui/` was deliberately read-only — a static test
blocks every mutating handler — so the browse layer had minimal attack
surface and all mutation went through the CLI/API. In practice that
made the UI a no-go for the day-to-day loop: you could not enter a
target and scan it (or check your own host) from the browser; you had
to drop to `wanderer scan` on the CLI. Right now Wanderer runs locally,
so the real distinction is **dev mode vs prod mode**, not "exposed vs
not".

**Decision.** Add an **opt-in dev mode**: `wanderer serve --ui-allow-scan`
mounts a single sanctioned mutating route, `POST /ui/scan`, and renders
a "Scan a target" form on the dashboard. The route scans the submitted
target, assesses it with the wand pack, and redirects to
`/ui/scans/<id>/assessment` so the Sovereignty overview + diagram
populate immediately. **Default off → the UI stays fully read-only (the
prod default).** The read-only static test is tightened, not removed:
PUT/PATCH/DELETE remain forbidden and `/scan` is the only permitted
POST.

Security posture: the scanner's SSRF guard still applies (no private /
metadata targets unless `--allow-private-targets`); scan IDs are
server-generated (no open redirect). When `--ui-allow-scan` is set with
no UI auth, serve logs a warning — for anything beyond localhost, gate
it behind the OIDC/htpasswd auth (ADR-0013). Mark accepted the
unauthenticated local default for dev.

## UI surface

- With `--ui-allow-scan`, the dashboard shows a "Scan a target" form
  (`form.scan-form`); submitting it POSTs to `/ui/scan` and lands on the
  scan's assessment page (overview + diagram).
- Without the flag, neither the form nor the route exists — the UI is
  read-only.

The Playwright spec `tests/playwright/specs/ui-dev-scan.spec.ts` asserts
the form renders against a serve instance started with `--ui-allow-scan`.

**Consequences.**

- One mutating route in a previously read-only package; bounded by the
  opt-in flag and the tightened read-only test.
- Synchronous scan: the POST blocks until the scan + assess finish
  (seconds). Acceptable for a dev convenience; a background/queued scan
  is a possible follow-up.

**Alternatives considered.**

- *Always-on scan form* — rejected: silently reverses the read-only
  security posture for every deployment. Opt-in keeps prod safe.
- *Form POSTs to the API `/scans`* — rejected: the API returns JSON, a
  poor browser UX, and it would not assess + redirect to the overview.

**See also.** ADR-0013 (UI auth — gate this when exposed);
`research-high-signal-observability` (the overview the scan lands on).
