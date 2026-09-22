---
status: draft
last_reviewed: 2026-09-22
---

# Accessibility gate

Wanderer's Playwright suite runs [axe-core](https://github.com/dequelabs/axe-core)
against every screen the suite already visits (vloot, antwoord, trends,
onderbouwing, regelpagina) and fails the build on findings of impact
`serious` or `critical`. See `tests/playwright/support/axe.ts`.

## What a green run proves — and what it doesn't

**A green run means "no automated-tooling-detectable errors of this
type." It does not mean "accessible."** axe-core finds roughly a third
of what can be wrong with a page: it catches contrast ratios, missing
form labels, empty table headers and similar structural issues, but it
cannot judge whether a screen reader user can actually complete a task,
whether the keyboard focus order makes sense, or whether alternative
text is meaningful rather than merely present.

The gate is a floor, not a certificate. A full WCAG 2.1 AA review —
keyboard paths, screen-reader transcripts, focus order — is separate
work that has to include a human, and a formal
*toegankelijkheidsverklaring* (accessibility statement) is a further
step still, required for Dutch public-sector buyers under the
*Tijdelijk besluit digitale toegankelijkheid*.

## What it currently checks

- Colour contrast on every element the suite's pages render, including
  the verdict badges (`ja`/`nee`/`onbekend`/`n.v.t.`,
  `soeverein`/`voldoende`/`afhankelijk`/`onbekend`) at their real
  rendered size (13.6px, normal weight) — WCAG 2.1 AA requires 4.5:1
  for text that size.
- That colour is never the only distinction between two verdicts: each
  badge also carries its own word.
- Only `serious`/`critical` impact findings fail the build;
  `moderate`/`minor` findings (axe's own "should fix", not "must fix")
  are not gated on.

## Known gaps

- The suite does not visit `/demo` (it isn't mounted by any
  `playwright.config.ts` project), so that screen has no axe coverage.
- `/ui/orgs/<slug>/fleet` renders its own "vloot" heading but is a
  different screen from `/ui/` (also called "vloot" in the task that
  introduced this gate); only `/ui/` is covered.
