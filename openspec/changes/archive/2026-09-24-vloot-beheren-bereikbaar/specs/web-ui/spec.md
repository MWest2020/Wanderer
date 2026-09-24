## ADDED Requirements

### Requirement: Het vlootbeheer is bereikbaar vanaf het overzicht

Vanaf `/ui/` SHALL het vlootbeheer in één klik bereikbaar zijn: toont het
overzicht de vloot van één organisatie, dan SHALL er een link "vloot
beheren" naar `/ui/orgs/{slug}/fleet` van die organisatie staan. Wie een
domein wil verwijderen, hoeft de URL niet te kennen.

#### Scenario: Eén organisatie

- **GIVEN** een instantie met precies één organisatie
- **WHEN** een ingelogde gebruiker `/ui/` opent
- **THEN** staat er een link "vloot beheren" naar
  `/ui/orgs/{slug}/fleet` van die organisatie

#### Scenario: Meerdere organisaties

- **GIVEN** een instantie met twee organisaties
- **WHEN** een ingelogde gebruiker `/ui/` opent
- **THEN** staat de Organisations-tabel er, en leidt elke organisatie
  naar een dashboard met een eigen link "vloot beheren"
