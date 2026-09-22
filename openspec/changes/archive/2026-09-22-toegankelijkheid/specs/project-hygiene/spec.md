## ADDED Requirements

### Requirement: De suite toetst toegankelijkheid en faalt erop

De Playwright-suite SHALL axe-core draaien op elk scherm dat zij al
toetst, en SHALL falen op bevindingen van niveau `serious` of
`critical`. De documentatie SHALL erbij zeggen dat een groene run niet
"toegankelijk" betekent maar "geen geautomatiseerd te vinden fouten van
dit type" — een geautomatiseerde toets vindt ongeveer een derde van wat
er mis kan zijn.

#### Scenario: Contrast verslechtert

- **GIVEN** een wijziging die een oordeelkleur onder 4,5:1 brengt
- **WHEN** de suite draait
- **THEN** faalt zij, met het element en de gemeten verhouding in de
  melding

#### Scenario: Groen is geen garantie

- **WHEN** de suite groen is
- **THEN** zegt de documentatie wat dat wel en niet betekent
