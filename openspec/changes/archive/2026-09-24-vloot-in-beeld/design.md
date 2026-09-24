# Design: vloot-in-beeld

## Visuals without a chart library

Inline SVG and CSS, rendered by the Go templates. No JavaScript, no CDN, no
new dependency: the UI has none today, the page must render behind a
Cloudflare tunnel and inside a Nextcloud ExApp iframe, and a server-rendered
SVG is testable with the same HTML assertions as the rest of the UI.

- **Ring:** one `<svg>` with `circle` segments via `stroke-dasharray`, sizes
  computed in Go (not in the template), so the arithmetic is unit-tested.
- **Bars:** `div`s with widths as percentages, computed in Go.
- **Grid:** a `table` with one `td` per flow carrying the existing
  `score-*` class; the cell text is a single letter or symbol for screen
  readers and print, the colour for the eye.

## Every picture has its words

Each visual carries the same facts as text: the ring has an `aria-label`
("24 van 46 vragen soeverein, 31 onbeantwoord"), each bar a label with the
counts, each grid cell a `title` with flow and verdict. Two reasons: people
who do not see colour, and tests that can assert facts instead of pixels.

## Colours

The existing verdict palette (`score-soeverein`, `score-voldoende`,
`score-afhankelijk`, `score-onbekend`), in light and dark. Onbekend is grey,
never a colour that reads as "fine" or "bad".

## The top 3 without the rationale

A rule has an English `Description` and `Rationale` but no short Dutch title.
The Tourist line is built from what the Tourist layer already speaks: the
flow the rule belongs to (Hosting, DNS, Certificaat, …) and the rule's
`handeling` with `{domein}` left out. If a rule has no flow or no handeling,
its ID is shown — never the rationale.
