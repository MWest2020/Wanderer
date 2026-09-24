# Habitat run 01 — de vlootlaag in beeld (tasks 1.1–1.7)

Contract: `openspec/changes/2026-09-24-vloot-in-beeld/` — read proposal.md
(Mark's words and what the page shows today), design.md, and the MODIFIED
requirement in specs/web-ui/spec.md. ADR-0018
(`docs/explanation/adr/0018-ui-three-layers.md`) says who this layer is for:
the Tourist, a CISO who wants to see at a glance where the fleet stands.

## Where things are
- Template: `internal/ui/templates/dashboard.tmpl`. The scan form is at the
  bottom (`<form class="scan-form" ... action="/ui/scan">`, ~line 127).
- Fleet aggregate: `internal/ui/fleet_score.go` (`TopRules` via
  `TopConcerns`, aggregate.go), per-domain flows: `SovereigntyFlows` /
  `classifyFlows` in `internal/ui/flows.go`. Handelingen:
  `wand.HandelingFor(ruleID)`.
- Dashboard view model: `internal/ui/ui.go` (the struct with `TopRules`,
  ~line 193) and the handlers that fill it.
- CSS: the UI's own stylesheet under `internal/ui/static/`; the verdict
  classes `score-*` exist already, light and dark.

## Decisions already made — do not reopen them
1. **No JavaScript, no chart library, no external asset.** Inline SVG + CSS
   from the Go templates. Geometry (ring segments, bar widths) is computed in
   Go and unit-tested, not in template arithmetic.
2. **Every visual carries its facts as text** (aria-label on the ring, a label
   per bar, a `title` per grid cell). Tests assert those texts, not pixels.
3. **No rationale on `/ui/`.** The top 3 show flow + handeling (with
   `{domein}` left out) + "x van n domeinen"; fall back to the rule ID, never
   to the English rationale or description.
4. **Scan form at the top**, compact, directly under the page header; the
   `allowScan` condition that decides whether it renders stays as it is.
5. **Scope is `/ui/` and `/ui/orgs/{slug}` only.** Do not touch the answer
   page, the assessment page, Trends or the fleet management screen.

## Scope — ONLY these tasks
- [ ] 1.1–1.6 as in tasks.md.
- [ ] 1.7 Go tests for every scenario in the spec delta; update the existing
      Playwright specs that assert the old layout (run
      `grep -rn "zwaarste\|scan-form\|Kost de vloot" tests/playwright/specs`),
      and add one Playwright assertion that the scan field precedes the fleet
      score. Check each new test once with its fix removed, and say so.
      CHANGELOG under [Unreleased].

## Out of scope
Release, deploy, ADR update, archive (tasks 2.x).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green and
`openspec validate 2026-09-24-vloot-in-beeld --strict` green. Playwright and
golangci-lint you cannot run in the cage; say so, they run afterwards.
Budget $9.
