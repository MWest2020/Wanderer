## ADDED Requirements

### Requirement: Een domein verwijder je waar je het ziet

Een ingelogde gebruiker SHALL een domein uit de vloot kunnen halen vanaf elke
plek waar het domein in een vlootlijst staat: het overzicht (`/ui/`,
`/ui/orgs/{slug}`) en het vlootscherm. De handeling SHALL naast de domeinnaam
staan, zodat die op elke schermbreedte zichtbaar is, en SHALL één keer om
bevestiging vragen zonder JavaScript. Na het verwijderen SHALL de gebruiker
terugkomen op de pagina waar die klikte; een terugadres dat niet een van die
pagina's is, SHALL genegeerd worden.

#### Scenario: Verwijderen vanaf het overzicht

- **GIVEN** een ingelogde gebruiker op `/ui/` met `schiphol.nk` in het raster
- **WHEN** die bij `schiphol.nk` op verwijderen en daarna op de bevestiging
  klikt
- **THEN** staat die weer op `/ui/`, en staat `schiphol.nk` niet meer in het
  raster

#### Scenario: Zichtbaar op een telefoon

- **GIVEN** het vlootscherm op een scherm van 390 px breed
- **WHEN** de pagina rendert
- **THEN** ligt de verwijderhandeling van elk domein binnen het scherm
- **AND** scrollt de pagina zelf niet horizontaal

#### Scenario: Geen open doorverwijzing

- **GIVEN** een verwijderverzoek met als terugadres `https://elders.example/`
- **WHEN** het verzoek wordt afgehandeld
- **THEN** komt de gebruiker op het vlootscherm, niet op dat adres
