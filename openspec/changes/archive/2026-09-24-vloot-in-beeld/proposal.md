# Change: vloot-in-beeld

## Why

Mark, 2026-09-24, after using the live instance: "Wanderer has the search bar
at the bottom. This is not intuitive for humans. Beyond that, more visuals on
the dashboard, it's still too much Analysis. Remember the roles we set for the
users?"

The roles are ADR-0018's: **de vloot** (`/ui/`) is the Tourist's layer — a
CISO who wants to see at a glance where the fleet stands. **Het domein** is the
Farmer's, **de techniek** the Explorer's.

What `/ui/` shows that Tourist today (rendered from a copy of the prod data,
2026-09-24, 11 domains):

- The fleet score is a small `24/46` badge in a line of text.
- "Per stroom" is a three-column text table.
- "Kost de vloot de meeste punten" prints three rule **rationales** — long
  English paragraphs about CAA records and the CLOUD Act. That is the
  Explorer's material on the Tourist's screen: the "too much analysis".
- Each domain row carries a "zwaarste: …" sentence.
- The scan field sits under the domain list, at the bottom of a page that is
  1,750 px tall; an "Organisations" table with one row closes the page.

Nothing on the page is a picture. Everything must be read.

## What changes

`/ui/` (and `/ui/orgs/{slug}`) keeps the same content, shown so it can be
taken in without reading:

1. **The scan field moves to the top**, as a compact bar under the page
   header. Still not the page's reason to exist (spec: "niet de hoofdzaak"),
   but where a person looks for it.
2. **The fleet score becomes a ring**: soeverein / niet soeverein /
   onbeantwoord as segments, `x/n` in the middle, and the number of domains
   that are not sovereign as a large figure beside it.
3. **Per stroom becomes seven bars**: per flow, how many domains are in the
   EEA, outside it, or unknown, as one stacked bar.
4. **Domains become a grid**: one row per domain, one coloured cell per flow,
   the `x/n` score at the end; worst first. The row links to the domain's
   answer page. The "zwaarste" sentence goes: the grid shows it.
5. **The top 3 speak the Tourist's language**: the flow and the Dutch action
   (`handeling`) with a bar "7 van 11 domeinen", not the English rationale.
   The rationale stays one click away on the rule page (de techniek).
6. **The Organisations table leaves the Tourist layer** when there is only
   one organisation.

## Not in scope

Het domein, de techniek, Trends, the fleet management screen. No change to
scoring, the assessor, or the store.
