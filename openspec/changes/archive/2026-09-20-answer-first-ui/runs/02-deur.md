# Habitat run 02 — de deur (tasks 2.1–2.3)

Contract: `openspec/changes/2026-09-20-answer-first-ui/`
(specs/web-ui/spec.md, requirements "The entry surface asks for a
domain and answers it" and the MODIFIED "UI surface stays read-only").

## Scope — ONLY these tasks
- [ ] 2.1 `/ui/` leads with one input that takes a domain and, on
  submit, starts a scan and redirects to that target's answer page.
  Under it: the recently answered targets, one line each, each line a
  verdict from run 01's function — not a table of scans. The fleet
  table, the rollups and the matrix move off this first screen (they
  stay reachable; `/ui/trends` keeps them).
- [ ] 2.2 The scan route is gated on a signed-in user, not on
  `--ui-allow-scan`: with OIDC or htpasswd configured a signed-in user
  may scan; with neither, the route is refused and the startup log says
  so once, naming why. Keep the static-analysis test that refuses new
  mutating routes — the scan route is the one sanctioned exception.
- [ ] 2.3 A host that reports through an agent (see `wanderer agent`
  and the host-side findings) is selectable from the same input, so
  "this system" is as easy as "this domain".

## Out of scope
The answer page itself (run 03) and the reasoning page (run 04). Land
on the existing scan page for now if the answer page does not exist
yet; do not build a second one.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` green.
Budget is $5.
