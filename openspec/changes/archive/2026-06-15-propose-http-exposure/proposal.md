# Proposal: HTTP exposure scoring (the "what is exposed / misusable" axis)

> **Status:** Accepted + implemented (2026-06-15). The misuse/exposure
> lead from research-high-signal-observability. Passive: it scores
> signals the HTTP probe already observed — no active/intrusive probing.

## Why

The HTTP probe already captures the baseline security headers
(`http.security_headers` present/missing) and the Server / X-Powered-By
banner (`http.response`), but nothing scored them. Missing HSTS leaves
a site open to transport downgrade; a version banner hands an attacker
the exact stack to target. These are the cheapest, most concrete
"exposed / misusable" signals and were a blind spot.

## What Changes

- A new `wand.operationeel.http_exposure` rule: no HSTS → afhankelijk;
  HSTS present but other baseline headers missing → voldoende; all
  present → soeverein. A Server / X-Powered-By stack disclosure is
  named in the verdict.

## Not in scope

- Active path probing (/.git, /.env, /server-status) or any intrusive
  check — Wanderer stays a passive observer of the operator's own target.

## Parallel-safe

One new rule file + registration + an assessor spec requirement. No
probe, schema, or collection change.
