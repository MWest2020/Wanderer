# Habitat run 01 — kleuren voor iedereen (tasks 1.1–1.4)

Contract: `openspec/changes/2026-09-26-kleuren-voor-iedereen/` — proposal (the
measured numbers), specs/web-ui/spec.md, and `meting-kleuren.py` (the reference
implementation of the measurement: WCAG contrast, Machado 2009 matrices on
linear RGB, CIEDE2000). Port that measurement to the Go test; do not invent a
different one.

## Decisions already made
1. Palette, exactly: `--soeverein: #004c8c; --voldoende: #3d8fd1;
   --afhankelijk: #c85200; --onbekend: #767676;` Text on fill: `#ffffff` on
   soeverein, afhankelijk, onbekend; `#1a1a1a` on voldoende. Add these as
   variables (e.g. `--soeverein-tekst`) so every rule uses them.
2. Every `.score-*` surface becomes a solid fill with its text variable:
   `.score-*` badges, `.domain-grid .grid-cell.score-*`, `.ring-legend-dot`,
   `.flow-bar-segment`, `.ring-segment` (stroke), `.sov-diagram .node`.
   Remove the `rgba(…, 0.08)` tints for these. Where a verdict colour is used
   as TEXT on white (e.g. a coloured word in a sentence), use a variant that
   clears 4.5:1 on white, or render it as a filled badge instead.
3. `--nvt` usages (`.answer-badge-nvt`, `.answer-row.answer-nvt`,
   `.pill-accountability-nvt`): no verdict colour; white background,
   `1px dashed var(--muted)` border, `var(--muted)` text.
4. Update the palette comment block at the top of `:root` to point at the test
   and at this change instead of stating numbers by hand.
5. Nothing else: no layout change, no template change unless a class is needed
   for the text-on-fill colour.

## Test (1.4)
`internal/ui/palette_test.go` (or similar): read `main.css`, extract the four
verdict variables and their text variables from `:root`, and assert: pairwise
CIEDE2000 ≥ 20 under normal/protan/deutan/tritan; each ≥ 3:1 against `--bg`;
text-on-fill ≥ 4.5:1. Put the old palette through the same function in a
second test case that expects it to FAIL (deutan pair ~4.2), so the test
proves it can fail. Check once by temporarily restoring the old palette in
main.css; say so. CHANGELOG under [Unreleased], `### Changed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-26-kleuren-voor-iedereen --strict` green.
Playwright runs afterwards. Budget $6.
