# Habitat run 02 — vier punten uit het nakijken (tasks 1b.1–1b.4)

Contract: `openspec/changes/2026-09-24-vloot-in-beeld/` (proposal, design,
specs/web-ui/spec.md). Run 01 is gemerged; lees tasks.md sectie 1b voor wat
screenshots op prod-data lieten zien.

## De vier punten, en wat al besloten is
1. **Top-3 nooit een kale regel-ID als er een handeling is.** In
   `internal/ui/top_rules.go`: stroom is optioneel. Stroom + handeling als
   beide er zijn; alleen de handeling als er geen stroom is; de regel-ID
   alleen als er géén handeling is. `handelingForFleet` blijft zoals hij is
   ("elk getroffen domein" — niet terugdraaien naar weglaten).
2. **Ring met zichtbare legenda** naast de ring: soeverein, niet soeverein,
   onbeantwoord — elk met een kleurbolletje in de kleur van zijn segment en
   het aantal. De getallen komen uit dezelfde Go-berekening als de segmenten.
3. **Legenda bij het raster**, één regel boven de tabel: S soeverein ·
   V voldoende · N afhankelijk · ? onbekend (of niet gemeten), in de kleuren
   van de cellen.
4. **Mobiel, 390 px breed:** de stroombalken blijven zichtbaar (label onder de
   balk mag), en het raster staat in een eigen container met
   `overflow-x: auto`, zodat de pagina zelf niet horizontaal scrollt.

## Tests
- Go: 1 (drie gevallen: stroom+handeling, alleen handeling, alleen ID) en 2
  (legenda-aantallen gelijk aan de segmenten).
- Playwright, nieuw: op viewport 390×844 is
  `document.documentElement.scrollWidth <= window.innerWidth`, en elke
  stroombalk heeft een breedte > 0. Op desktop staat de legenda-tekst
  "onbeantwoord" zichtbaar.
- Elke nieuwe test één keer rood gezien met de reparatie eruit; zeg het.

## Scope
Alleen `/ui/` en `/ui/orgs/{slug}` en hun CSS. Geen andere pagina's.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen,
`openspec validate 2026-09-24-vloot-in-beeld --strict` groen. Playwright en
golangci-lint draaien na de run. Budget $7.
