## ADDED Requirements

### Requirement: Een nieuwe stylesheet komt meteen aan

Elke pagina SHALL de stylesheet linken met een versie die uit de inhoud van
die stylesheet volgt, zodat een gewijzigde stylesheet een nieuw adres heeft en
geen cache (browser of edge) de oude kan teruggeven. Een ongewijzigde
stylesheet SHALL zijn adres houden.

#### Scenario: Na een release

- **GIVEN** een release die `main.css` wijzigt
- **WHEN** een pagina rendert
- **THEN** linkt die naar `/ui/static/main.css?v=` gevolgd door de hash van
  de nieuwe inhoud, niet door die van de oude

#### Scenario: Zonder wijziging

- **GIVEN** twee builds met dezelfde `main.css`
- **WHEN** een pagina rendert
- **THEN** is het stylesheet-adres in beide gelijk
