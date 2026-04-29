# Tasks: Refresh architecture docs

## 1. Rewrite

- [x] 1.1 Replace `docs/architecture.md` content per the design outline
- [x] 1.2 Mermaid diagram for the three-modi triad
- [x] 1.3 Per-modus prose sections with ProbeID prefix names

## 2. How-to sections

- [x] 2.1 "How to add a perimeter probe" — refreshed for current code
- [x] 2.2 "How to add an inventory inspector" — new
- [x] 2.3 "How to add an egress scanner" — new
- [x] 2.4 "How to add a DICTU rule" — refreshed

## 3. Cross-references

- [x] 3.1 Link every per-capability doc from architecture.md
- [x] 3.2 Update `docs/README.md` index ordering if needed
- [x] 3.3 Verify each link target exists with `find` + `grep`

## 4. CHANGELOG

- [x] 4.1 Entry under `### Changed` (docs only)

## Notes

- The proposal listed an explicit `docs/ui.md` link target. That
  page does not exist in the repository (the read-only UI is
  documented inline in architecture.md and operationally in
  `docs/operator.md`), so the link was redirected to
  `docs/operator.md` rather than fabricating an empty file.
- 3.2 (README ordering) was a no-op — the existing
  [`docs/README.md`](../docs/README.md) index already lists pages
  alphabetically with architecture as the first entry, which matches
  the new triad-first framing.
- 3.3 verified by listing each linked path; every target resolves
  (`docs/{agent,assessor,drift,egress,exporters,findings,maintainability,mcp,observability,operator,scheduling,tutorial}.md`,
  `docs/decisions/`, `openspec/specs/scanner/spec.md`,
  `internal/probe/egress/vendors.yaml`, `internal/agent/config.go`,
  `pkg/models/finding.go`).
