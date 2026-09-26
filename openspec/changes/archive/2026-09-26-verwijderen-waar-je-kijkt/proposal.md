# Change: verwijderen-waar-je-kijkt

## Why

Mark, 2026-09-26, looking at `schiphol.nk` on the live instance: "het zit er
nog in, maar waarom niet in de ui ook een verwijder knop?"

Measured on a copy of the prod data (v0.11.1):

- The overview (`/ui/`) shows every domain in the grid, but offers no way to
  remove one; only a "vloot beheren" link to another page.
- On that page (`/ui/orgs/{slug}/fleet`) the "Verwijderen" button is the last
  column of a wide table. At 1280 px it is visible; **at 390 px (a phone) it
  sits at x = 776, off screen**, and the table does not scroll on its own.

Two attempts to remove `schiphol.nk` over three days did not reach the
instance. A button a person cannot find is the same as no button.

## What changes

1. The remove action sits **next to the domain name** — in the overview's grid
   and on the fleet page — so it is on screen at any width.
2. It asks once, without JavaScript: `verwijderen` opens a
   `<details>` with "Ja, haal {domein} uit de vloot" and a line that the scans
   stay.
3. After removing, the person lands back where they clicked: the overview or
   the fleet page.
4. The fleet page's table scrolls inside its own frame on a narrow screen, like
   the overview's grid.

Removal stays what it is: `removed_at` is stamped, scans and assessments stay,
adding the domain again restores it.
