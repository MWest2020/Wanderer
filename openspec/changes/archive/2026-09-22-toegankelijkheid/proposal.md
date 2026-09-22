# Proposal: leesbaar voor wie het moet lezen

## Why

Gemeten op 2026-09-22 met axe-core in een echte Chromium, over vier
schermen:

| scherm | bevindingen |
| --- | --- |
| `/ui/` (vloot) | 3× contrast |
| antwoord | 6× contrast |
| trends | 49× contrast, 1× lege tabelkop |
| onderbouwing | 61× contrast |

Concreet, van de vlootpagina:

```
<span class="badge answer-badge-nee">nee</span>
  contrast 3.4 — voorgrond #d9534f op #faeaea, 13,6px
<span class="badge answer-badge-ja">ja</span>
  contrast 3.89 — voorgrond #198754 op #e3f1ea, 13,6px
```

WCAG 2.1 AA eist 4,5 voor tekst van die grootte. Het zijn dus precies
de labels die het oordeel dragen die het slechtst leesbaar zijn — en
voor iemand met verminderd kleurzicht dragen ze niets, want kleur is
het enige verschil tussen "ja" en "nee".

Daarnaast scrollen twee schermen horizontaal op een telefoon van 390px
breed: trends en de onderbouwing.

**Waarom dit geen bijzaak is voor dit product.** De kopers zijn
Nederlandse overheidsorganisaties. Die vallen onder het Tijdelijk besluit
digitale toegankelijkheid: WCAG 2.1 AA, met een verplichte
toegankelijkheidsverklaring. Een instrument dat hun soevereiniteit meet
en zelf niet aan die norm voldoet, is bij de eerste demo een lastig
gesprek — en terecht.

## What Changes

- De oordeelkleuren halen 4,5:1 bij hun werkelijke tekstgrootte.
- Kleur is nooit het enige onderscheid: een oordeel draagt ook tekst of
  vorm, zodat het zonder kleur leesbaar blijft.
- De brede tabellen (trends, onderbouwing) passen op 390px, of scrollen
  binnen hun eigen kader in plaats van de hele pagina mee te trekken.
- De lege tabelkop krijgt een naam.
- **axe-core draait in de Playwright-suite** op elk scherm dat we al
  toetsen, en faalt op `serious` en `critical`. Zonder die stap is dit
  een eenmalige schoonmaak en staat het contrast over een maand weer
  fout.

## Scope / Not in scope

**In:** contrast, het niet-alleen-kleur-principe, de brede tabellen op
telefoonbreedte, de lege tabelkop, en axe in de suite.

**Out:** een volledige WCAG-audit (toetsenbordpaden, schermlezerteksten,
focusvolgorde) en een toegankelijkheidsverklaring. Dat is eigen werk
zodra deze basis staat — en het hoort met een mens getoetst, niet alleen
met een tool: axe vindt ongeveer een derde van wat er mis kan zijn.

## Wat dit eerlijk houdt

Een groene axe-run betekent niet "toegankelijk". Het betekent: geen
geautomatiseerd te vinden fouten van dit type. Die zin hoort in de
documentatie te staan, anders wordt een gate een geruststelling.
