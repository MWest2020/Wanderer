# Change: kleuren-voor-iedereen

## Why

Mark, 2026-09-26: "kleuren mogen contrasterender. voor iemand als ik is er te
weinig verschil."

Measured, not eyeballed (`meting-kleuren.py` in this change: CIEDE2000 between
every pair of verdict colours, with normal vision and with protanopia,
deuteranopia and tritanopia simulated per Machado et al. 2009, severity 1.0):

| palette | normal | protan | deutan | tritan |
|---|---|---|---|---|
| today (`#146c43`, `#45700d`, `#b02a37`, `#495057`) | 13.8 | 9.2 | **4.2** | 7.9 |
| proposed (`#004c8c`, `#3d8fd1`, `#c85200`, `#767676`) | 22.8 | 23.1 | 23.0 | 22.6 |

(smallest ΔE of any pair; below ~10 two colours read as the same at a glance.)

Under deuteranopia — the most common colour-vision deficiency — today's
**voldoende and afhankelijk differ by ΔE 4.2**: "fine" and "dependent" are the
same colour. The cause is that all four are equally dark; only the hue
differs, and hue is exactly what goes missing. On top of that the grid cells
are an 8% tint of that colour: practically white.

## What changes

1. **Palette:** soeverein dark blue `#004c8c`, voldoende medium blue
   `#3d8fd1`, afhankelijk dark orange `#c85200`, onbekend grey `#767676`. Two
   blues for the good end (darker = better), orange for dependent, grey for
   unknown: they differ in lightness, not only in hue.
2. **Solid fills** where a verdict is shown as a surface — grid cells, badges,
   legend dots, ring, bars — with the text colour that clears 4.5:1 on that
   fill: white on dark blue, orange and grey; near-black `#1a1a1a` on medium
   blue.
3. **`--nvt` stops being blue.** "Niet van toepassing" is not a verdict; it
   becomes a neutral outline (grey dashed border, muted text) so it cannot be
   mistaken for soeverein or voldoende.
4. **The property becomes a test**, not a comment: a Go test reads the four
   variables from `main.css` and checks the numbers above.
